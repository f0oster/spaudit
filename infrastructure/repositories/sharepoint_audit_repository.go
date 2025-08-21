package repositories

import (
	"context"

	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
)

// SharePointAuditRepositoryImpl provides audit operations scoped to a specific SharePoint site.
type SharePointAuditRepositoryImpl struct {
	*BaseRepository
	siteID    int64
	auditRepo contracts.AuditRepository
}

// NewSharePointAuditRepository creates a repository that automatically applies site ID to all operations.
func NewSharePointAuditRepository(
	base *BaseRepository,
	siteID int64,
	auditRepo contracts.AuditRepository,
) contracts.SharePointAuditRepository {
	return &SharePointAuditRepositoryImpl{
		BaseRepository: base,
		siteID:         siteID,
		auditRepo:      auditRepo,
	}
}

// GetSiteID returns the site ID this repository is scoped to.
func (r *SharePointAuditRepositoryImpl) GetSiteID() int64 {
	return r.siteID
}

// SaveSite persists a site with site ID validation.
func (r *SharePointAuditRepositoryImpl) SaveSite(ctx context.Context, site *sharepoint.Site) error {
	// Handle the case where we're updating a placeholder site with real SharePoint data
	if site.ID == 0 {
		// New site being saved - set it to our scoped site ID
		site.ID = r.siteID
	} else if site.ID != r.siteID {
		// Site has a different ID than our scope - this is a safety check
		// Only allow if our scope is 0 (placeholder) or IDs match
		if r.siteID != 0 {
			return ErrSiteMismatch{Expected: r.siteID, Actual: site.ID}
		}
		// If our scope is 0, don't mutate the repository's siteID field
		// as this creates thread-safety issues with concurrent jobs
		// The repository should remain immutable after creation
	}
	return r.auditRepo.SaveSite(ctx, site)
}

// GetSiteByURL retrieves a site by its URL.
func (r *SharePointAuditRepositoryImpl) GetSiteByURL(ctx context.Context, siteURL string) (*sharepoint.Site, error) {
	return r.auditRepo.GetSiteByURL(ctx, siteURL)
}

// SaveWeb persists a web with automatic site ID assignment.
func (r *SharePointAuditRepositoryImpl) SaveWeb(ctx context.Context, auditRunID int64, web *sharepoint.Web) error {
	web.SiteID = r.siteID
	return r.auditRepo.SaveWeb(ctx, auditRunID, web)
}

// SaveList persists a list with automatic site ID assignment.
func (r *SharePointAuditRepositoryImpl) SaveList(ctx context.Context, auditRunID int64, list *sharepoint.List) error {
	list.SiteID = r.siteID
	return r.auditRepo.SaveList(ctx, auditRunID, list)
}

// SaveItem persists an item with automatic site ID assignment.
func (r *SharePointAuditRepositoryImpl) SaveItem(ctx context.Context, auditRunID int64, item *sharepoint.Item) error {
	item.SiteID = r.siteID
	return r.auditRepo.SaveItem(ctx, auditRunID, item)
}

// SaveRoleDefinitions persists role definitions with automatic site ID assignment.
func (r *SharePointAuditRepositoryImpl) SaveRoleDefinitions(ctx context.Context, auditRunID int64, roleDefs []*sharepoint.RoleDefinition) error {
	// Apply site ID to all role definitions
	for _, roleDef := range roleDefs {
		roleDef.SiteID = r.siteID
	}
	return r.auditRepo.SaveRoleDefinitions(ctx, auditRunID, r.siteID, roleDefs)
}

// SavePrincipal persists a principal with automatic site ID assignment.
func (r *SharePointAuditRepositoryImpl) SavePrincipal(ctx context.Context, auditRunID int64, principal *sharepoint.Principal) error {
	principal.SiteID = r.siteID
	return r.auditRepo.SavePrincipal(ctx, auditRunID, principal)
}

// SaveRoleAssignments persists role assignments with automatic site ID assignment.
func (r *SharePointAuditRepositoryImpl) SaveRoleAssignments(ctx context.Context, auditRunID int64, assignments []*sharepoint.RoleAssignment) error {
	// Apply site ID to all assignments
	for _, assignment := range assignments {
		assignment.SiteID = r.siteID
	}
	return r.auditRepo.SaveRoleAssignments(ctx, auditRunID, r.siteID, assignments)
}

// ClearRoleAssignments clears role assignments for an object using the scoped site ID.
func (r *SharePointAuditRepositoryImpl) ClearRoleAssignments(ctx context.Context, objectType, objectKey string) error {
	return r.auditRepo.ClearRoleAssignments(ctx, r.siteID, objectType, objectKey)
}

// SaveSharingLinks persists sharing links with automatic site ID assignment.
func (r *SharePointAuditRepositoryImpl) SaveSharingLinks(ctx context.Context, auditRunID int64, links []*sharepoint.SharingLink) error {
	// Apply site ID to all links and their nested principals
	for _, link := range links {
		link.SiteID = r.siteID

		// Ensure site ID is set at the domain level for consistency
		if link.CreatedBy != nil {
			link.CreatedBy.SiteID = r.siteID
		}
		if link.LastModifiedBy != nil {
			link.LastModifiedBy.SiteID = r.siteID
		}
		for _, member := range link.Members {
			member.SiteID = r.siteID
		}
	}
	return r.auditRepo.SaveSharingLinks(ctx, auditRunID, r.siteID, links)
}

// ClearSharingLinks clears sharing links for an item using the scoped site ID.
func (r *SharePointAuditRepositoryImpl) ClearSharingLinks(ctx context.Context, itemGUID string) error {
	return r.auditRepo.ClearSharingLinks(ctx, r.siteID, itemGUID)
}

// GetAllSharingLinks retrieves all sharing links for the scoped site.
func (r *SharePointAuditRepositoryImpl) GetAllSharingLinks(ctx context.Context) ([]*sharepoint.Principal, error) {
	return r.auditRepo.GetAllSharingLinks(ctx, r.siteID)
}

// GetFlexibleSharingLinks retrieves flexible sharing links for the scoped site.
func (r *SharePointAuditRepositoryImpl) GetFlexibleSharingLinks(ctx context.Context) ([]*sharepoint.Principal, error) {
	return r.auditRepo.GetFlexibleSharingLinks(ctx, r.siteID)
}

// GetItemByGUID retrieves an item by GUID using the scoped site ID.
func (r *SharePointAuditRepositoryImpl) GetItemByGUID(ctx context.Context, itemGUID string) (*sharepoint.Item, error) {
	return r.auditRepo.GetItemByGUID(ctx, r.siteID, itemGUID)
}

// GetItemByListItemGUID retrieves an item by list item GUID using the scoped site ID.
func (r *SharePointAuditRepositoryImpl) GetItemByListItemGUID(ctx context.Context, listItemGUID string) (*sharepoint.Item, error) {
	return r.auditRepo.GetItemByListItemGUID(ctx, r.siteID, listItemGUID)
}

// GetItemByListAndID retrieves an item by list and ID using the scoped site ID.
func (r *SharePointAuditRepositoryImpl) GetItemByListAndID(ctx context.Context, listID string, itemID int64) (*sharepoint.Item, error) {
	return r.auditRepo.GetItemByListAndID(ctx, r.siteID, listID, itemID)
}

// SaveSharingGovernance persists sharing governance data using the scoped site ID.
func (r *SharePointAuditRepositoryImpl) SaveSharingGovernance(ctx context.Context, sharingInfo *sharepoint.SharingInfo) error {
	return r.auditRepo.SaveSharingGovernance(ctx, r.siteID, sharingInfo)
}

// SaveSharingAbilities persists sharing abilities data using the scoped site ID.
func (r *SharePointAuditRepositoryImpl) SaveSharingAbilities(ctx context.Context, abilities *sharepoint.SharingAbilities) error {
	return r.auditRepo.SaveSharingAbilities(ctx, r.siteID, abilities)
}

// SaveRecipientLimits persists recipient limits data using the scoped site ID.
func (r *SharePointAuditRepositoryImpl) SaveRecipientLimits(ctx context.Context, limits *sharepoint.RecipientLimits) error {
	return r.auditRepo.SaveRecipientLimits(ctx, r.siteID, limits)
}

// SaveSensitivityLabel persists sharing-related sensitivity label data using the scoped site ID.
func (r *SharePointAuditRepositoryImpl) SaveSensitivityLabel(ctx context.Context, itemGUID string, label *sharepoint.SensitivityLabelInformation) error {
	return r.auditRepo.SaveSensitivityLabel(ctx, r.siteID, itemGUID, label)
}

// SaveItemSensitivityLabel persists item-level sensitivity label data with automatic site ID assignment.
func (r *SharePointAuditRepositoryImpl) SaveItemSensitivityLabel(ctx context.Context, label *sharepoint.ItemSensitivityLabel) error {
	if label != nil {
		// Ensure site ID matches the scoped repository
		label.SiteID = r.siteID
	}
	return r.auditRepo.SaveItemSensitivityLabel(ctx, label)
}
