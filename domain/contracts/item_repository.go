package contracts

import (
	"context"

	"spaudit/domain/sharepoint"
)

// ItemRepository defines operations for Item entities.
// TODO: Enhance repository interface for pagination and performance:
// - Add GetItemsCount and GetItemsWithUniqueCount methods for total counts
// - Consider adding GetItemsSummaryForList for lighter-weight queries (IDs, names, types only)
// - Add sorting parameters to methods (by name, date, size, risk level)
// - Add filtering parameters (by type, permission level, last modified date)
// - Consider cursor-based pagination for very large datasets
type ItemRepository interface {
	// GetItemsForList retrieves all items for a list.
	GetItemsForList(ctx context.Context, siteID int64, listID string, offset, limit int64) ([]*sharepoint.Item, error)

	// GetItemsWithUniqueForList retrieves only items with unique permissions for a list.
	GetItemsWithUniqueForList(ctx context.Context, siteID int64, listID string, offset, limit int64) ([]*sharepoint.Item, error)

	// TODO: Add these methods for proper pagination support:
	// GetItemsCountForList(ctx context.Context, siteID int64, listID string) (int64, error)
	// GetItemsWithUniqueCountForList(ctx context.Context, siteID int64, listID string) (int64, error)
	// GetItemsSummaryForList(...) - lightweight version with minimal fields
}
