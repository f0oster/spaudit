package contracts

import (
	"context"

	"spaudit/domain/sharepoint"
)

// ListRepository defines operations for List entities.
type ListRepository interface {
	// Save persists a list to the database.
	Save(ctx context.Context, list *sharepoint.List) error

	// GetByID retrieves a list by its ID, returns domain model.
	GetByID(ctx context.Context, siteID int64, listID string) (*sharepoint.List, error)

	// GetByWebID retrieves all lists for a web, returns domain models.
	GetByWebID(ctx context.Context, siteID int64, webID string) ([]*sharepoint.List, error)

	// GetAllForSite retrieves all lists for a site with their metadata.
	GetAllForSite(ctx context.Context, siteID int64) ([]*sharepoint.List, error)
}
