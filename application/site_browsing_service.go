package application

import (
	"context"

	"spaudit/domain/contracts"
)

// SiteBrowsingService handles site browsing and selection.
type SiteBrowsingService struct {
	contentAggregate contracts.SiteContentAggregateRepository
}

// NewSiteBrowsingService creates a new site browsing service.
func NewSiteBrowsingService(contentAggregate contracts.SiteContentAggregateRepository) *SiteBrowsingService {
	return &SiteBrowsingService{
		contentAggregate: contentAggregate,
	}
}

// GetAllSitesWithMetadata retrieves all sites with metadata.
func (s *SiteBrowsingService) GetAllSitesWithMetadata(ctx context.Context) ([]*contracts.SiteWithMetadata, error) {
	return s.contentAggregate.GetAllSitesWithMetadata(ctx)
}

// GetSiteWithMetadata retrieves a site with metadata.
func (s *SiteBrowsingService) GetSiteWithMetadata(ctx context.Context, siteID int64) (*contracts.SiteWithMetadata, error) {
	return s.contentAggregate.GetSiteWithMetadata(ctx, siteID)
}

// SearchSites filters sites by search query.
func (s *SiteBrowsingService) SearchSites(ctx context.Context, searchQuery string) ([]*contracts.SiteWithMetadata, error) {
	return s.contentAggregate.SearchSites(ctx, searchQuery)
}
