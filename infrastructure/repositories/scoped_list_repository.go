package repositories

import (
	"context"

	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/gen/db"
)

// ScopedListRepository wraps a ListRepository with automatic site and audit run scoping
type ScopedListRepository struct {
	*BaseRepository
	queries     *db.Queries
	siteID      int64
	auditRunID  int64
}

// NewScopedListRepository creates a new scoped list repository
func NewScopedListRepository(baseRepo *BaseRepository, queries *db.Queries, siteID, auditRunID int64) contracts.ListRepository {
	return &ScopedListRepository{
		BaseRepository: baseRepo,
		queries:       queries,
		siteID:        siteID,
		auditRunID:    auditRunID,
	}
}

// GetAllForSite returns all lists scoped to the configured site and audit run  
func (r *ScopedListRepository) GetAllForSite(ctx context.Context, siteID int64) ([]*sharepoint.List, error) {
	return r.getListsForSiteAndAuditRun(ctx, siteID, false)
}

// GetListsWithUniquePermissions returns lists with unique permissions scoped to audit run
func (r *ScopedListRepository) GetListsWithUniquePermissions(ctx context.Context, siteID int64) ([]*sharepoint.List, error) {
	return r.getListsForSiteAndAuditRun(ctx, siteID, true)
}

// getListsForSiteAndAuditRun is a helper method to get lists with optional unique filter
func (r *ScopedListRepository) getListsForSiteAndAuditRun(ctx context.Context, siteID int64, uniqueOnly bool) ([]*sharepoint.List, error) {
	// Verify the requested siteID matches our scoped siteID
	if siteID != r.siteID {
		return nil, contracts.ErrSiteScopeMismatch
	}

	var rows []db.GetListsByAuditRunRow
	var err error

	if uniqueOnly {
		uniqueRows, err := r.queries.GetListsWithUniqueByAuditRun(ctx, db.GetListsWithUniqueByAuditRunParams{
			SiteID:     r.siteID,
			AuditRunID: r.auditRunID,
		})
		if err != nil {
			return nil, err
		}
		// Convert unique rows to regular rows format
		rows = make([]db.GetListsByAuditRunRow, len(uniqueRows))
		for i, ur := range uniqueRows {
			rows[i] = db.GetListsByAuditRunRow{
				SiteID:       ur.SiteID,
				ListID:       ur.ListID,
				WebID:        ur.WebID,
				Title:        ur.Title,
				Url:          ur.Url,
				BaseTemplate: ur.BaseTemplate,
				ItemCount:    ur.ItemCount,
				HasUnique:    ur.HasUnique,
				WebTitle:     ur.WebTitle,
				AuditRunID:   ur.AuditRunID,
			}
		}
	} else {
		rows, err = r.queries.GetListsByAuditRun(ctx, db.GetListsByAuditRunParams{
			SiteID:     r.siteID,
			AuditRunID: r.auditRunID,
		})
		if err != nil {
			return nil, err
		}
	}

	// Convert to domain objects
	lists := make([]*sharepoint.List, 0, len(rows))
	for _, row := range rows {
		list := &sharepoint.List{
			ID:           row.ListID,
			SiteID:       row.SiteID,
			WebID:        row.WebID,
			Title:        row.Title,
			URL:          r.FromNullString(row.Url),
			BaseTemplate: int(r.FromNullInt64(row.BaseTemplate)),
			ItemCount:    int(r.FromNullInt64(row.ItemCount)),
			HasUnique:    r.FromNullBool(row.HasUnique),
			AuditRunID:   &r.auditRunID,
		}
		lists = append(lists, list)
	}

	return lists, nil
}


// GetByID gets a specific list scoped to audit run
func (r *ScopedListRepository) GetByID(ctx context.Context, siteID int64, listID string) (*sharepoint.List, error) {
	// Verify the requested siteID matches our scoped siteID
	if siteID != r.siteID {
		return nil, contracts.ErrSiteScopeMismatch
	}

	// Query specific list with audit run filtering
	row, err := r.queries.GetListByAuditRun(ctx, db.GetListByAuditRunParams{
		SiteID:     r.siteID,
		ListID:     listID,
		AuditRunID: r.auditRunID,
	})
	if err != nil {
		return nil, err
	}

	// Convert to domain object
	list := &sharepoint.List{
		ID:           row.ListID,
		SiteID:       row.SiteID,
		WebID:        row.WebID,
		Title:        row.Title,
		URL:          r.FromNullString(row.Url),
		BaseTemplate: int(r.FromNullInt64(row.BaseTemplate)),
		ItemCount:    int(r.FromNullInt64(row.ItemCount)),
		HasUnique:    r.FromNullBool(row.HasUnique),
		AuditRunID:   &r.auditRunID,
	}

	return list, nil
}

// GetByWebID gets all lists for a web scoped to audit run
func (r *ScopedListRepository) GetByWebID(ctx context.Context, siteID int64, webID string) ([]*sharepoint.List, error) {
	// Verify the requested siteID matches our scoped siteID
	if siteID != r.siteID {
		return nil, contracts.ErrSiteScopeMismatch
	}

	// For now, get all lists and filter by webID in memory
	// TODO: Add audit-run-scoped GetListsByWebID query
	allLists, err := r.GetAllForSite(ctx, siteID)
	if err != nil {
		return nil, err
	}

	var webLists []*sharepoint.List
	for _, list := range allLists {
		if list.WebID == webID {
			webLists = append(webLists, list)
		}
	}

	return webLists, nil
}

// Save is not implemented for scoped repository (use audit repository for saving)
func (r *ScopedListRepository) Save(ctx context.Context, list *sharepoint.List) error {
	panic("Save not supported on scoped repository - use audit repository for saving")
}

