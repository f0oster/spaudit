package application

import (
	"context"

	"spaudit/domain/contracts"
)

// SiteBrowsingService handles site browsing and selection business logic using aggregate repository.
type SiteBrowsingService struct {
	contentAggregate contracts.SiteContentAggregateRepository
}

// NewSiteBrowsingService creates a new site browsing service with aggregate repository dependency injection.
func NewSiteBrowsingService(contentAggregate contracts.SiteContentAggregateRepository) *SiteBrowsingService {
	return &SiteBrowsingService{
		contentAggregate: contentAggregate,
	}
}

// GetAllSitesWithMetadata retrieves all sites with computed metadata using aggregate repository.
func (s *SiteBrowsingService) GetAllSitesWithMetadata(ctx context.Context) ([]*contracts.SiteWithMetadata, error) {
	return s.contentAggregate.GetAllSitesWithMetadata(ctx)
}

// GetSiteWithMetadata retrieves a single site with computed metadata using aggregate repository.
func (s *SiteBrowsingService) GetSiteWithMetadata(ctx context.Context, siteID int64) (*contracts.SiteWithMetadata, error) {
	return s.contentAggregate.GetSiteWithMetadata(ctx, siteID)
}

// SearchSites filters sites based on search query using business rules.
func (s *SiteBrowsingService) SearchSites(ctx context.Context, searchQuery string) ([]*contracts.SiteWithMetadata, error) {
	return s.contentAggregate.SearchSites(ctx, searchQuery)
}
