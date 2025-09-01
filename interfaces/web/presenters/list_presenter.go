// Package presenters transforms domain data into UI-ready view models.
package presenters

import (
	"fmt"
	"strings"
	"time"

	"spaudit/application"
	"spaudit/domain/sharepoint"
)

// AuditRunOption represents an audit run option for UI selection
type AuditRunOption struct {
	ID        int64     `json:"id"`
	StartedAt time.Time `json:"started_at"`
	Status    string    `json:"status"`
}

// SiteListsVM is the view model for the site lists page
type SiteListsVM struct {
	Site            SiteWithMetadata
	Lists           []ListSummary
	TotalLists      int
	ListsWithUnique int
	TotalItems      int
	AuditRunID      int64
	AuditRuns       []AuditRunOption
}

// ListPresenter transforms site and list data for UI display.
// TODO: Add pagination support to presenter layer:
// - Create PaginatedItemsResponse struct with Items []ItemSummary + PaginationMeta
// - Add MapItemsWithPagination method for transforming paginated business data
// - Include pagination controls data (current page, total pages, has next/prev)
// - Add client-side state management for infinite scroll or traditional pagination
// - Consider performance optimizations for large lists (virtual scrolling, lazy loading)
type ListPresenter struct{}

// NewListPresenter creates a list presenter.
func NewListPresenter() *ListPresenter {
	return &ListPresenter{}
}

// ToSiteListsViewModel converts site data to UI view model with formatted dates and statistics.
// Returns safe defaults if data is nil.
func (p *ListPresenter) ToSiteListsViewModel(data *application.SiteWithListsData) *SiteListsVM {
	if data == nil {
		return &SiteListsVM{
			Site:            SiteWithMetadata{},
			Lists:           []ListSummary{},
			TotalLists:      0,
			ListsWithUnique: 0,
			TotalItems:      0,
			AuditRunID:      0,
			AuditRuns:       []AuditRunOption{},
		}
	}

	return &SiteListsVM{
		Site:            p.toSiteWithMetadata(data),
		Lists:           p.toListSummaries(data.Lists),
		TotalLists:      data.TotalLists,
		ListsWithUnique: data.ListsWithUnique,
		TotalItems:      int(data.TotalItems),
		AuditRunID:      data.AuditRunID,
		AuditRuns:       []AuditRunOption{}, // Will be populated by handler
	}
}

// toSiteWithMetadata converts service data to site metadata with formatted audit dates.
func (p *ListPresenter) toSiteWithMetadata(data *application.SiteWithListsData) SiteWithMetadata {
	if data == nil {
		return SiteWithMetadata{}
	}

	lastAuditDate := p.formatRelativeDate(data.LastAuditDaysAgo, data.LastAuditDate)

	// Handle nil site gracefully
	var siteID int64
	var siteURL, title string
	if data.Site != nil {
		siteID = data.Site.ID
		siteURL = data.Site.URL
		title = data.Site.Title
	}

	return SiteWithMetadata{
		SiteID:          siteID,
		SiteURL:         siteURL,
		Title:           title,
		Description:     "", // TODO: Investigate if description should be provided by service
		TotalLists:      data.TotalLists,
		ListsWithUnique: data.ListsWithUnique,
		LastAuditDate:   lastAuditDate,
		DaysAgo:         data.LastAuditDaysAgo,
	}
}

// toListSummaries converts domain lists to view models with formatted timestamps.
func (p *ListPresenter) toListSummaries(domainLists []*sharepoint.List) []ListSummary {
	summaries := make([]ListSummary, len(domainLists))

	for i, list := range domainLists {
		var auditRunID int64
		if list.AuditRunID != nil {
			auditRunID = *list.AuditRunID
		}
		
		summaries[i] = ListSummary{
			SiteID:       list.SiteID,
			SiteURL:      "", // TODO: Investigate if SiteURL should be available in domain model
			ListID:       list.ID,
			WebID:        list.WebID,
			Title:        list.Title,
			URL:          list.URL,
			ItemCount:    int64(list.ItemCount),
			HasUnique:    list.HasUnique,
			WebTitle:     "", // TODO: Add WebTitle to sharepoint.List or fetch separately
			LastModified: p.formatAuditRunID(list.AuditRunID),
			AuditRunID:   auditRunID,
		}
	}

	return summaries
}

// ToListSummaries converts domain lists to view models with formatted timestamps.
func (p *ListPresenter) ToListSummaries(domainLists []*sharepoint.List) []ListSummary {
	return p.toListSummaries(domainLists)
}

// FilterListsForSearch filters lists by search query across title, URL, and web title.
// Case-insensitive matching. Returns all lists if query is empty.
func (p *ListPresenter) FilterListsForSearch(lists []ListSummary, searchQuery string) []ListSummary {
	if strings.TrimSpace(searchQuery) == "" {
		return lists
	}

	var filteredLists []ListSummary
	searchLower := strings.ToLower(strings.TrimSpace(searchQuery))

	for _, list := range lists {
		titleMatch := strings.Contains(strings.ToLower(list.Title), searchLower)
		urlMatch := strings.Contains(strings.ToLower(list.URL), searchLower)
		webTitleMatch := strings.Contains(strings.ToLower(list.WebTitle), searchLower)

		if titleMatch || urlMatch || webTitleMatch {
			filteredLists = append(filteredLists, list)
		}
	}

	return filteredLists
}


// formatRelativeDate formats audit dates as relative time (e.g., "5 days ago", "Today").
func (p *ListPresenter) formatRelativeDate(daysAgo int, auditDate *time.Time) string {
	if auditDate == nil {
		return "Never audited"
	}

	if daysAgo == 0 {
		return "Today"
	} else if daysAgo == 1 {
		return "1 day ago"
	} else {
		return fmt.Sprintf("%d days ago", daysAgo)
	}
}

// formatAuditRunID formats audit run ID for display.
// Returns empty string if audit run ID is nil (never audited).
func (p *ListPresenter) formatAuditRunID(auditRunID *int64) string {
	if auditRunID == nil {
		return ""
	}
	return fmt.Sprintf("Run #%d", *auditRunID)
}
