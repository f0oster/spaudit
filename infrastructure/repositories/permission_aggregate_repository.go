package repositories

import (
	"context"
	"fmt"

	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/gen/db"
)

// PermissionAggregateRepositoryImpl implements the permission aggregate repository by composing entity repositories.
type PermissionAggregateRepositoryImpl struct {
	*BaseRepository
	itemRepo           contracts.ItemRepository
	sharingRepo        contracts.SharingRepository
	permissionsService *sharepoint.PermissionsService
	contentService     *sharepoint.ContentService
}

// NewPermissionAggregateRepository creates a new permission aggregate repository.
func NewPermissionAggregateRepository(
	base *BaseRepository,
	itemRepo contracts.ItemRepository,
	sharingRepo contracts.SharingRepository,
) contracts.PermissionAggregateRepository {
	return &PermissionAggregateRepositoryImpl{
		BaseRepository:     base,
		itemRepo:           itemRepo,
		sharingRepo:        sharingRepo,
		permissionsService: sharepoint.NewPermissionsService(),
		contentService:     sharepoint.NewContentService(),
	}
}

// GetPermissionAnalysisComponents retrieves raw components needed for permission analysis (audit-scoped).
func (r *PermissionAggregateRepositoryImpl) GetPermissionAnalysisComponents(
	ctx context.Context,
	siteID int64,
	auditRunID int64,
	list *sharepoint.List,
) (*contracts.PermissionAnalysisComponents, error) {
	var components *contracts.PermissionAnalysisComponents

	// Execute within a read transaction for consistency
	err := r.WithReadTx(func(queries *db.Queries) error {
		// Create audit-scoped assignment repository
		scopedAssignmentRepo := NewScopedAssignmentRepository(r.BaseRepository, queries, siteID, auditRunID)
		
		// Get assignments for the list
		assignments, err := scopedAssignmentRepo.GetAssignmentsForObject(ctx, siteID, "list", list.ID)
		if err != nil {
			return fmt.Errorf("failed to get assignments: %w", err)
		}

		// Get sharing links (don't fail if not available)
		sharingLinks, err := r.sharingRepo.GetSharingLinksForList(ctx, siteID, list.ID)
		if err != nil {
			sharingLinks = nil // Continue without sharing links
		}

		// Get items (don't fail if not available)
		items, err := r.itemRepo.GetItemsForList(ctx, siteID, list.ID, 0, 999999) // Get all items
		if err != nil {
			items = nil // Continue without items
		}

		components = &contracts.PermissionAnalysisComponents{
			Assignments:  assignments,
			Items:        items,
			SharingLinks: sharingLinks,
			List:         list,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return components, nil
}
