package factories

import (
	"spaudit/database"
	"spaudit/domain/contracts"
	"spaudit/infrastructure/repositories"
)

// ScopedRepositoryFactory creates audit-run-scoped repositories
type ScopedRepositoryFactory interface {
	GetBaseRepository() *repositories.BaseRepository
	CreateScopedSiteRepository(siteID, auditRunID int64) contracts.SiteRepository
	CreateScopedListRepository(siteID, auditRunID int64) contracts.ListRepository
	CreateScopedItemRepository(siteID, auditRunID int64) contracts.ItemRepository
	CreateScopedSharingRepository(siteID, auditRunID int64) contracts.SharingRepository
	CreateScopedJobRepository(siteID, auditRunID int64) contracts.JobRepository
	CreateScopedAssignmentRepository(siteID, auditRunID int64) contracts.AssignmentRepository
}

// ScopedRepositoryFactoryImpl implements the factory
type ScopedRepositoryFactoryImpl struct {
	db       *database.Database
	baseRepo *repositories.BaseRepository
}

// NewScopedRepositoryFactory creates a new repository factory
func NewScopedRepositoryFactory(db *database.Database) ScopedRepositoryFactory {
	return &ScopedRepositoryFactoryImpl{
		db:       db,
		baseRepo: repositories.NewBaseRepository(db),
	}
}

// GetBaseRepository returns the base repository for aggregate repository creation
func (f *ScopedRepositoryFactoryImpl) GetBaseRepository() *repositories.BaseRepository {
	return f.baseRepo
}

// CreateScopedSiteRepository creates an audit-run-scoped site repository
func (f *ScopedRepositoryFactoryImpl) CreateScopedSiteRepository(siteID, auditRunID int64) contracts.SiteRepository {
	return repositories.NewScopedSiteRepository(f.baseRepo, f.db.Queries(), siteID, auditRunID)
}

// CreateScopedListRepository creates an audit-run-scoped list repository
func (f *ScopedRepositoryFactoryImpl) CreateScopedListRepository(siteID, auditRunID int64) contracts.ListRepository {
	return repositories.NewScopedListRepository(f.baseRepo, f.db.Queries(), siteID, auditRunID)
}

// CreateScopedItemRepository creates an audit-run-scoped item repository
func (f *ScopedRepositoryFactoryImpl) CreateScopedItemRepository(siteID, auditRunID int64) contracts.ItemRepository {
	return repositories.NewScopedItemRepository(f.baseRepo, f.db.Queries(), siteID, auditRunID)
}

// CreateScopedSharingRepository creates an audit-run-scoped sharing repository
func (f *ScopedRepositoryFactoryImpl) CreateScopedSharingRepository(siteID, auditRunID int64) contracts.SharingRepository {
	return repositories.NewScopedSharingRepository(f.baseRepo, f.db.Queries(), siteID, auditRunID)
}

// CreateScopedJobRepository creates an audit-run-scoped job repository
func (f *ScopedRepositoryFactoryImpl) CreateScopedJobRepository(siteID, auditRunID int64) contracts.JobRepository {
	return repositories.NewScopedJobRepository(f.baseRepo, f.db.Queries(), siteID, auditRunID)
}

// CreateScopedAssignmentRepository creates an audit-run-scoped assignment repository
func (f *ScopedRepositoryFactoryImpl) CreateScopedAssignmentRepository(siteID, auditRunID int64) contracts.AssignmentRepository {
	return repositories.NewScopedAssignmentRepository(f.baseRepo, f.db.Queries(), siteID, auditRunID)
}