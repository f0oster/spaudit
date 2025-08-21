package application

import (
	"context"
	"time"

	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
)

// SiteWithListsData represents the business data for a site and its lists with computed statistics.
type SiteWithListsData struct {
	Site             *sharepoint.Site
	Lists            []*sharepoint.List
	TotalLists       int
	ListsWithUnique  int
	TotalItems       int64
	LastAuditDate    *time.Time
	LastAuditDaysAgo int
}

// SiteContentService handles business logic for site content hierarchy operations using aggregate repository.
type SiteContentService struct {
	contentAggregate contracts.SiteContentAggregateRepository
}

// NewSiteContentService creates a new site content service with aggregate repository dependency injection.
func NewSiteContentService(
	contentAggregate contracts.SiteContentAggregateRepository,
) *SiteContentService {
	return &SiteContentService{
		contentAggregate: contentAggregate,
	}
}

// GetSiteWithLists retrieves a site and all its lists with computed statistics and audit metadata.
func (s *SiteContentService) GetSiteWithLists(ctx context.Context, siteID int64) (*SiteWithListsData, error) {
	// Get site with metadata from aggregate repository
	siteWithMeta, err := s.contentAggregate.GetSiteWithMetadata(ctx, siteID)
	if err != nil {
		return nil, err
	}

	// Get lists for the site
	lists, err := s.contentAggregate.GetListsForSite(ctx, siteID)
	if err != nil {
		return nil, err
	}

	// Get last audit date
	lastAuditDate, err := s.contentAggregate.GetLastAuditDate(ctx, siteID)
	if err != nil {
		// Continue if audit date lookup fails - not critical for operation
		lastAuditDate = nil
	}

	// Calculate business statistics from list data
	totalItems := int64(0)
	listsWithUnique := 0

	for _, list := range lists {
		if list.HasUnique {
			listsWithUnique++
		}
		totalItems += int64(list.ItemCount)
	}

	// Calculate days since last audit for display
	lastAuditDaysAgo := 0
	if lastAuditDate != nil {
		lastAuditDaysAgo = int(time.Since(*lastAuditDate).Hours() / 24)
	}

	return &SiteWithListsData{
		Site:             siteWithMeta.Site,
		Lists:            lists,
		TotalLists:       len(lists),
		ListsWithUnique:  listsWithUnique,
		TotalItems:       totalItems,
		LastAuditDate:    lastAuditDate,
		LastAuditDaysAgo: lastAuditDaysAgo,
	}, nil
}

// GetListByID retrieves a single list by ID.
func (s *SiteContentService) GetListByID(ctx context.Context, siteID int64, listID string) (*sharepoint.List, error) {
	return s.contentAggregate.GetListByID(ctx, siteID, listID)
}

// GetListsForSite retrieves all lists for a site.
func (s *SiteContentService) GetListsForSite(ctx context.Context, siteID int64) ([]*sharepoint.List, error) {
	return s.contentAggregate.GetListsForSite(ctx, siteID)
}

// GetListAssignmentsWithRootCause retrieves resolved assignments with root cause analysis for a list.
func (s *SiteContentService) GetListAssignmentsWithRootCause(ctx context.Context, siteID int64, listID string) ([]*sharepoint.ResolvedAssignment, error) {
	return s.contentAggregate.GetListAssignmentsWithRootCause(ctx, siteID, listID)
}

// GetListItems retrieves items with unique permissions for a list with pagination.
func (s *SiteContentService) GetListItems(ctx context.Context, siteID int64, listID string, offset, limit int) ([]*sharepoint.Item, error) {
	return s.contentAggregate.GetListItems(ctx, siteID, listID, offset, limit)
}

// GetListSharingLinks retrieves sharing links for a list.
func (s *SiteContentService) GetListSharingLinks(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLink, error) {
	return s.contentAggregate.GetListSharingLinks(ctx, siteID, listID)
}

// GetListSharingLinksWithItemData retrieves sharing links with item data for UI display.
func (s *SiteContentService) GetListSharingLinksWithItemData(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLinkWithItemData, error) {
	return s.contentAggregate.GetListSharingLinksWithItemData(ctx, siteID, listID)
}

// GetAssignmentsForObject retrieves assignments for any object type.
func (s *SiteContentService) GetAssignmentsForObject(ctx context.Context, siteID int64, objectType, objectKey string) ([]*sharepoint.Assignment, error) {
	return s.contentAggregate.GetAssignmentsForObject(ctx, siteID, objectType, objectKey)
}

// GetSharingLinkMembers retrieves members for a sharing link.
func (s *SiteContentService) GetSharingLinkMembers(ctx context.Context, siteID int64, linkID string) ([]*sharepoint.Principal, error) {
	return s.contentAggregate.GetSharingLinkMembers(ctx, siteID, linkID)
}
