package repositories

import (
	"context"

	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
)

// SharePointAuditRepositoryImpl provides audit operations scoped to a specific SharePoint site and audit run.
type SharePointAuditRepositoryImpl struct {
	*BaseRepository
	siteID     int64
	auditRunID int64
	auditRepo  contracts.AuditRepository
}

// NewSharePointAuditRepository creates a repository that automatically applies site ID and audit run ID to all operations.
func NewSharePointAuditRepository(
	base *BaseRepository,
	siteID int64,
	auditRunID int64,
	auditRepo contracts.AuditRepository,
) contracts.SharePointAuditRepository {
	return &SharePointAuditRepositoryImpl{
		BaseRepository: base,
		siteID:         siteID,
		auditRunID:     auditRunID,
		auditRepo:      auditRepo,
	}
}

// GetSiteID returns the site ID this repository is scoped to.
func (r *SharePointAuditRepositoryImpl) GetSiteID() int64 {
	return r.siteID
}

// GetAuditRunID returns the audit run ID this repository is scoped to.
func (r *SharePointAuditRepositoryImpl) GetAuditRunID() int64 {
	return r.auditRunID
}

// SaveSite persists a site with automatic site ID assignment.
func (r *SharePointAuditRepositoryImpl) SaveSite(ctx context.Context, site *sharepoint.Site) error {
	// Always ensure the site has our scoped site ID
	site.ID = r.siteID
	return r.auditRepo.SaveSite(ctx, site)
}

// GetSiteByURL retrieves a site by its URL.
func (r *SharePointAuditRepositoryImpl) GetSiteByURL(ctx context.Context, siteURL string) (*sharepoint.Site, error) {
	return r.auditRepo.GetSiteByURL(ctx, siteURL)
}

// SaveWeb persists a web with automatic site ID and audit run ID assignment.
func (r *SharePointAuditRepositoryImpl) SaveWeb(ctx context.Context, web *sharepoint.Web) error {
	web.SiteID = r.siteID
	web.AuditRunID = &r.auditRunID
	return r.auditRepo.SaveWeb(ctx, r.auditRunID, web)
}

// SaveList persists a list with automatic site ID and audit run ID assignment.
func (r *SharePointAuditRepositoryImpl) SaveList(ctx context.Context, list *sharepoint.List) error {
	list.SiteID = r.siteID
	list.AuditRunID = &r.auditRunID
	return r.auditRepo.SaveList(ctx, r.auditRunID, list)
}

// SaveItem persists an item with automatic site ID and audit run ID assignment.
func (r *SharePointAuditRepositoryImpl) SaveItem(ctx context.Context, item *sharepoint.Item) error {
	item.SiteID = r.siteID
	item.AuditRunID = &r.auditRunID
	return r.auditRepo.SaveItem(ctx, r.auditRunID, item)
}

// SaveRoleDefinitions persists role definitions with automatic site ID assignment.
func (r *SharePointAuditRepositoryImpl) SaveRoleDefinitions(ctx context.Context, roleDefs []*sharepoint.RoleDefinition) error {
	// Apply site ID to all role definitions
	for _, roleDef := range roleDefs {
		roleDef.SiteID = r.siteID
	}
	return r.auditRepo.SaveRoleDefinitions(ctx, r.auditRunID, r.siteID, roleDefs)
}

// SavePrincipal persists a principal with automatic site ID assignment.
func (r *SharePointAuditRepositoryImpl) SavePrincipal(ctx context.Context, principal *sharepoint.Principal) error {
	principal.SiteID = r.siteID
	return r.auditRepo.SavePrincipal(ctx, r.auditRunID, principal)
}

// SaveRoleAssignments persists role assignments with automatic site ID assignment.
func (r *SharePointAuditRepositoryImpl) SaveRoleAssignments(ctx context.Context, assignments []*sharepoint.RoleAssignment) error {
	// Apply site ID to all assignments
	for _, assignment := range assignments {
		assignment.SiteID = r.siteID
	}
	return r.auditRepo.SaveRoleAssignments(ctx, r.auditRunID, r.siteID, assignments)
}

// ClearRoleAssignments clears role assignments for an object using the scoped site ID.
func (r *SharePointAuditRepositoryImpl) ClearRoleAssignments(ctx context.Context, objectType, objectKey string) error {
	return r.auditRepo.ClearRoleAssignments(ctx, r.siteID, objectType, objectKey)
}

// SaveSharingLinks persists sharing links with automatic site ID assignment.
func (r *SharePointAuditRepositoryImpl) SaveSharingLinks(ctx context.Context, links []*sharepoint.SharingLink) error {
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
	return r.auditRepo.SaveSharingLinks(ctx, r.auditRunID, r.siteID, links)
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
	return r.auditRepo.SaveSharingGovernance(ctx, r.auditRunID, r.siteID, sharingInfo)
}

// SaveSharingAbilities persists sharing abilities data using the scoped site ID.
func (r *SharePointAuditRepositoryImpl) SaveSharingAbilities(ctx context.Context, abilities *sharepoint.SharingAbilities) error {
	return r.auditRepo.SaveSharingAbilities(ctx, r.auditRunID, r.siteID, abilities)
}

// SaveRecipientLimits persists recipient limits data using the scoped site ID.
func (r *SharePointAuditRepositoryImpl) SaveRecipientLimits(ctx context.Context, limits *sharepoint.RecipientLimits) error {
	return r.auditRepo.SaveRecipientLimits(ctx, r.auditRunID, r.siteID, limits)
}

// SaveSensitivityLabel persists sharing-related sensitivity label data using the scoped site ID.
func (r *SharePointAuditRepositoryImpl) SaveSensitivityLabel(ctx context.Context, itemGUID string, label *sharepoint.SensitivityLabelInformation) error {
	return r.auditRepo.SaveSensitivityLabel(ctx, r.auditRunID, r.siteID, itemGUID, label)
}

// SaveItemSensitivityLabel persists item-level sensitivity label data with automatic site ID assignment.
func (r *SharePointAuditRepositoryImpl) SaveItemSensitivityLabel(ctx context.Context, label *sharepoint.ItemSensitivityLabel) error {
	if label != nil {
		// Ensure site ID matches the scoped repository
		label.SiteID = r.siteID
	}
	return r.auditRepo.SaveItemSensitivityLabel(ctx, label)
}
