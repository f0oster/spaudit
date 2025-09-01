package contracts

import (
	"context"

	"spaudit/domain/sharepoint"
)

// SharePointAuditRepository provides audit operations for SharePoint sites.
// All operations are scoped to a specific site instance and audit run.
type SharePointAuditRepository interface {
	// Scope information
	GetSiteID() int64
	GetAuditRunID() int64

	// Site operations
	SaveSite(ctx context.Context, site *sharepoint.Site) error
	GetSiteByURL(ctx context.Context, siteURL string) (*sharepoint.Site, error)

	// Web operations
	SaveWeb(ctx context.Context, web *sharepoint.Web) error

	// List operations
	SaveList(ctx context.Context, list *sharepoint.List) error

	// Item operations
	SaveItem(ctx context.Context, item *sharepoint.Item) error

	// Permission operations (site and audit run scoped by default)
	SaveRoleDefinitions(ctx context.Context, roleDefs []*sharepoint.RoleDefinition) error
	SavePrincipal(ctx context.Context, principal *sharepoint.Principal) error
	SaveRoleAssignments(ctx context.Context, assignments []*sharepoint.RoleAssignment) error
	ClearRoleAssignments(ctx context.Context, objectType, objectKey string) error

	// Sharing operations (site and audit run scoped by default)
	SaveSharingLinks(ctx context.Context, links []*sharepoint.SharingLink) error
	ClearSharingLinks(ctx context.Context, itemGUID string) error
	GetAllSharingLinks(ctx context.Context) ([]*sharepoint.Principal, error)
	GetFlexibleSharingLinks(ctx context.Context) ([]*sharepoint.Principal, error)

	// Item lookup operations (site-scoped by default)
	GetItemByGUID(ctx context.Context, itemGUID string) (*sharepoint.Item, error)
	GetItemByListItemGUID(ctx context.Context, listItemGUID string) (*sharepoint.Item, error)
	GetItemByListAndID(ctx context.Context, listID string, itemID int64) (*sharepoint.Item, error)

	// Governance operations (site and audit run scoped by default)
	SaveSharingGovernance(ctx context.Context, sharingInfo *sharepoint.SharingInfo) error
	SaveSharingAbilities(ctx context.Context, abilities *sharepoint.SharingAbilities) error
	SaveRecipientLimits(ctx context.Context, limits *sharepoint.RecipientLimits) error
	SaveSensitivityLabel(ctx context.Context, itemGUID string, label *sharepoint.SensitivityLabelInformation) error
	SaveItemSensitivityLabel(ctx context.Context, label *sharepoint.ItemSensitivityLabel) error
}
