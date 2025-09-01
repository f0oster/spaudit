package application

import (
	"context"
	"time"

	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
)

// SiteWithListsData represents a site and its lists with computed statistics.
type SiteWithListsData struct {
	Site             *sharepoint.Site
	Lists            []*sharepoint.List
	TotalLists       int
	ListsWithUnique  int
	TotalItems       int64
	LastAuditDate    *time.Time
	LastAuditDaysAgo int
	AuditRunID       int64
}

// SiteContentService handles site content operations.
type SiteContentService struct {
	contentAggregate contracts.SiteContentAggregateRepository
	auditRunID       int64 // For audit-scoped operations
}

// NewSiteContentService creates a new site content service.
func NewSiteContentService(
	contentAggregate contracts.SiteContentAggregateRepository,
) *SiteContentService {
	return newSiteContentService(contentAggregate, 0) // 0 means no specific audit run
}

// NewAuditScopedSiteContentService creates a site content service scoped to a specific audit run.
func NewAuditScopedSiteContentService(
	contentAggregate contracts.SiteContentAggregateRepository,
	auditRunID int64,
) *SiteContentService {
	return newSiteContentService(contentAggregate, auditRunID)
}

// newSiteContentService is the common constructor.
func newSiteContentService(
	contentAggregate contracts.SiteContentAggregateRepository,
	auditRunID int64,
) *SiteContentService {
	return &SiteContentService{
		contentAggregate: contentAggregate,
		auditRunID:       auditRunID,
	}
}

// GetSiteWithLists retrieves a site and its lists with statistics.
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

// GetListAssignmentsWithRootCause retrieves resolved assignments with root cause analysis for a list (audit-scoped).
func (s *SiteContentService) GetListAssignmentsWithRootCause(ctx context.Context, siteID int64, listID string) ([]*sharepoint.ResolvedAssignment, error) {
	return s.contentAggregate.GetListAssignmentsWithRootCause(ctx, siteID, s.auditRunID, listID)
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

// GetAssignmentsForObject retrieves assignments for any object type (audit-scoped).
func (s *SiteContentService) GetAssignmentsForObject(ctx context.Context, siteID int64, objectType, objectKey string) ([]*sharepoint.Assignment, error) {
	return s.contentAggregate.GetAssignmentsForObject(ctx, siteID, s.auditRunID, objectType, objectKey)
}

// GetSharingLinkMembers retrieves members for a sharing link.
func (s *SiteContentService) GetSharingLinkMembers(ctx context.Context, siteID int64, linkID string) ([]*sharepoint.Principal, error) {
	return s.contentAggregate.GetSharingLinkMembers(ctx, siteID, linkID)
}
