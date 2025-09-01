package contracts

import (
	"context"
	"time"

	"spaudit/domain/sharepoint"
)

// SiteContentAggregateRepository handles operations across sites, lists, items, assignments, and sharing.
type SiteContentAggregateRepository interface {
	// Site operations with metadata
	GetSiteWithMetadata(ctx context.Context, siteID int64) (*SiteWithMetadata, error)
	GetAllSitesWithMetadata(ctx context.Context) ([]*SiteWithMetadata, error)

	// Site browsing operations
	SearchSites(ctx context.Context, searchQuery string) ([]*SiteWithMetadata, error)

	// List operations
	GetListByID(ctx context.Context, siteID int64, listID string) (*sharepoint.List, error)
	GetListsForSite(ctx context.Context, siteID int64) ([]*sharepoint.List, error)

	// List assignment operations (audit-scoped)
	GetListAssignmentsWithRootCause(ctx context.Context, siteID int64, auditRunID int64, listID string) ([]*sharepoint.ResolvedAssignment, error)
	GetAssignmentsForObject(ctx context.Context, siteID int64, auditRunID int64, objectType, objectKey string) ([]*sharepoint.Assignment, error)

	// List item operations
	GetListItems(ctx context.Context, siteID int64, listID string, offset, limit int) ([]*sharepoint.Item, error)

	// List sharing operations
	GetListSharingLinks(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLink, error)
	GetListSharingLinksWithItemData(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLinkWithItemData, error)
	GetSharingLinkMembers(ctx context.Context, siteID int64, linkID string) ([]*sharepoint.Principal, error)

	// Job/audit date operations
	GetLastAuditDate(ctx context.Context, siteID int64) (*time.Time, error)
}
