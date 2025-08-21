package repositories

import (
	"context"
	"strings"
	"time"

	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
)

// SiteContentAggregateRepositoryImpl implements the site content aggregate repository by composing entity repositories.
type SiteContentAggregateRepositoryImpl struct {
	*BaseRepository
	siteRepo       contracts.SiteRepository
	listRepo       contracts.ListRepository
	jobRepo        contracts.JobRepository
	assignmentRepo contracts.AssignmentRepository
	itemRepo       contracts.ItemRepository
	sharingRepo    contracts.SharingRepository
}

// NewSiteContentAggregateRepository creates a new site content aggregate repository.
func NewSiteContentAggregateRepository(
	base *BaseRepository,
	siteRepo contracts.SiteRepository,
	listRepo contracts.ListRepository,
	jobRepo contracts.JobRepository,
	assignmentRepo contracts.AssignmentRepository,
	itemRepo contracts.ItemRepository,
	sharingRepo contracts.SharingRepository,
) contracts.SiteContentAggregateRepository {
	return &SiteContentAggregateRepositoryImpl{
		BaseRepository: base,
		siteRepo:       siteRepo,
		listRepo:       listRepo,
		jobRepo:        jobRepo,
		assignmentRepo: assignmentRepo,
		itemRepo:       itemRepo,
		sharingRepo:    sharingRepo,
	}
}

// GetSiteWithMetadata retrieves a site with computed metadata including list statistics and audit history.
func (r *SiteContentAggregateRepositoryImpl) GetSiteWithMetadata(ctx context.Context, siteID int64) (*contracts.SiteWithMetadata, error) {
	return r.siteRepo.GetWithMetadata(ctx, siteID)
}

// GetAllSitesWithMetadata retrieves all sites with computed metadata.
func (r *SiteContentAggregateRepositoryImpl) GetAllSitesWithMetadata(ctx context.Context) ([]*contracts.SiteWithMetadata, error) {
	return r.siteRepo.GetAllWithMetadata(ctx)
}

// This method has been removed - service layer will compose the data from individual repository calls

// SearchSites filters sites based on search query using business rules.
func (r *SiteContentAggregateRepositoryImpl) SearchSites(ctx context.Context, searchQuery string) ([]*contracts.SiteWithMetadata, error) {
	allSites, err := r.GetAllSitesWithMetadata(ctx)
	if err != nil {
		return nil, err
	}

	// Return all sites if no search query provided
	if searchQuery == "" {
		return allSites, nil
	}

	// Apply business search filtering logic
	return r.applySiteSearchFilter(allSites, searchQuery), nil
}

// applySiteSearchFilter applies business rules for site search filtering.
func (r *SiteContentAggregateRepositoryImpl) applySiteSearchFilter(sites []*contracts.SiteWithMetadata, searchQuery string) []*contracts.SiteWithMetadata {
	var filteredSites []*contracts.SiteWithMetadata

	// Apply case-insensitive search across title and URL
	searchLower := strings.ToLower(searchQuery)

	for _, site := range sites {
		titleMatch := strings.Contains(strings.ToLower(site.Site.Title), searchLower)
		urlMatch := strings.Contains(strings.ToLower(site.Site.URL), searchLower)

		if titleMatch || urlMatch {
			filteredSites = append(filteredSites, site)
		}
	}

	return filteredSites
}

// GetListByID retrieves a single list by ID.
func (r *SiteContentAggregateRepositoryImpl) GetListByID(ctx context.Context, siteID int64, listID string) (*sharepoint.List, error) {
	return r.listRepo.GetByID(ctx, siteID, listID)
}

// GetListsForSite retrieves all lists for a site.
func (r *SiteContentAggregateRepositoryImpl) GetListsForSite(ctx context.Context, siteID int64) ([]*sharepoint.List, error) {
	return r.listRepo.GetAllForSite(ctx, siteID)
}

// GetListAssignmentsWithRootCause retrieves resolved assignments with root cause analysis for a list.
func (r *SiteContentAggregateRepositoryImpl) GetListAssignmentsWithRootCause(ctx context.Context, siteID int64, listID string) ([]*sharepoint.ResolvedAssignment, error) {
	return r.assignmentRepo.GetResolvedAssignmentsForObject(ctx, siteID, "list", listID)
}

// GetAssignmentsForObject retrieves assignments for any object type.
func (r *SiteContentAggregateRepositoryImpl) GetAssignmentsForObject(ctx context.Context, siteID int64, objectType, objectKey string) ([]*sharepoint.Assignment, error) {
	return r.assignmentRepo.GetAssignmentsForObject(ctx, siteID, objectType, objectKey)
}

// GetListItems retrieves items with unique permissions for a list with pagination.
func (r *SiteContentAggregateRepositoryImpl) GetListItems(ctx context.Context, siteID int64, listID string, offset, limit int) ([]*sharepoint.Item, error) {
	return r.itemRepo.GetItemsWithUniqueForList(ctx, siteID, listID, int64(offset), int64(limit))
}

// GetListSharingLinks retrieves sharing links for a list.
func (r *SiteContentAggregateRepositoryImpl) GetListSharingLinks(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLink, error) {
	return r.sharingRepo.GetSharingLinksForList(ctx, siteID, listID)
}

// GetListSharingLinksWithItemData retrieves sharing links with item data for UI display.
func (r *SiteContentAggregateRepositoryImpl) GetListSharingLinksWithItemData(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLinkWithItemData, error) {
	return r.sharingRepo.GetSharingLinksWithItemDataForList(ctx, siteID, listID)
}

// GetSharingLinkMembers retrieves members for a sharing link.
func (r *SiteContentAggregateRepositoryImpl) GetSharingLinkMembers(ctx context.Context, siteID int64, linkID string) ([]*sharepoint.Principal, error) {
	return r.sharingRepo.GetSharingLinkMembers(ctx, siteID, linkID)
}

// GetLastAuditDate retrieves the last audit date for a site.
func (r *SiteContentAggregateRepositoryImpl) GetLastAuditDate(ctx context.Context, siteID int64) (*time.Time, error) {
	return r.jobRepo.GetLastAuditDate(ctx, siteID)
}
