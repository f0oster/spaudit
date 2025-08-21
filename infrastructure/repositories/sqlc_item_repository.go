package repositories

import (
	"context"

	"spaudit/database"
	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/gen/db"
)

// SqlcItemRepository implements contracts.ItemRepository using sqlc with read/write separation
type SqlcItemRepository struct {
	*BaseRepository
}

// NewSqlcItemRepository creates a new item repository with read/write database separation
func NewSqlcItemRepository(database *database.Database) contracts.ItemRepository {
	return &SqlcItemRepository{
		BaseRepository: NewBaseRepository(database),
	}
}

// GetItemsForList retrieves all items for a list
func (r *SqlcItemRepository) GetItemsForList(ctx context.Context, siteID int64, listID string, offset, limit int64) ([]*sharepoint.Item, error) {
	items, err := r.ReadQueries().ItemsForList(ctx, db.ItemsForListParams{
		SiteID: siteID,
		ListID: listID,
		Offset: offset,
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}

	// Transform SQLC rows to domain Items
	domainItems := make([]*sharepoint.Item, len(items))
	for i, item := range items {
		domainItems[i] = &sharepoint.Item{
			SiteID:       item.SiteID,
			GUID:         item.ItemGuid,
			ListItemGUID: r.FromNullString(item.ListItemGuid),
			ListID:       item.ListID,
			ID:           int(item.ItemID),
			URL:          r.FromNullString(item.Url),
			Name:         r.FromNullString(item.Name),
			IsFile:       r.FromNullBool(item.IsFile),
			IsFolder:     r.FromNullBool(item.IsFolder),
			HasUnique:    r.FromNullBool(item.HasUnique),
			AuditRunID:   r.FromNullInt64ToPointer(item.AuditRunID),
		}
	}
	return domainItems, nil
}

// GetItemsWithUniqueForList retrieves only items with unique permissions for a list
func (r *SqlcItemRepository) GetItemsWithUniqueForList(ctx context.Context, siteID int64, listID string, offset, limit int64) ([]*sharepoint.Item, error) {
	items, err := r.ReadQueries().ItemsWithUniqueForList(ctx, db.ItemsWithUniqueForListParams{
		SiteID: siteID,
		ListID: listID,
		Offset: offset,
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}

	// Transform SQLC rows to domain Items
	domainItems := make([]*sharepoint.Item, len(items))
	for i, item := range items {
		domainItems[i] = &sharepoint.Item{
			SiteID:       item.SiteID,
			GUID:         item.ItemGuid,
			ListItemGUID: r.FromNullString(item.ListItemGuid),
			ListID:       item.ListID,
			ID:           int(item.ItemID),
			URL:          r.FromNullString(item.Url),
			Name:         r.FromNullString(item.Name),
			IsFile:       r.FromNullBool(item.IsFile),
			IsFolder:     r.FromNullBool(item.IsFolder),
			HasUnique:    r.FromNullBool(item.HasUnique),
			AuditRunID:   r.FromNullInt64ToPointer(item.AuditRunID),
		}
	}
	return domainItems, nil
}
