package spclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"spaudit/domain/audit"
	"spaudit/domain/sharepoint"
	"spaudit/logging"

	"github.com/koltyakov/gosip"
	"github.com/koltyakov/gosip/api"
)

// SharePointPaginatedResult represents a paginated response from SharePoint API calls.
// Used for handling large result sets that need to be processed in chunks.
type SharePointPaginatedResult[T any] struct {
	Items         []T         // Current page of items
	HasMorePages  bool        // True if additional pages are available
	TotalCount    int         // Total count if available from SharePoint, -1 if unknown
	NextPageToken interface{} // Opaque token for retrieving the next page
}

// SharePointClient interface abstracts SharePoint REST API operations for audit data collection.
// Provides high-level methods for retrieving site structure, permissions, and sharing information
// while handling authentication, throttling, and API response parsing.
type SharePointClient interface {
	// Site Structure Operations
	GetSiteWeb(ctx context.Context) (*sharepoint.Web, error)
	GetWebLists(ctx context.Context, webID string) ([]*sharepoint.List, error)

	// Permission Operations
	GetSiteRoleDefinitions(ctx context.Context) ([]*sharepoint.RoleDefinition, error)
	GetObjectRoleAssignments(ctx context.Context, target PermissionTarget) ([]*sharepoint.RoleAssignment, []*sharepoint.Principal, error)
	CheckUniquePermissions(ctx context.Context, target PermissionTarget) (bool, error)

	// Sharing Operations
	GetItemSharingInfo(ctx context.Context, itemGUID string) (*sharepoint.SharingInfo, error)

	// Item Resolution Operations
	ResolveFileByGUID(ctx context.Context, itemGUID string) (*sharepoint.Item, error)
	ResolveFolderByGUID(ctx context.Context, itemGUID string) (*sharepoint.Item, error)

	// List Item Batch Operations (for efficient scanning)
	CreateListItemsQuery(ctx context.Context, listID string, batchSize int) *api.Items
	ConvertItemResponse(ctx context.Context, itemResp interface{}, listID string) (*sharepoint.Item, error)
	ConvertItemWithSensitivityLabel(ctx context.Context, itemResp interface{}, listID string, siteID int64) (*sharepoint.Item, *sharepoint.ItemSensitivityLabel, error)

	// List Metadata Operations
	CheckListVisibility(listID string) bool // Returns true if list is hidden from normal interfaces
}

// JSON response structures and helpers.

type parentListJSON struct {
	Id         string `json:"Id"`
	Title      string `json:"Title"`
	RootFolder struct {
		ServerRelativeUrl string `json:"ServerRelativeUrl"`
	} `json:"RootFolder"`
}

type itemJSON struct {
	FileSystemObjectType int            `json:"FileSystemObjectType"`
	Id                   int            `json:"Id"`   // lowercase "Id"
	IDAlt                int            `json:"ID"`   // sometimes also present as "ID"
	GUID                 string         `json:"GUID"` // List Item GUID
	FileLeafRef          *string        `json:"FileLeafRef"`
	FileRef              *string        `json:"FileRef"`
	ParentList           parentListJSON `json:"ParentList"`
}

// Verbose OData envelope: {"d": {...}}
type verboseEnvelope struct {
	D itemJSON `json:"d"`
}

// decodeItemJSON auto-detects verbose vs minimal JSON and normalizes Id.
func decodeItemJSON(b []byte) (itemJSON, error) {
	var env verboseEnvelope
	if err := json.Unmarshal(b, &env); err == nil && (env.D.Id != 0 || env.D.ParentList.Id != "") {
		if env.D.Id == 0 && env.D.IDAlt != 0 {
			env.D.Id = env.D.IDAlt
		}
		return env.D, nil
	}
	var m itemJSON
	if err := json.Unmarshal(b, &m); err != nil {
		return itemJSON{}, err
	}
	if m.Id == 0 && m.IDAlt != 0 {
		m.Id = m.IDAlt
	}
	return m, nil
}

func ptrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// PermissionTarget represents a SharePoint object that can have role assignments (permissions).
// Used to specify which object to query or modify permissions for.
type PermissionTarget struct {
	ObjectType string // SharePoint object type: "web", "list", or "item"
	ObjectID   string // Primary identifier: web ID, list ID, or item GUID
	ListItemID int    // Required for items: SharePoint list item integer ID
}

// SharePointClientImpl wraps the Gosip API client to provide SharePoint operations.
// Handles authentication, request configuration, and response parsing for audit operations.
type SharePointClientImpl struct {
	gosipAPI            *api.SP                // Primary Gosip API client for SharePoint operations
	authClient          *gosip.SPClient        // Authentication client for direct HTTP calls
	defaultConfig       *api.RequestConfig     // Default request configuration (timeout, headers, etc.)
	cachedWebID         string                 // Cached web ID to avoid repeated API calls
	cachedWebURL        string                 // Cached web URL for constructing absolute URLs
	listVisibilityCache map[string]bool        // Cache of listID -> isHidden to avoid repeated queries
	logger              *logging.Logger        // Component logger for debugging and monitoring
	parameters          *audit.AuditParameters // Audit parameters for batch sizes, timeouts, etc.
}

// NewSharePointClient creates a new SharePoint client implementation with authentication and parameters.
// The Gosip API client handles most operations, while the auth client is used for
// direct HTTP calls to APIs not covered by Gosip (like sharing APIs).
func NewSharePointClient(gosipAPI *api.SP, authClient *gosip.SPClient, parameters *audit.AuditParameters) SharePointClient {
	if parameters == nil {
		parameters = audit.DefaultParameters()
	}

	return &SharePointClientImpl{
		gosipAPI:      gosipAPI,
		authClient:    authClient,
		defaultConfig: &api.RequestConfig{
			// Default configuration that can be extended with timeouts, headers, etc.
		},
		listVisibilityCache: make(map[string]bool),
		logger:              logging.Default().WithComponent("sharepoint_client"),
		parameters:          parameters,
	}
}

// createRequestConfig creates a RequestConfig with the provided context, inheriting default configuration.
// This ensures all requests have proper context for cancellation and timeouts.
func (c *SharePointClientImpl) createRequestConfig(ctx context.Context) *api.RequestConfig {
	config := *c.defaultConfig // Copy default config
	config.Context = ctx       // Override with per-request context
	return &config
}

// GetSiteWeb retrieves web (site) information including basic metadata and permissions.
// This is typically the first call made during site auditing to establish the site context.
func (c *SharePointClientImpl) GetSiteWeb(ctx context.Context) (*sharepoint.Web, error) {
	sp := c.gosipAPI.Conf(c.createRequestConfig(ctx))
	res, err := sp.Web().Select(WebFields).Get()
	if err != nil {
		return nil, fmt.Errorf("get web: %w", err)
	}

	var webData struct {
		Id          string
		Title       string
		Url         string
		WebTemplate string
	}
	if err := json.Unmarshal(res.Normalized(), &webData); err != nil {
		return nil, fmt.Errorf("decode web: %w", err)
	}

	// Cache web info to avoid repeated API calls
	c.cachedWebID = webData.Id
	c.cachedWebURL = webData.Url

	hasUnique, err := c.CheckUniquePermissions(ctx, PermissionTarget{ObjectType: sharepoint.ObjectTypeWeb})
	if err != nil {
		c.logger.Debug("Failed to check web unique assignments", "error", err.Error())
		hasUnique = false
	}

	return &sharepoint.Web{
		ID:        webData.Id,
		URL:       webData.Url,
		Title:     webData.Title,
		Template:  webData.WebTemplate,
		HasUnique: hasUnique,
	}, nil
}

// GetWebLists retrieves all lists for a web, including metadata and permission inheritance info.
// This provides the foundation for list-level auditing by discovering all available lists.
func (c *SharePointClientImpl) GetWebLists(ctx context.Context, webID string) ([]*sharepoint.List, error) {
	sp := c.gosipAPI.Conf(c.createRequestConfig(ctx))
	res, err := sp.Web().Lists().Select(ListFields).Expand(`RootFolder`).Get()
	if err != nil {
		return nil, fmt.Errorf("get lists: %w", err)
	}

	var listsData []struct {
		Id           string
		Title        string
		Hidden       bool
		ItemCount    int
		BaseTemplate int
		RootFolder   struct{ ServerRelativeUrl string }
	}
	if err := json.Unmarshal(res.Normalized(), &listsData); err != nil {
		return nil, fmt.Errorf("decode lists: %w", err)
	}

	lists := make([]*sharepoint.List, 0, len(listsData))
	for _, l := range listsData {
		listURL := joinURL(c.cachedWebURL, l.RootFolder.ServerRelativeUrl)

		hasUnique, err := c.CheckUniquePermissions(ctx, PermissionTarget{ObjectType: sharepoint.ObjectTypeList, ObjectID: l.Id})
		if err != nil {
			c.logger.Debug("Failed to check list unique assignments", "list_title", l.Title, "error", err.Error())
			hasUnique = false
		}

		list := &sharepoint.List{
			ID:           l.Id,
			WebID:        webID,
			Title:        l.Title,
			URL:          listURL,
			BaseTemplate: l.BaseTemplate,
			ItemCount:    l.ItemCount,
			HasUnique:    hasUnique,
		}

		// Cache visibility status to avoid repeated queries
		c.listVisibilityCache[l.Id] = l.Hidden

		lists = append(lists, list)
	}

	return lists, nil
}

// CreateListItemsQuery creates a Gosip query object for efficient pagination of list items.
// Returns an *api.Items query that can be used with GetPaged() for continuous iteration.
// The query selects essential metadata and supports both files and folders.
//
// Parameters:
//   - listID: SharePoint list GUID
//   - batchSize: Number of items per page (1-5000, values outside range are clamped)
//
// Usage:
//
//	query := client.CreateListItemsQuery(ctx, listID, 1000)
//	// Pass to walkListItems() or use directly with GetPaged()
func (c *SharePointClientImpl) CreateListItemsQuery(ctx context.Context, listID string, batchSize int) *api.Items {
	// Use parameters-based batch size clamping
	if batchSize <= 0 {
		batchSize = c.parameters.GetEffectiveBatchSize()
	}

	// Clamp batch size to SharePoint API limits using default constraints
	constraints := audit.DefaultApiConstraints()
	if batchSize < constraints.MinBatchSize {
		batchSize = constraints.MinBatchSize
	} else if batchSize > constraints.MaxBatchSize {
		batchSize = constraints.MaxBatchSize
	}

	sp := c.gosipAPI.Conf(c.createRequestConfig(ctx))
	return sp.Web().Lists().GetByID(listID).Items().
		Select(ItemFields).
		Expand("File,Folder,File/Properties").
		Top(batchSize)
}

// SharePoint FileSystemObjectType constants
const (
	SharePointFile   = 0 // File object
	SharePointFolder = 1 // Folder object
)

// SharePoint OData field selectors for consistent API queries
const (
	WebFields  = `Id,Title,Url,WebTemplate`
	ListFields = `
		Id,Title,Hidden,ItemCount,BaseTemplate,
		RootFolder/ServerRelativeUrl
	`
	ItemFields           = `Id,GUID,FileSystemObjectType,File/ServerRelativeUrl,Folder/ServerRelativeUrl,FileLeafRef,Title,FileRef`
	RoleAssignmentFields = `
		RoleAssignments/Member/Id,
		RoleAssignments/Member/Title,
		RoleAssignments/Member/LoginName,
		RoleAssignments/Member/PrincipalType,
		RoleAssignments/Member/Email,
		RoleAssignments/RoleDefinitionBindings/Id,
		RoleAssignments/RoleDefinitionBindings/Name,
		RoleAssignments/RoleDefinitionBindings/Description
	`
	FileFields = `
		UniqueId,Name,ServerRelativeUrl,Length,TimeCreated,TimeLastModified,
		ListItemAllFields/Id,ListItemAllFields/GUID
	`
	FolderFields = `
		UniqueId,Name,ServerRelativeUrl,ItemCount,TimeCreated,TimeLastModified,
		ListItemAllFields/Id,ListItemAllFields/GUID
	`
)

// ConvertItemResponse processes a single SharePoint list item response from Gosip pagination.
// Converts api.ItemResp (raw JSON bytes) into a domain sharepoint.Item with metadata and permissions.
//
// Parameters:
//   - itemResp: Must be api.ItemResp from Gosip pagination (implements Normalized() method)
//   - listID: SharePoint list GUID for the item
//
// Returns:
//   - Fully populated Item with URL, type classification, and unique permissions status
//   - Error if the response cannot be processed or contains invalid data
//
// Usage:
//
//	// In pagination callback:
//	item, err := client.ConvertItemResponse(ctx, itemResp, listID)
func (c *SharePointClientImpl) ConvertItemResponse(ctx context.Context, itemResp interface{}, listID string) (*sharepoint.Item, error) {
	// itemResp should be api.ItemResp (which is []byte with generated Normalized() method)
	// Try to cast to the actual gosip ItemResp type
	if ir, ok := itemResp.(api.ItemResp); ok {
		// Use the generated Normalized() method directly
		normalizedData := ir.Normalized()

		// Parse the JSON item data using the enhanced API response structure
		var it ListItemApiResponse

		if err := json.Unmarshal(normalizedData, &it); err != nil {
			return nil, fmt.Errorf("failed to unmarshal item: %w", err)
		}

		var (
			isFile   bool
			isFolder bool
			itemURL  string
		)

		switch it.FileSystemObjectType {
		case SharePointFile:
			isFile = true
			if it.File != nil {
				itemURL = joinURL(c.cachedWebURL, it.File.ServerRelativeUrl)
			}
		case SharePointFolder:
			isFolder = true
			if it.Folder != nil {
				itemURL = joinURL(c.cachedWebURL, it.Folder.ServerRelativeUrl)
			}
		}

		name := it.FileLeafRef
		if name == "" && it.Title != "" {
			name = it.Title // Fallback to Title if FileLeafRef is empty
		}

		// Check for unique permissions
		hasUnique, err := c.CheckUniquePermissions(ctx, PermissionTarget{ObjectType: sharepoint.ObjectTypeItem, ObjectID: listID, ListItemID: it.ID})
		if err != nil {
			c.logger.Debug("Failed to check item unique assignments", "item_id", it.ID, "error", err.Error())
			hasUnique = false
		}

		return &sharepoint.Item{
			GUID:         it.GUID,
			ListItemGUID: it.GUID,
			ListID:       listID,
			ID:           it.ID,
			URL:          itemURL,
			Name:         name,
			IsFile:       isFile,
			IsFolder:     isFolder,
			HasUnique:    hasUnique,
		}, nil
	}

	return nil, fmt.Errorf("itemResp is not api.ItemResp type, got: %T", itemResp)
}

// ConvertItemWithSensitivityLabel converts a SharePoint item response to both domain Item and ItemSensitivityLabel in a single parse.
// This is more efficient than calling ConvertItemResponse and ExtractItemSensitivityLabel separately.
func (c *SharePointClientImpl) ConvertItemWithSensitivityLabel(ctx context.Context, itemResp interface{}, listID string, siteID int64) (*sharepoint.Item, *sharepoint.ItemSensitivityLabel, error) {
	// itemResp should be api.ItemResp (which is []byte with generated Normalized() method)
	if ir, ok := itemResp.(api.ItemResp); ok {
		// Use the generated Normalized() method directly
		normalizedData := ir.Normalized()

		// Parse the JSON item data using the enhanced API response structure
		var it ListItemApiResponse
		if err := json.Unmarshal(normalizedData, &it); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal item: %w", err)
		}

		// Validate site ID consistency
		if siteID <= 0 {
			return nil, nil, fmt.Errorf("invalid site ID: %d", siteID)
		}

		// Log item processing for debugging if needed
		if it.File != nil && it.File.Properties != nil {
			c.logger.Debug("Processing file with properties", "item_guid", it.GUID, "item_title", it.Title, "item_id", it.ID)
		}

		// Log sensitivity label detection with proper structured logging
		var sensitivityLabel *sharepoint.ItemSensitivityLabel
		if it.File != nil && it.File.Properties != nil {
			props := it.File.Properties
			if props.IPLabelDisplayName != "" || props.IPLabelOwnerEmail != "" || props.MSIPLabelSetDate != "" {
				c.logger.Debug("Sensitivity label detected on file",
					"item_guid", it.GUID,
					"item_title", it.Title,
					"label_display_name", props.IPLabelDisplayName,
					"label_owner_email", props.IPLabelOwnerEmail,
					"assignment_method", props.IPLabelAssignmentMethod,
					"msip_set_date", props.MSIPLabelSetDate,
					"additional_properties_count", len(props.OtherProperties))

				// Extract sensitivity label using the mapping function
				sensitivityLabel = props.MapToSensitivityLabel(siteID, it.GUID)
			}
		}

		var (
			isFile   bool
			isFolder bool
			itemURL  string
		)

		switch it.FileSystemObjectType {
		case SharePointFile:
			isFile = true
			if it.File != nil {
				itemURL = joinURL(c.cachedWebURL, it.File.ServerRelativeUrl)
			}
		case SharePointFolder:
			isFolder = true
			if it.Folder != nil {
				itemURL = joinURL(c.cachedWebURL, it.Folder.ServerRelativeUrl)
			}
		}

		name := it.FileLeafRef
		if name == "" && it.Title != "" {
			name = it.Title // Fallback to Title if FileLeafRef is empty
		}

		// Check for unique permissions
		hasUnique, err := c.CheckUniquePermissions(ctx, PermissionTarget{ObjectType: sharepoint.ObjectTypeItem, ObjectID: listID, ListItemID: it.ID})
		if err != nil {
			c.logger.Debug("Failed to check item unique assignments", "item_id", it.ID, "error", err.Error())
			hasUnique = false
		}

		item := &sharepoint.Item{
			GUID:         it.GUID,
			ListItemGUID: it.GUID,
			ListID:       listID,
			ID:           it.ID,
			URL:          itemURL,
			Name:         name,
			IsFile:       isFile,
			IsFolder:     isFolder,
			HasUnique:    hasUnique,
		}

		return item, sensitivityLabel, nil
	}

	return nil, nil, fmt.Errorf("itemResp is not api.ItemResp type, got: %T", itemResp)
}

// GetSiteRoleDefinitions retrieves all role definitions (permission levels) for the web.
// Role definitions define what actions users can perform (e.g., "Full Control", "Read", "Contribute").
// These are cached and reused throughout the audit for performance.
func (c *SharePointClientImpl) GetSiteRoleDefinitions(ctx context.Context) ([]*sharepoint.RoleDefinition, error) {
	sp := c.gosipAPI.Conf(c.createRequestConfig(ctx))
	roleDefs, err := sp.Web().RoleDefinitions().Get()
	if err != nil {
		return nil, fmt.Errorf("get role definitions: %w", err)
	}

	definitions := make([]*sharepoint.RoleDefinition, 0, len(roleDefs))
	for _, rd := range roleDefs {
		definitions = append(definitions, &sharepoint.RoleDefinition{
			ID:          int64(rd.ID),
			Name:        rd.Name,
			Description: rd.Description,
		})
	}

	return definitions, nil
}

// GetObjectRoleAssignments retrieves role assignments (permissions) for a specific SharePoint object.
// Returns both the role assignments and the principals (users/groups) involved.
// This is used to discover who has access to webs, lists, and individual items.
func (c *SharePointClientImpl) GetObjectRoleAssignments(ctx context.Context, target PermissionTarget) ([]*sharepoint.RoleAssignment, []*sharepoint.Principal, error) {
	sp := c.gosipAPI.Conf(c.createRequestConfig(ctx))
	var normalizedData []byte

	switch target.ObjectType {
	case sharepoint.ObjectTypeWeb:
		webRes, webErr := sp.Web().
			Select(RoleAssignmentFields).
			Expand(`
				RoleAssignments,
				RoleAssignments/Member,
				RoleAssignments/RoleDefinitionBindings
			`).
			Conf(c.createRequestConfig(ctx)).
			Get()
		if webErr != nil {
			return nil, nil, fmt.Errorf("get web role assignments: %w", webErr)
		}
		normalizedData = webRes.Normalized()

	case sharepoint.ObjectTypeList:
		listRes, listErr := sp.Web().Lists().GetByID(target.ObjectID).
			Select(RoleAssignmentFields).
			Expand(`
				RoleAssignments,
				RoleAssignments/Member,
				RoleAssignments/RoleDefinitionBindings
			`).
			Conf(c.createRequestConfig(ctx)).
			Get()
		if listErr != nil {
			return nil, nil, fmt.Errorf("get list role assignments: %w", listErr)
		}
		normalizedData = listRes.Normalized()

	case sharepoint.ObjectTypeItem:
		itemRes, itemErr := sp.Web().Lists().GetByID(target.ObjectID).Items().GetByID(target.ListItemID).
			Select(RoleAssignmentFields).
			Expand(`
				RoleAssignments,
				RoleAssignments/Member,
				RoleAssignments/RoleDefinitionBindings
			`).
			Conf(c.createRequestConfig(ctx)).
			Get()
		if itemErr != nil {
			return nil, nil, fmt.Errorf("get item role assignments: %w", itemErr)
		}
		normalizedData = itemRes.Normalized()

	default:
		return nil, nil, fmt.Errorf("unknown target type: %s", target.ObjectType)
	}

	return c.parseRoleAssignments(target.ObjectType, target.ObjectID, normalizedData)
}

// CheckUniquePermissions checks if a SharePoint object has unique role assignments.
// This determines whether the object inherits permissions from its parent or has custom permissions.
// Returns true if the object has unique (non-inherited) permissions, false if inherited.
// This is a key optimization - items without unique permissions don't need individual permission queries.
func (c *SharePointClientImpl) CheckUniquePermissions(ctx context.Context, target PermissionTarget) (bool, error) {
	sp := c.gosipAPI.Conf(c.createRequestConfig(ctx))
	switch target.ObjectType {
	case sharepoint.ObjectTypeWeb:
		return sp.Web().Roles().HasUniqueAssignments()
	case sharepoint.ObjectTypeList:
		return sp.Web().Lists().GetByID(target.ObjectID).Roles().HasUniqueAssignments()
	case sharepoint.ObjectTypeItem:
		return sp.Web().Lists().GetByID(target.ObjectID).Items().GetByID(target.ListItemID).Roles().HasUniqueAssignments()
	default:
		return false, fmt.Errorf("unknown target type: %s", target.ObjectType)
	}
}

// GetItemSharingInfo retrieves sharing information for an item using SharePoint's sharing API.
// This provides detailed information about sharing links, permissions, and access settings.
// Returns empty sharing info if the sharing API is unavailable to avoid breaking the audit.
func (c *SharePointClientImpl) GetItemSharingInfo(ctx context.Context, itemGUID string) (*sharepoint.SharingInfo, error) {
	// Check if we have the auth client needed for HTTP calls
	if c.authClient == nil {
		c.logger.Warn("No auth client available for sharing API", "item_guid", itemGUID)
		return &sharepoint.SharingInfo{
			ItemUniqueID: itemGUID,
			Links:        []*sharepoint.SharingLink{},
		}, nil
	}

	// Use GOSIP HTTPClient pattern to call SharePoint sharing API
	spClient := api.NewHTTPClient(c.authClient)

	// Get the site URL - we need the full site URL for the endpoint
	siteURL := c.cachedWebURL
	if siteURL == "" {
		// Get the web URL if not cached
		sp := c.gosipAPI.Conf(c.createRequestConfig(ctx))
		webRes, err := sp.Web().Select("Url").Get()
		if err != nil {
			return nil, fmt.Errorf("get web URL: %w", err)
		}
		var webData struct {
			Url string `json:"Url"`
		}
		if err := json.Unmarshal(webRes.Normalized(), &webData); err != nil {
			return nil, fmt.Errorf("decode web URL: %w", err)
		}
		siteURL = webData.Url
		c.cachedWebURL = siteURL // Cache it
	}

	// Construct the SharePoint sharing API endpoint
	endpoint := fmt.Sprintf(
		"%s/_api/web/GetFileById(guid'%s')/ListItemAllFields/GetSharingInformation?$expand=permissionsInformation,pickerSettings",
		siteURL, itemGUID,
	)

	// Make the API call using POST with empty body (SharePoint sharing API pattern)
	data, err := spClient.Post(endpoint, bytes.NewBufferString("{}"), &api.RequestConfig{Context: ctx})
	if err != nil {
		c.logger.Warn("Failed to get sharing info", "item_guid", itemGUID, "error", err.Error())
		// Return empty sharing info instead of failing to avoid breaking the audit
		return &sharepoint.SharingInfo{
			ItemUniqueID: itemGUID,
			Links:        []*sharepoint.SharingLink{},
		}, nil
	}

	// Parse the SharePoint sharing response using existing decoder
	sharingApiResponse, err := DecodeSharingApiResponse(data)
	if err != nil {
		c.logger.Warn("Failed to decode sharing data", "item_guid", itemGUID, "error", err.Error())
		return &sharepoint.SharingInfo{
			ItemUniqueID: itemGUID,
			Links:        []*sharepoint.SharingLink{},
		}, nil
	}

	// Convert SharingApiResponse to sharepoint.SharingInfo
	return c.mapSharingApiResponseToSharingInfo(sharingApiResponse), nil
}

// ResolveFileByGUID retrieves file details by GUID using SharePoint's File API.
// This resolves a file's UniqueId to its full metadata including list context and URLs.
// Used primarily for resolving sharing link targets to their source items.
func (c *SharePointClientImpl) ResolveFileByGUID(ctx context.Context, itemGUID string) (*sharepoint.Item, error) {
	if c.authClient == nil {
		return nil, fmt.Errorf("no auth client available for GetFileByGUID %s", itemGUID)
	}

	spClient := api.NewHTTPClient(c.authClient)
	siteURL := c.authClient.AuthCnfg.GetSiteURL()

	endpoint := fmt.Sprintf(
		"%s/_api/web/GetFileById(guid'%s')/ListItemAllFields"+
			"?$select=Id,GUID,FileSystemObjectType,FileLeafRef,FileRef,"+
			"ParentList/Id,ParentList/Title,ParentList/RootFolder/ServerRelativeUrl"+
			"&$expand=ParentList,ParentList/RootFolder",
		siteURL, itemGUID,
	)

	data, err := spClient.Get(endpoint, &api.RequestConfig{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("get file by GUID %s: %w", itemGUID, err)
	}

	// Optional pretty log (ignore indent errors)
	if len(data) > 0 {
		var out bytes.Buffer
		_ = json.Indent(&out, data, "", "  ")
		if out.Len() > 0 {
			c.logger.Debug("Raw JSON response from GetFileByGUID", "item_guid", itemGUID)
		} else {
			c.logger.Debug("Raw response from GetFileByGUID", "item_guid", itemGUID)
		}
	}

	itm, err := decodeItemJSON(data)
	if err != nil {
		return nil, fmt.Errorf("decode item JSON: %w", err)
	}
	// Guard against poisoning DB with (“”, 0)
	if itm.ParentList.Id == "" || itm.Id == 0 {
		return nil, fmt.Errorf("missing ParentList.Id/Id (got list_id=%q, item_id=%d)", itm.ParentList.Id, itm.Id)
	}

	name := ptrOrEmpty(itm.FileLeafRef)
	absoluteURL := strings.TrimRight(siteURL, "/") + ptrOrEmpty(itm.FileRef)

	// itemGUID is the File UniqueId (used in sharing links); itm.GUID is the List Item GUID.
	// Store both GUIDs to properly handle sharing link resolution
	return &sharepoint.Item{
		GUID:         itemGUID,          // File UniqueId (used in sharing links)
		ListItemGUID: itm.GUID,          // List Item GUID
		ListID:       itm.ParentList.Id, // List GUID
		ID:           itm.Id,            // List item integer ID
		Name:         name,
		IsFile:       true,
		IsFolder:     false,
		URL:          absoluteURL,
	}, nil
}

// ResolveFolderByGUID retrieves folder details by GUID using SharePoint's Folder API.
// This resolves a folder's UniqueId to its full metadata including list context and URLs.
// Used primarily for resolving sharing link targets to their source items.
func (c *SharePointClientImpl) ResolveFolderByGUID(ctx context.Context, itemGUID string) (*sharepoint.Item, error) {
	if c.authClient == nil {
		return nil, fmt.Errorf("no auth client available for GetFolderByGUID %s", itemGUID)
	}
	spClient := api.NewHTTPClient(c.authClient)

	siteURL := c.authClient.AuthCnfg.GetSiteURL()
	endpoint := fmt.Sprintf(
		"%s/_api/web/GetFolderById(guid'%s')/ListItemAllFields"+
			"?$select=Id,GUID,FileSystemObjectType,FileLeafRef,FileRef,"+
			"ParentList/Id,ParentList/Title,ParentList/RootFolder/ServerRelativeUrl"+
			"&$expand=ParentList,ParentList/RootFolder",
		siteURL, itemGUID,
	)

	data, err := spClient.Get(endpoint, &api.RequestConfig{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("get folder by GUID %s: %w", itemGUID, err)
	}

	// Optional pretty log
	if len(data) > 0 {
		var out bytes.Buffer
		_ = json.Indent(&out, data, "", "  ")
		if out.Len() > 0 {
			c.logger.Debug("Raw JSON response from GetFolderByGUID", "item_guid", itemGUID)
		} else {
			c.logger.Debug("Raw response from GetFolderByGUID", "item_guid", itemGUID)
		}
	}

	itm, err := decodeItemJSON(data)
	if err != nil {
		return nil, fmt.Errorf("decode folder JSON: %w", err)
	}
	if itm.ParentList.Id == "" || itm.Id == 0 {
		return nil, fmt.Errorf("missing ParentList.Id/Id (got list_id=%q, item_id=%d)", itm.ParentList.Id, itm.Id)
	}

	name := ptrOrEmpty(itm.FileLeafRef)
	absoluteURL := strings.TrimRight(siteURL, "/") + ptrOrEmpty(itm.FileRef)

	// itemGUID is the Folder UniqueId (used in sharing links); itm.GUID is the List Item GUID.
	// Store both GUIDs to properly handle sharing link resolution
	return &sharepoint.Item{
		GUID:         itemGUID,          // Folder UniqueId (used in sharing links)
		ListItemGUID: itm.GUID,          // List Item GUID
		ListID:       itm.ParentList.Id, // List GUID
		ID:           itm.Id,            // List item integer ID
		Name:         name,
		IsFile:       false,
		IsFolder:     true,
		URL:          absoluteURL,
	}, nil
}

// CheckListVisibility checks if a list is marked as hidden in SharePoint.
// Uses cached visibility information from list enumeration to avoid additional API calls.
// Returns true if the list is hidden from normal user interfaces.
func (c *SharePointClientImpl) CheckListVisibility(listID string) bool {
	hidden, exists := c.listVisibilityCache[listID]
	return exists && hidden
}

// parseRoleAssignments converts SharePoint role assignment JSON to domain models.
// Handles both wrapped and direct JSON response formats from SharePoint API.
// Returns role assignments and principals separately for efficient storage.
func (c *SharePointClientImpl) parseRoleAssignments(objectType, objectKey string, data []byte) ([]*sharepoint.RoleAssignment, []*sharepoint.Principal, error) {
	type assignmentPayload struct {
		RoleAssignments []*struct {
			Member *struct {
				Id            int
				Title         string
				LoginName     string
				PrincipalType int
				Email         string
			}
			RoleDefinitionBindings []*struct {
				Id          int
				Name        string
				Description string
			}
		}
	}

	var payload assignmentPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		// Fallback: array directly
		var rs []*struct {
			Member *struct {
				Id            int
				Title         string
				LoginName     string
				PrincipalType int
				Email         string
			}
			RoleDefinitionBindings []*struct {
				Id          int
				Name        string
				Description string
			}
		}
		if err2 := json.Unmarshal(data, &rs); err2 != nil {
			return nil, nil, fmt.Errorf("decode role assignments: %v / %v", err, err2)
		}
		payload.RoleAssignments = rs
	}

	var assignments []*sharepoint.RoleAssignment
	principalMap := make(map[int64]*sharepoint.Principal)

	for _, ra := range payload.RoleAssignments {
		if ra == nil || ra.Member == nil {
			continue
		}

		// Extract principal information (like original code)
		principalID := int64(ra.Member.Id)
		if _, exists := principalMap[principalID]; !exists {
			principalMap[principalID] = &sharepoint.Principal{
				ID:            principalID,
				PrincipalType: int64(ra.Member.PrincipalType),
				Title:         strings.TrimSpace(ra.Member.Title),
				LoginName:     ra.Member.LoginName,
				Email:         ra.Member.Email,
			}
		}

		// Create role assignments
		for _, rd := range ra.RoleDefinitionBindings {
			if rd == nil {
				continue
			}

			assignments = append(assignments, &sharepoint.RoleAssignment{
				ObjectType:  objectType,
				ObjectKey:   objectKey,
				PrincipalID: principalID,
				RoleDefID:   int64(rd.Id),
				Inherited:   false, // these calls fetch explicit assignments
			})
		}
	}

	// Convert principal map to slice
	principals := make([]*sharepoint.Principal, 0, len(principalMap))
	for _, principal := range principalMap {
		principals = append(principals, principal)
	}

	return assignments, principals, nil
}

// mapSharingApiResponseToSharingInfo converts SharingApiResponse to sharepoint.SharingInfo.
// Transforms SharePoint's sharing API response format to our domain model.
// Handles timestamp parsing, principal conversion, and link member enumeration.
func (c *SharePointClientImpl) mapSharingApiResponseToSharingInfo(sl SharingApiResponse) *sharepoint.SharingInfo {
	sharingInfo := &sharepoint.SharingInfo{
		DisplayName:                sl.DisplayName,
		ItemUniqueID:               sl.ItemUniqueID,
		WebURL:                     sl.WebURL,
		DirectURL:                  sl.DirectURL,
		FileExtension:              sl.FileExtension,
		HasUniquePermissions:       sl.HasUniquePermissions,
		DefaultLinkKind:            sl.DefaultLinkKind,
		DefaultShareLinkPermission: sl.DefaultShareLinkPermission,
		DefaultShareLinkScope:      sl.DefaultShareLinkScope,
		Links:                      make([]*sharepoint.SharingLink, 0, len(sl.PermissionsInformation.Links.Results)),

		// Enhanced governance fields
		TenantID:                               sl.TenantID,
		TenantDisplayName:                      sl.TenantDisplayName,
		SharePointSiteID:                       sl.SiteID,
		AnonymousLinkExpirationRestrictionDays: sl.AnonymousLinkExpirationRestrictionDays,
		AnyoneLinkTrackUsers:                   sl.AnyoneLinkTrackUsers,
		CanAddExternalPrincipal:                sl.CanAddExternalPrincipal,
		CanAddInternalPrincipal:                sl.CanAddInternalPrincipal,
		BlockPeoplePickerAndSharing:            sl.BlockPeoplePickerAndSharing,
		CanRequestAccessForGrantAccess:         sl.CanRequestAccessForGrantAccess,
		SiteIBMode:                             sl.SiteIBMode,
		SiteIBSegmentIDs:                       sl.SiteIBSegmentIDs.Results,
		EnforceIBSegmentFiltering:              sl.EnforceIBSegmentFiltering,
	}

	// Map sensitivity label information
	if sl.SensitivityLabelInformation != nil {
		sharingInfo.SensitivityLabel = &sharepoint.SensitivityLabelInformation{
			ID:                             sl.SensitivityLabelInformation.ID,
			DisplayName:                    sl.SensitivityLabelInformation.DisplayName,
			Color:                          sl.SensitivityLabelInformation.Color,
			Tooltip:                        sl.SensitivityLabelInformation.Tooltip,
			HasIRMProtection:               sl.SensitivityLabelInformation.HasIRMProtection,
			SensitivityLabelProtectionType: sl.SensitivityLabelInformation.SensitivityLabelProtectionType,
		}
	}

	// Map sharing abilities
	if sl.SharingAbilities != nil {
		sharingInfo.SharingAbilities = c.mapSharingAbilities(sl.SharingAbilities)
	}

	// Map recipient limits
	if sl.RecipientLimits != nil {
		sharingInfo.RecipientLimits = c.mapRecipientLimits(sl.RecipientLimits)
	}

	// Convert links with enhanced governance fields
	for _, linkLite := range sl.PermissionsInformation.Links.Results {
		ld := linkLite.LinkDetails

		link := &sharepoint.SharingLink{
			ID:                   ld.ShareId,     // Use ShareID as the unique identifier
			ItemGUID:             "",              // Will be set when we know the ListItem GUID
			FileFolderUniqueID:   sl.ItemUniqueID, // File/Folder UniqueId (used in sharing links)
			ShareID:              ld.ShareId,
			URL:                  c.stringPtrToString(ld.URL),
			LinkKind:             ld.LinkKind,
			Scope:                ld.Scope,
			IsActive:             ld.IsActive,
			IsDefault:            ld.IsDefault,
			IsEditLink:           ld.IsEditLink,
			IsReviewLink:         ld.IsReviewLink,
			BlocksDownload:       ld.BlocksDownload,
			RequiresPassword:     ld.RequiresPassword,
			RestrictedMembership: ld.RestrictedShareMembership,
			IsInherited:          linkLite.IsInherited,
			InheritedFrom:        "", // Not available in LinkLite
			TotalMembersCount:    linkLite.TotalLinkMembersCount,
			ShareToken:           c.stringPtrToString(ld.ShareTokenString),

			// Enhanced governance fields
			HasExternalGuestInvitees:        ld.HasExternalGuestInvitees,
			TrackLinkUsers:                  ld.TrackLinkUsers,
			IsEphemeral:                     ld.IsEphemeral,
			IsUnhealthy:                     ld.IsUnhealthy,
			IsAddressBarLink:                ld.IsAddressBarLink,
			IsCreateOnlyLink:                ld.IsCreateOnlyLink,
			IsFormsLink:                     ld.IsFormsLink,
			IsMainLink:                      ld.IsMainLink,
			IsManageListLink:                ld.IsManageListLink,
			AllowsAnonymousAccess:           ld.AllowsAnonymousAccess,
			Embeddable:                      ld.Embeddable,
			LimitUseToApplication:           ld.LimitUseToApplication,
			RestrictToExistingRelationships: ld.RestrictToExistingRelationships,
			SharingLinkStatus:               &ld.SharingLinkStatus,
		}

		// Convert timestamps
		if ld.Created != "" {
			if t, err := time.Parse(time.RFC3339, ld.Created); err == nil {
				link.CreatedAt = &t
			}
		}
		if ld.LastModified != "" {
			if t, err := time.Parse(time.RFC3339, ld.LastModified); err == nil {
				link.LastModifiedAt = &t
			}
		}
		if ld.Expiration != "" {
			if t, err := time.Parse(time.RFC3339, ld.Expiration); err == nil {
				link.Expiration = &t
			}
		}
		if ld.PasswordLastModified != "" {
			if t, err := time.Parse(time.RFC3339, ld.PasswordLastModified); err == nil {
				link.PasswordLastModified = &t
			}
		}

		// Convert principals
		if ld.CreatedBy != nil {
			link.CreatedBy = &sharepoint.Principal{
				ID:            int64(ld.CreatedBy.ID),
				PrincipalType: int64(ld.CreatedBy.PrincipalType),
				Title:         ld.CreatedBy.Name,
				LoginName:     ld.CreatedBy.LoginName,
				Email:         c.stringPtrToString(ld.CreatedBy.Email),
			}
		}

		if ld.LastModifiedBy != nil {
			link.LastModifiedBy = &sharepoint.Principal{
				ID:            int64(ld.LastModifiedBy.ID),
				PrincipalType: int64(ld.LastModifiedBy.PrincipalType),
				Title:         ld.LastModifiedBy.Name,
				LoginName:     ld.LastModifiedBy.LoginName,
				Email:         c.stringPtrToString(ld.LastModifiedBy.Email),
			}
		}

		if ld.PasswordLastModifiedBy != nil {
			link.PasswordLastModifiedBy = &sharepoint.Principal{
				ID:            int64(ld.PasswordLastModifiedBy.ID),
				PrincipalType: int64(ld.PasswordLastModifiedBy.PrincipalType),
				Title:         ld.PasswordLastModifiedBy.Name,
				LoginName:     ld.PasswordLastModifiedBy.LoginName,
				Email:         c.stringPtrToString(ld.PasswordLastModifiedBy.Email),
			}
		}

		// Convert members
		for _, memberLite := range linkLite.LinkMembers.Results {
			member := &sharepoint.Principal{
				ID:            int64(memberLite.ID),
				PrincipalType: int64(memberLite.PrincipalType),
				Title:         memberLite.Name,
				LoginName:     memberLite.LoginName,
				Email:         c.stringPtrToString(memberLite.Email),
			}
			link.Members = append(link.Members, member)
		}

		sharingInfo.Links = append(sharingInfo.Links, link)
	}

	// Convert principals and site admins (if needed)
	// This could be extended to populate sharingInfo.Principals and sharingInfo.SiteAdmins

	return sharingInfo
}

// stringPtrToString safely converts a string pointer to string, returning empty string for nil.
// Helper function for processing SharePoint API responses that may have null string fields.
func (c *SharePointClientImpl) stringPtrToString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// Helper functions for mapping complex governance structures

func (c *SharePointClientImpl) mapSharingAbilities(sa *SharingAbilitiesApiData) *sharepoint.SharingAbilities {
	abilities := &sharepoint.SharingAbilities{
		CanStopSharing: sa.CanStopSharing,
	}

	if sa.AnonymousLinkAbilities != nil {
		abilities.AnonymousLinkAbilities = c.mapSharingLinkAbilities(sa.AnonymousLinkAbilities)
	}

	if sa.AnyoneLinkAbilities != nil {
		abilities.AnyoneLinkAbilities = c.mapSharingLinkAbilities(sa.AnyoneLinkAbilities)
	}

	if sa.OrganizationLinkAbilities != nil {
		abilities.OrganizationLinkAbilities = c.mapSharingLinkAbilities(sa.OrganizationLinkAbilities)
	}

	if sa.PeopleSharingLinkAbilities != nil {
		abilities.PeopleSharingLinkAbilities = c.mapSharingLinkAbilities(sa.PeopleSharingLinkAbilities)
	}

	if sa.DirectSharingAbilities != nil {
		abilities.DirectSharingAbilities = c.mapDirectSharingAbilities(sa.DirectSharingAbilities)
	}

	return abilities
}

func (c *SharePointClientImpl) mapSharingLinkAbilities(sla *SharingLinkAbilitiesApiData) *sharepoint.SharingLinkAbilities {
	abilities := &sharepoint.SharingLinkAbilities{
		CanAddNewExternalPrincipals:             c.mapSharingAbilityStatus(sla.CanAddNewExternalPrincipals),
		CanDeleteEditLink:                       c.mapSharingAbilityStatus(sla.CanDeleteEditLink),
		CanDeleteManageListLink:                 c.mapSharingAbilityStatus(sla.CanDeleteManageListLink),
		CanDeleteReadLink:                       c.mapSharingAbilityStatus(sla.CanDeleteReadLink),
		CanDeleteRestrictedViewLink:             c.mapSharingAbilityStatus(sla.CanDeleteRestrictedViewLink),
		CanDeleteReviewLink:                     c.mapSharingAbilityStatus(sla.CanDeleteReviewLink),
		CanDeleteSubmitOnlyLink:                 c.mapSharingAbilityStatus(sla.CanDeleteSubmitOnlyLink),
		CanGetEditLink:                          c.mapSharingAbilityStatus(sla.CanGetEditLink),
		CanGetManageListLink:                    c.mapSharingAbilityStatus(sla.CanGetManageListLink),
		CanGetReadLink:                          c.mapSharingAbilityStatus(sla.CanGetReadLink),
		CanGetRestrictedViewLink:                c.mapSharingAbilityStatus(sla.CanGetRestrictedViewLink),
		CanGetReviewLink:                        c.mapSharingAbilityStatus(sla.CanGetReviewLink),
		CanGetSubmitOnlyLink:                    c.mapSharingAbilityStatus(sla.CanGetSubmitOnlyLink),
		CanHaveExternalUsers:                    c.mapSharingAbilityStatus(sla.CanHaveExternalUsers),
		CanManageEditLink:                       c.mapSharingAbilityStatus(sla.CanManageEditLink),
		CanManageManageListLink:                 c.mapSharingAbilityStatus(sla.CanManageManageListLink),
		CanManageReadLink:                       c.mapSharingAbilityStatus(sla.CanManageReadLink),
		CanManageRestrictedViewLink:             c.mapSharingAbilityStatus(sla.CanManageRestrictedViewLink),
		CanManageReviewLink:                     c.mapSharingAbilityStatus(sla.CanManageReviewLink),
		CanManageSubmitOnlyLink:                 c.mapSharingAbilityStatus(sla.CanManageSubmitOnlyLink),
		SupportsRestrictedView:                  c.mapSharingAbilityStatus(sla.SupportsRestrictedView),
		SupportsRestrictToExistingRelationships: c.mapSharingAbilityStatus(sla.SupportsRestrictToExistingRelationships),
	}

	if sla.LinkExpiration != nil {
		abilities.LinkExpiration = &sharepoint.SharingLinkExpirationAbilityStatus{
			Enabled:                 sla.LinkExpiration.Enabled,
			DisabledReason:          sla.LinkExpiration.DisabledReason,
			DefaultExpirationInDays: sla.LinkExpiration.DefaultExpirationInDays,
		}
	}

	if sla.PasswordProtected != nil {
		abilities.PasswordProtected = &sharepoint.SharingLinkPasswordAbilityStatus{
			Enabled:        sla.PasswordProtected.Enabled,
			DisabledReason: sla.PasswordProtected.DisabledReason,
		}
	}

	if sla.SubmitOnlyLinkExpiration != nil {
		abilities.SubmitOnlyLinkExpiration = &sharepoint.SharingLinkExpirationAbilityStatus{
			Enabled:                 sla.SubmitOnlyLinkExpiration.Enabled,
			DisabledReason:          sla.SubmitOnlyLinkExpiration.DisabledReason,
			DefaultExpirationInDays: sla.SubmitOnlyLinkExpiration.DefaultExpirationInDays,
		}
	}

	return abilities
}

func (c *SharePointClientImpl) mapDirectSharingAbilities(dsa *DirectSharingAbilitiesApiData) *sharepoint.DirectSharingAbilities {
	return &sharepoint.DirectSharingAbilities{
		CanAddExternalPrincipal:                           c.mapSharingAbilityStatus(dsa.CanAddExternalPrincipal),
		CanAddInternalPrincipal:                           c.mapSharingAbilityStatus(dsa.CanAddInternalPrincipal),
		CanAddNewExternalPrincipal:                        c.mapSharingAbilityStatus(dsa.CanAddNewExternalPrincipal),
		CanRequestGrantAccess:                             c.mapSharingAbilityStatus(dsa.CanRequestGrantAccess),
		CanRequestGrantAccessForExistingExternalPrincipal: c.mapSharingAbilityStatus(dsa.CanRequestGrantAccessForExistingExternalPrincipal),
		CanRequestGrantAccessForInternalPrincipal:         c.mapSharingAbilityStatus(dsa.CanRequestGrantAccessForInternalPrincipal),
		CanRequestGrantAccessForNewExternalPrincipal:      c.mapSharingAbilityStatus(dsa.CanRequestGrantAccessForNewExternalPrincipal),
		SupportsEditPermission:                            c.mapSharingAbilityStatus(dsa.SupportsEditPermission),
		SupportsManageListPermission:                      c.mapSharingAbilityStatus(dsa.SupportsManageListPermission),
		SupportsReadPermission:                            c.mapSharingAbilityStatus(dsa.SupportsReadPermission),
		SupportsRestrictedViewPermission:                  c.mapSharingAbilityStatus(dsa.SupportsRestrictedViewPermission),
		SupportsReviewPermission:                          c.mapSharingAbilityStatus(dsa.SupportsReviewPermission),
	}
}

func (c *SharePointClientImpl) mapSharingAbilityStatus(sas SharingAbilityStatusApiData) sharepoint.SharingAbilityStatus {
	return sharepoint.SharingAbilityStatus{
		Enabled:        sas.Enabled,
		DisabledReason: sas.DisabledReason,
	}
}

func (c *SharePointClientImpl) mapRecipientLimits(rl *RecipientLimitsApiData) *sharepoint.RecipientLimits {
	limits := &sharepoint.RecipientLimits{}

	if rl.CheckPermissions != nil {
		limits.CheckPermissions = &sharepoint.RecipientLimitsInfo{
			AliasOnly:       rl.CheckPermissions.AliasOnly,
			EmailOnly:       rl.CheckPermissions.EmailOnly,
			MixedRecipients: rl.CheckPermissions.MixedRecipients,
			ObjectIdOnly:    rl.CheckPermissions.ObjectIdOnly,
		}
	}

	if rl.GrantDirectAccess != nil {
		limits.GrantDirectAccess = &sharepoint.RecipientLimitsInfo{
			AliasOnly:       rl.GrantDirectAccess.AliasOnly,
			EmailOnly:       rl.GrantDirectAccess.EmailOnly,
			MixedRecipients: rl.GrantDirectAccess.MixedRecipients,
			ObjectIdOnly:    rl.GrantDirectAccess.ObjectIdOnly,
		}
	}

	if rl.ShareLink != nil {
		limits.ShareLink = &sharepoint.RecipientLimitsInfo{
			AliasOnly:       rl.ShareLink.AliasOnly,
			EmailOnly:       rl.ShareLink.EmailOnly,
			MixedRecipients: rl.ShareLink.MixedRecipients,
			ObjectIdOnly:    rl.ShareLink.ObjectIdOnly,
		}
	}

	if rl.ShareLinkWithDeferRedeem != nil {
		limits.ShareLinkWithDeferRedeem = &sharepoint.RecipientLimitsInfo{
			AliasOnly:       rl.ShareLinkWithDeferRedeem.AliasOnly,
			EmailOnly:       rl.ShareLinkWithDeferRedeem.EmailOnly,
			MixedRecipients: rl.ShareLinkWithDeferRedeem.MixedRecipients,
			ObjectIdOnly:    rl.ShareLinkWithDeferRedeem.ObjectIdOnly,
		}
	}

	return limits
}
