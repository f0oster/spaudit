package repositories

import (
	"context"
	"database/sql"
	"time"

	"spaudit/database"
	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/gen/db"
)

// SqlcSiteRepository implements contracts.SiteRepository using sqlc queries with read/write separation.
type SqlcSiteRepository struct {
	*BaseRepository
}

// NewSqlcSiteRepository creates a new site repository with read/write database separation.
func NewSqlcSiteRepository(database *database.Database) contracts.SiteRepository {
	return &SqlcSiteRepository{
		BaseRepository: NewBaseRepository(database),
	}
}

// GetByID retrieves a site by its ID.
func (r *SqlcSiteRepository) GetByID(ctx context.Context, siteID int64) (*sharepoint.Site, error) {
	siteRow, err := r.ReadQueries().GetSiteByID(ctx, siteID)
	if err != nil {
		return nil, err
	}

	// Transform SQLC row to domain Site
	return &sharepoint.Site{
		ID:        siteRow.SiteID,
		URL:       siteRow.SiteUrl,
		Title:     r.FromNullString(siteRow.Title),
		CreatedAt: r.FromNullTime(siteRow.CreatedAt),
		UpdatedAt: r.FromNullTime(siteRow.UpdatedAt),
	}, nil
}

// Save persists a site to the database.
func (r *SqlcSiteRepository) Save(ctx context.Context, site *sharepoint.Site) error {
	// Transform domain Site to SQLC params
	params := db.UpsertSiteParams{
		SiteUrl: site.URL,
		Title:   r.ToNullString(site.Title),
	}
	_, err := r.WriteQueries().UpsertSite(ctx, params)
	return err
}

// ListAll retrieves all sites.
func (r *SqlcSiteRepository) ListAll(ctx context.Context) ([]*sharepoint.Site, error) {
	siteRows, err := r.ReadQueries().ListSites(ctx)
	if err != nil {
		return nil, err
	}

	// Transform SQLC rows to domain Sites
	sites := make([]*sharepoint.Site, len(siteRows))
	for i, row := range siteRows {
		sites[i] = &sharepoint.Site{
			ID:        row.SiteID,
			URL:       row.SiteUrl,
			Title:     r.FromNullString(row.Title),
			CreatedAt: r.FromNullTime(row.CreatedAt),
			UpdatedAt: r.FromNullTime(row.UpdatedAt),
		}
	}
	return sites, nil
}

// GetWithMetadata retrieves a site with computed metadata including list statistics and audit history.
func (r *SqlcSiteRepository) GetWithMetadata(ctx context.Context, siteID int64) (*contracts.SiteWithMetadata, error) {
	// Retrieve basic site information
	siteInfo, err := r.ReadQueries().GetSiteByID(ctx, siteID)
	if err != nil {
		return nil, err
	}

	// Convert to domain model
	site := &sharepoint.Site{
		ID:        siteInfo.SiteID,
		URL:       siteInfo.SiteUrl,
		Title:     r.FromNullString(siteInfo.Title),
		CreatedAt: r.FromNullTime(siteInfo.CreatedAt),
		UpdatedAt: r.FromNullTime(siteInfo.UpdatedAt),
	}

	// Retrieve list statistics for metadata computation
	listRows, err := r.ReadQueries().GetListsForSite(ctx, siteID)
	if err != nil {
		return nil, err
	}

	totalLists := len(listRows)
	listsWithUnique := 0

	for _, list := range listRows {
		if r.FromNullBool(list.HasUnique) {
			listsWithUnique++
		}
	}

	// Retrieve last completed audit date for this site
	var lastAuditDate *time.Time
	daysAgo := 0

	lastJob, err := r.ReadQueries().GetLastCompletedJobForSite(ctx, db.GetLastCompletedJobForSiteParams{
		SiteID: sql.NullInt64{
			Int64: siteID,
			Valid: true,
		},
		SiteUrl: siteInfo.SiteUrl,
	})
	if err == nil && lastJob.CompletedAt.Valid {
		completedTime := lastJob.CompletedAt.Time
		lastAuditDate = &completedTime

		// Calculate days since last audit for display
		daysAgo = int(time.Since(completedTime).Hours() / 24)
	}

	return &contracts.SiteWithMetadata{
		Site:             site,
		TotalLists:       totalLists,
		ListsWithUnique:  listsWithUnique,
		LastAuditDate:    lastAuditDate,
		LastAuditDaysAgo: daysAgo,
	}, nil
}

// GetAllWithMetadata retrieves all sites with computed metadata.
func (r *SqlcSiteRepository) GetAllWithMetadata(ctx context.Context) ([]*contracts.SiteWithMetadata, error) {
	siteRows, err := r.ReadQueries().ListSites(ctx)
	if err != nil {
		return nil, err
	}

	sitesWithMetadata := make([]*contracts.SiteWithMetadata, len(siteRows))
	for i, site := range siteRows {
		siteWithMetadata, err := r.GetWithMetadata(ctx, site.SiteID)
		if err != nil {
			// Use basic site info if metadata computation fails
			sitesWithMetadata[i] = &contracts.SiteWithMetadata{
				Site: &sharepoint.Site{
					ID:        site.SiteID,
					URL:       site.SiteUrl,
					Title:     r.FromNullString(site.Title),
					CreatedAt: r.FromNullTime(site.CreatedAt),
					UpdatedAt: r.FromNullTime(site.UpdatedAt),
				},
				TotalLists:       0,
				ListsWithUnique:  0,
				LastAuditDate:    nil,
				LastAuditDaysAgo: 0,
			}
			continue
		}
		sitesWithMetadata[i] = siteWithMetadata
	}

	return sitesWithMetadata, nil
}
