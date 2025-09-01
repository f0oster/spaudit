package repositories

import (
	"context"

	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/gen/db"
)

// ScopedSiteRepository wraps a SiteRepository with automatic site and audit run scoping
type ScopedSiteRepository struct {
	*BaseRepository
	queries     *db.Queries
	siteID      int64
	auditRunID  int64
}

// NewScopedSiteRepository creates a new scoped site repository
func NewScopedSiteRepository(baseRepo *BaseRepository, queries *db.Queries, siteID, auditRunID int64) contracts.SiteRepository {
	return &ScopedSiteRepository{
		BaseRepository: baseRepo,
		queries:       queries,
		siteID:        siteID,
		auditRunID:    auditRunID,
	}
}

// GetByID retrieves a site by ID (scoped to configured site)
func (r *ScopedSiteRepository) GetByID(ctx context.Context, siteID int64) (*sharepoint.Site, error) {
	// Verify the requested siteID matches our scoped siteID
	if siteID != r.siteID {
		return nil, contracts.ErrSiteScopeMismatch
	}

	// Get the basic site info
	site, err := r.queries.GetSiteByID(ctx, r.siteID)
	if err != nil {
		return nil, err
	}

	return &sharepoint.Site{
		ID:    site.SiteID,
		URL:   site.SiteUrl,
		Title: r.FromNullString(site.Title),
	}, nil
}

// GetWithMetadata returns site with metadata scoped to the configured audit run
func (r *ScopedSiteRepository) GetWithMetadata(ctx context.Context, siteID int64) (*contracts.SiteWithMetadata, error) {
	// Verify the requested siteID matches our scoped siteID
	if siteID != r.siteID {
		return nil, contracts.ErrSiteScopeMismatch
	}

	// Get the basic site info
	site, err := r.queries.GetSiteByID(ctx, r.siteID)
	if err != nil {
		return nil, err
	}

	// Calculate metadata for this specific audit run
	listsRows, err := r.queries.GetListsByAuditRun(ctx, db.GetListsByAuditRunParams{
		SiteID:     r.siteID,
		AuditRunID: r.auditRunID,
	})
	if err != nil {
		return nil, err
	}

	// Calculate metadata from scoped lists
	totalLists := len(listsRows)
	listsWithUnique := 0
	totalItems := int64(0)
	
	for _, list := range listsRows {
		if r.FromNullBool(list.HasUnique) {
			listsWithUnique++
		}
		totalItems += r.FromNullInt64(list.ItemCount)
	}

	// Get audit run info for last audit date
	auditRun, err := r.queries.GetAuditRun(ctx, r.auditRunID)
	if err != nil {
		return nil, err
	}

	return &contracts.SiteWithMetadata{
		Site: &sharepoint.Site{
			ID:    site.SiteID,
			URL:   site.SiteUrl,
			Title: r.FromNullString(site.Title),
		},
		TotalLists:       totalLists,
		ListsWithUnique:  listsWithUnique,
		LastAuditDate:    &auditRun.StartedAt,
		LastAuditDaysAgo: int(auditRun.StartedAt.UTC().Sub(auditRun.StartedAt.UTC()).Hours() / 24), // TODO: Calculate actual days ago
	}, nil
}

// ListAll is not implemented for scoped repository (use unscoped for listing all sites)
func (r *ScopedSiteRepository) ListAll(ctx context.Context) ([]*sharepoint.Site, error) {
	panic("ListAll not supported on scoped repository - use unscoped for listing all sites")
}

// GetAllWithMetadata is not implemented for scoped repository
func (r *ScopedSiteRepository) GetAllWithMetadata(ctx context.Context) ([]*contracts.SiteWithMetadata, error) {
	panic("GetAllWithMetadata not supported on scoped repository - use unscoped for listing all sites")
}

// Save is not implemented for scoped repository (use audit repository for saving)
func (r *ScopedSiteRepository) Save(ctx context.Context, site *sharepoint.Site) error {
	panic("Save not supported on scoped repository - use audit repository for saving")
}

