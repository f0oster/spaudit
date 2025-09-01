package repositories

import (
	"context"
	"database/sql"

	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/gen/db"
)

// ScopedItemRepository wraps an ItemRepository with automatic site and audit run scoping
type ScopedItemRepository struct {
	*BaseRepository
	queries     *db.Queries
	siteID      int64
	auditRunID  int64
}

// NewScopedItemRepository creates a new scoped item repository
func NewScopedItemRepository(baseRepo *BaseRepository, queries *db.Queries, siteID, auditRunID int64) contracts.ItemRepository {
	return &ScopedItemRepository{
		BaseRepository: baseRepo,
		queries:       queries,
		siteID:        siteID,
		auditRunID:    auditRunID,
	}
}

// GetByGUID gets an item by GUID scoped to audit run
func (r *ScopedItemRepository) GetByGUID(ctx context.Context, siteID int64, itemGUID string) (*sharepoint.Item, error) {
	// Verify the requested siteID matches our scoped siteID
	if siteID != r.siteID {
		return nil, contracts.ErrSiteScopeMismatch
	}

	// Get item with audit run scoping
	row, err := r.queries.GetItemByGUID(ctx, db.GetItemByGUIDParams{
		SiteID:   r.siteID,
		ItemGuid: itemGUID,
	})
	if err != nil {
		return nil, err
	}

	// Verify the item is from the correct audit run
	if row.AuditRunID != r.auditRunID {
		return nil, sql.ErrNoRows // Item not found in this audit run
	}

	return &sharepoint.Item{
		SiteID:       row.SiteID,
		GUID:         row.ItemGuid,
		ListItemGUID: r.FromNullString(row.ListItemGuid),
		ListID:       row.ListID,
		ID:           int(row.ItemID),
		URL:          r.FromNullString(row.Url),
		IsFile:       r.FromNullBool(row.IsFile),
		IsFolder:     r.FromNullBool(row.IsFolder),
		HasUnique:    r.FromNullBool(row.HasUnique),
		Name:         r.FromNullString(row.Name),
		AuditRunID:   &r.auditRunID,
	}, nil
}

// GetItemsForList gets items for a list scoped to audit run  
func (r *ScopedItemRepository) GetItemsForList(ctx context.Context, siteID int64, listID string, offset, limit int64) ([]*sharepoint.Item, error) {
	// Verify the requested siteID matches our scoped siteID
	if siteID != r.siteID {
		return nil, contracts.ErrSiteScopeMismatch
	}

	// Get items with audit run scoping
	rows, err := r.queries.ItemsForListByAuditRun(ctx, db.ItemsForListByAuditRunParams{
		SiteID: r.siteID,
		ListID: listID,
		AuditRunID: r.auditRunID,
		Offset: offset,
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}

	// Transform rows to domain objects
	var items []*sharepoint.Item
	for _, row := range rows {

		item := &sharepoint.Item{
			SiteID:       row.SiteID,
			GUID:         row.ItemGuid,
			ListItemGUID: r.FromNullString(row.ListItemGuid),
			ListID:       row.ListID,
			ID:           int(row.ItemID),
			URL:          r.FromNullString(row.Url),
			IsFile:       r.FromNullBool(row.IsFile),
			IsFolder:     r.FromNullBool(row.IsFolder),
			HasUnique:    r.FromNullBool(row.HasUnique),
			Name:         r.FromNullString(row.Name),
			AuditRunID:   &r.auditRunID,
		}
		items = append(items, item)
	}

	return items, nil
}

// GetItemsWithUniqueForList gets items with unique permissions for a list scoped to audit run
func (r *ScopedItemRepository) GetItemsWithUniqueForList(ctx context.Context, siteID int64, listID string, offset, limit int64) ([]*sharepoint.Item, error) {
	// Verify the requested siteID matches our scoped siteID
	if siteID != r.siteID {
		return nil, contracts.ErrSiteScopeMismatch
	}

	// Get items with unique permissions and audit run scoping
	rows, err := r.queries.ItemsWithUniqueForListByAuditRun(ctx, db.ItemsWithUniqueForListByAuditRunParams{
		SiteID: r.siteID,
		ListID: listID,
		AuditRunID: r.auditRunID,
		Offset: offset,
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}

	// Transform rows to domain objects
	var items []*sharepoint.Item
	for _, row := range rows {

		item := &sharepoint.Item{
			SiteID:       row.SiteID,
			GUID:         row.ItemGuid,
			ListItemGUID: r.FromNullString(row.ListItemGuid),
			ListID:       row.ListID,
			ID:           int(row.ItemID),
			URL:          r.FromNullString(row.Url),
			IsFile:       r.FromNullBool(row.IsFile),
			IsFolder:     r.FromNullBool(row.IsFolder),
			HasUnique:    r.FromNullBool(row.HasUnique),
			Name:         r.FromNullString(row.Name),
			AuditRunID:   &r.auditRunID,
		}
		items = append(items, item)
	}

	return items, nil
}

// Save is not implemented for scoped repository (use audit repository for saving)
func (r *ScopedItemRepository) Save(ctx context.Context, item *sharepoint.Item) error {
	panic("Save not supported on scoped repository - use audit repository for saving")
}

