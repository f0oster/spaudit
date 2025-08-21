package contracts

import (
	"context"

	"spaudit/domain/sharepoint"
)

// SharingRepository defines operations for sharing-related entities.
type SharingRepository interface {
	// GetSharingLinksForList retrieves sharing links for a list.
	GetSharingLinksForList(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLink, error)

	// GetSharingLinksWithItemDataForList retrieves sharing links with item data for UI display.
	GetSharingLinksWithItemDataForList(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLinkWithItemData, error)

	// GetSharingLinkMembers retrieves members of a sharing link.
	GetSharingLinkMembers(ctx context.Context, siteID int64, linkID string) ([]*sharepoint.Principal, error)
}
