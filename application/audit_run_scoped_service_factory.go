package application

import (
	"context"
	"fmt"
	"strconv"

	"spaudit/domain/contracts"
	"spaudit/infrastructure/factories"
	"spaudit/infrastructure/repositories"
)

// AuditRunScopedServices contains application services scoped to an audit run.
type AuditRunScopedServices struct {
	SiteContentService  *SiteContentService
	PermissionService   *PermissionService
	SiteBrowsingService *SiteBrowsingService
	AuditRunID          int64
}

// AuditRunScopedServiceFactory creates audit-run-scoped services.
type AuditRunScopedServiceFactory interface {
	CreateForAuditRun(ctx context.Context, siteID int64, auditRunIDStr string) (*AuditRunScopedServices, error)
}

// AuditRunScopedServiceFactoryImpl implements the factory.
type AuditRunScopedServiceFactoryImpl struct {
	repositoryFactory factories.ScopedRepositoryFactory
	baseAuditRepo     contracts.AuditRepository
}

// NewAuditRunScopedServiceFactory creates a new service factory.
func NewAuditRunScopedServiceFactory(
	repositoryFactory factories.ScopedRepositoryFactory,
	baseAuditRepo contracts.AuditRepository,
) AuditRunScopedServiceFactory {
	return &AuditRunScopedServiceFactoryImpl{
		repositoryFactory: repositoryFactory,
		baseAuditRepo:     baseAuditRepo,
	}
}

// CreateForAuditRun creates audit-run-scoped services.
func (f *AuditRunScopedServiceFactoryImpl) CreateForAuditRun(
	ctx context.Context,
	siteID int64,
	auditRunIDStr string,
) (*AuditRunScopedServices, error) {
	
	// Step 1: Resolve audit run ID
	auditRunID, err := f.resolveAuditRunID(ctx, siteID, auditRunIDStr)
	if err != nil {
		return nil, fmt.Errorf("resolve audit run ID: %w", err)
	}

	// Step 2: Create audit-run-scoped repositories through factory
	siteRepo := f.repositoryFactory.CreateScopedSiteRepository(siteID, auditRunID)
	listRepo := f.repositoryFactory.CreateScopedListRepository(siteID, auditRunID)
	itemRepo := f.repositoryFactory.CreateScopedItemRepository(siteID, auditRunID)
	sharingRepo := f.repositoryFactory.CreateScopedSharingRepository(siteID, auditRunID)
	jobRepo := f.repositoryFactory.CreateScopedJobRepository(siteID, auditRunID)

	// Step 4: Create aggregate repositories with audit-run-scoped repo
	siteContentAggregate := repositories.NewSiteContentAggregateRepository(
		f.repositoryFactory.GetBaseRepository(),
		siteRepo,
		listRepo,
		jobRepo,
		itemRepo,
		sharingRepo,
	)
	permissionAggregate := repositories.NewPermissionAggregateRepository(
		f.repositoryFactory.GetBaseRepository(),
		itemRepo,
		sharingRepo,
	)

	// Note: All individual repositories (siteRepo, listRepo, etc.) are now audit-run-scoped for reading

	// Step 5: Create audit-run-scoped application services
	siteContentService := NewAuditScopedSiteContentService(siteContentAggregate, auditRunID)
	permissionService := NewAuditScopedPermissionService(permissionAggregate, auditRunID)
	siteBrowsingService := NewSiteBrowsingService(siteContentAggregate) // Site browsing doesn't need audit scoping

	return &AuditRunScopedServices{
		SiteContentService:  siteContentService,
		PermissionService:   permissionService,
		SiteBrowsingService: siteBrowsingService,
		AuditRunID:          auditRunID,
	}, nil
}

// resolveAuditRunID converts "latest" or numeric string to actual audit run ID
func (f *AuditRunScopedServiceFactoryImpl) resolveAuditRunID(
	ctx context.Context,
	siteID int64,
	auditRunIDStr string,
) (int64, error) {
	
	if auditRunIDStr == "latest" {
		// Get the latest audit run for this site
		latestRun, err := f.repositoryFactory.GetBaseRepository().ReadQueries().GetLatestAuditRunForSite(ctx, siteID)
		if err != nil {
			return 0, fmt.Errorf("get latest audit run for site %d: %w", siteID, err)
		}
		return latestRun.AuditRunID, nil
	}

	// Parse numeric audit run ID
	auditRunID, err := strconv.ParseInt(auditRunIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid audit run ID '%s': %w", auditRunIDStr, err)
	}

	// Validate that this audit run exists for this site
	auditRun, err := f.repositoryFactory.GetBaseRepository().ReadQueries().GetAuditRun(ctx, auditRunID)
	if err != nil {
		return 0, fmt.Errorf("audit run %d not found: %w", auditRunID, err)
	}

	if auditRun.SiteID != siteID {
		return 0, fmt.Errorf("audit run %d belongs to site %d, not site %d", 
			auditRunID, auditRun.SiteID, siteID)
	}

	return auditRunID, nil
}

