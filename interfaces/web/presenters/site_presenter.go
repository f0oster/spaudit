package presenters

import (
	"spaudit/domain/contracts"
)

// Site-related view data structures

// SiteSelectionVM is the view model for the site selection page
type SiteSelectionVM struct {
	Sites         []SiteWithMetadata
	HasActiveJobs bool
}

// SitePresenter transforms site service data into UI-ready view models.
type SitePresenter struct{}

// NewSitePresenter creates a new site presenter.
func NewSitePresenter() *SitePresenter {
	return &SitePresenter{}
}

// ToSiteSelectionViewModel converts service data to site selection view model.
func (p *SitePresenter) ToSiteSelectionViewModel(sitesData []*contracts.SiteWithMetadata, hasActiveJobs bool) *SiteSelectionVM {
	return &SiteSelectionVM{
		Sites:         p.ToSitesWithMetadata(sitesData),
		HasActiveJobs: hasActiveJobs,
	}
}

// ToSitesWithMetadata converts service data to view model collection.
func (p *SitePresenter) ToSitesWithMetadata(sitesData []*contracts.SiteWithMetadata) []SiteWithMetadata {
	viewModels := make([]SiteWithMetadata, len(sitesData))

	for i, siteData := range sitesData {
		viewModels[i] = p.toSiteWithMetadata(siteData)
	}

	return viewModels
}


// toSiteWithMetadata converts single service data to view model with formatted audit date.
func (p *SitePresenter) toSiteWithMetadata(siteData *contracts.SiteWithMetadata) SiteWithMetadata {
	lastAuditDate := ""
	if siteData.LastAuditDate != nil {
		lastAuditDate = siteData.LastAuditDate.Format("Jan 2, 2006")
	}

	return SiteWithMetadata{
		SiteID:          siteData.Site.ID,
		SiteURL:         siteData.Site.URL,
		Title:           siteData.Site.Title,
		Description:     "", // Description field not available in domain model
		TotalLists:      siteData.TotalLists,
		ListsWithUnique: siteData.ListsWithUnique,
		LastAuditDate:   lastAuditDate,
		DaysAgo:         siteData.LastAuditDaysAgo,
	}
}
