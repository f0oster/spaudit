package repositories

import (
	"context"

	"spaudit/database"
	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/gen/db"
)

// SqlcListRepository implements contracts.ListRepository using sqlc queries with read/write separation.
type SqlcListRepository struct {
	*BaseRepository
}

// NewSqlcListRepository creates a new list repository with read/write database separation.
func NewSqlcListRepository(database *database.Database) contracts.ListRepository {
	return &SqlcListRepository{
		BaseRepository: NewBaseRepository(database),
	}
}

// Save persists a list to the database.
func (r *SqlcListRepository) Save(ctx context.Context, list *sharepoint.List) error {
	// Transform domain List to SQLC params
	params := db.InsertListParams{
		SiteID:       list.SiteID,
		ListID:       list.ID,
		WebID:        list.WebID,
		Title:        list.Title,
		Url:          r.ToNullString(list.URL),
		BaseTemplate: r.ToNullInt64(int64(list.BaseTemplate)),
		ItemCount:    r.ToNullInt64(int64(list.ItemCount)),
		HasUnique:    r.ToNullBool(list.HasUnique),
	}
	return r.WriteQueries().InsertList(ctx, params)
}

// GetByID retrieves a list by its ID.
func (r *SqlcListRepository) GetByID(ctx context.Context, siteID int64, listID string) (*sharepoint.List, error) {
	sqlcList, err := r.ReadQueries().GetList(ctx, db.GetListParams{
		SiteID: siteID,
		ListID: listID,
	})
	if err != nil {
		return nil, err
	}

	// Transform SQLC row to domain List
	return &sharepoint.List{
		SiteID:       sqlcList.SiteID,
		ID:           sqlcList.ListID,
		WebID:        sqlcList.WebID,
		Title:        sqlcList.Title,
		URL:          r.FromNullString(sqlcList.Url),
		BaseTemplate: int(r.FromNullInt64(sqlcList.BaseTemplate)),
		ItemCount:    int(r.FromNullInt64(sqlcList.ItemCount)),
		HasUnique:    r.FromNullBool(sqlcList.HasUnique),
		AuditRunID:   r.FromNullInt64ToPointer(sqlcList.AuditRunID),
	}, nil
}

// GetByWebID retrieves all lists for a web.
func (r *SqlcListRepository) GetByWebID(ctx context.Context, siteID int64, webID string) ([]*sharepoint.List, error) {
	sqlcLists, err := r.ReadQueries().GetListsByWebID(ctx, db.GetListsByWebIDParams{
		SiteID: siteID,
		WebID:  webID,
	})
	if err != nil {
		return nil, err
	}

	// Transform SQLC rows to domain Lists
	domainLists := make([]*sharepoint.List, len(sqlcLists))
	for i, row := range sqlcLists {
		domainLists[i] = &sharepoint.List{
			SiteID:       row.SiteID,
			ID:           row.ListID,
			WebID:        row.WebID,
			Title:        row.Title,
			URL:          r.FromNullString(row.Url),
			BaseTemplate: int(r.FromNullInt64(row.BaseTemplate)),
			ItemCount:    int(r.FromNullInt64(row.ItemCount)),
			HasUnique:    r.FromNullBool(row.HasUnique),
			AuditRunID:   r.FromNullInt64ToPointer(row.AuditRunID),
		}
	}
	return domainLists, nil
}

// GetAllForSite retrieves all lists for a site.
func (r *SqlcListRepository) GetAllForSite(ctx context.Context, siteID int64) ([]*sharepoint.List, error) {
	sqlcLists, err := r.ReadQueries().GetListsForSite(ctx, siteID)
	if err != nil {
		return nil, err
	}

	// Transform SQLC rows to domain Lists
	domainLists := make([]*sharepoint.List, len(sqlcLists))
	for i, row := range sqlcLists {
		domainLists[i] = &sharepoint.List{
			SiteID:       row.SiteID,
			ID:           row.ListID,
			WebID:        row.WebID,
			Title:        row.Title,
			URL:          r.FromNullString(row.Url),
			BaseTemplate: 0, // Not available in this query
			ItemCount:    int(r.FromNullInt64(row.ItemCount)),
			HasUnique:    r.FromNullBool(row.HasUnique),
			AuditRunID:   nil, // Not available in this query
		}
	}
	return domainLists, nil
}
