package contracts

import (
	"context"
	"time"

	"spaudit/domain/sharepoint"
)

// SiteWithMetadata represents site information with computed metadata.
type SiteWithMetadata struct {
	Site             *sharepoint.Site
	TotalLists       int
	ListsWithUnique  int
	LastAuditDate    *time.Time
	LastAuditDaysAgo int
}

// SiteRepository defines operations for Site entities with metadata support.
type SiteRepository interface {
	// GetByID retrieves a site by its ID.
	GetByID(ctx context.Context, siteID int64) (*sharepoint.Site, error)

	// Save persists a site.
	Save(ctx context.Context, site *sharepoint.Site) error

	// ListAll retrieves all sites.
	ListAll(ctx context.Context) ([]*sharepoint.Site, error)

	// GetWithMetadata retrieves a site with computed metadata including list statistics and audit history.
	GetWithMetadata(ctx context.Context, siteID int64) (*SiteWithMetadata, error)

	// GetAllWithMetadata retrieves all sites with computed metadata.
	GetAllWithMetadata(ctx context.Context) ([]*SiteWithMetadata, error)
}
