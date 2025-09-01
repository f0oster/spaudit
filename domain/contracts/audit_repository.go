package contracts

import (
	"context"

	"spaudit/domain/sharepoint"
)

// AuditRepository defines the interface for audit data persistence.
type AuditRepository interface {
	// Site operations
	SaveSite(ctx context.Context, site *sharepoint.Site) error
	GetSiteByURL(ctx context.Context, siteURL string) (*sharepoint.Site, error)

	// Web operations
	SaveWeb(ctx context.Context, auditRunID int64, web *sharepoint.Web) error

	// List operations
	SaveList(ctx context.Context, auditRunID int64, list *sharepoint.List) error

	// Item operations
	SaveItem(ctx context.Context, auditRunID int64, item *sharepoint.Item) error

	// Permission operations
	SaveRoleDefinitions(ctx context.Context, auditRunID int64, siteID int64, roleDefs []*sharepoint.RoleDefinition) error
	SavePrincipal(ctx context.Context, auditRunID int64, principal *sharepoint.Principal) error
	SaveRoleAssignments(ctx context.Context, auditRunID int64, siteID int64, assignments []*sharepoint.RoleAssignment) error
	ClearRoleAssignments(ctx context.Context, siteID int64, objectType, objectKey string) error

	// Sharing operations
	SaveSharingLinks(ctx context.Context, auditRunID int64, siteID int64, links []*sharepoint.SharingLink) error
	ClearSharingLinks(ctx context.Context, siteID int64, itemGUID string) error
	GetAllSharingLinks(ctx context.Context, siteID int64) ([]*sharepoint.Principal, error)
	GetFlexibleSharingLinks(ctx context.Context, siteID int64) ([]*sharepoint.Principal, error)

	// Item lookup operations
	GetItemByGUID(ctx context.Context, siteID int64, itemGUID string) (*sharepoint.Item, error)
	GetItemByListItemGUID(ctx context.Context, siteID int64, listItemGUID string) (*sharepoint.Item, error)
	GetItemByListAndID(ctx context.Context, siteID int64, listID string, itemID int64) (*sharepoint.Item, error)

	// Governance operations
	SaveSharingGovernance(ctx context.Context, auditRunID, siteID int64, sharingInfo *sharepoint.SharingInfo) error
	SaveSharingAbilities(ctx context.Context, auditRunID, siteID int64, abilities *sharepoint.SharingAbilities) error
	SaveRecipientLimits(ctx context.Context, auditRunID, siteID int64, limits *sharepoint.RecipientLimits) error
	SaveSensitivityLabel(ctx context.Context, auditRunID, siteID int64, itemGUID string, label *sharepoint.SensitivityLabelInformation) error
	SaveItemSensitivityLabel(ctx context.Context, label *sharepoint.ItemSensitivityLabel) error
}
