package helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/mock"

	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/test/mocks"
)

// MockRepositories holds all repository mocks for easy injection
type MockRepositories struct {
	Site       *mocks.MockSiteRepository
	List       *mocks.MockListRepository
	Job        *mocks.MockJobRepository
	Assignment *mocks.MockAssignmentRepository
	Item       *mocks.MockItemRepository
	Sharing    *mocks.MockSharingRepository

	// Aggregate repositories
	SiteContentAggregate *mocks.MockSiteContentAggregateRepository
}

// NewMockRepositories creates a new set of repository mocks
func NewMockRepositories() *MockRepositories {
	return &MockRepositories{
		Site:       &mocks.MockSiteRepository{},
		List:       &mocks.MockListRepository{},
		Job:        &mocks.MockJobRepository{},
		Assignment: &mocks.MockAssignmentRepository{},
		Item:       &mocks.MockItemRepository{},
		Sharing:    &mocks.MockSharingRepository{},

		// Aggregate repositories
		SiteContentAggregate: &mocks.MockSiteContentAggregateRepository{},
	}
}

// GetRepositories returns the repositories for manual service creation
func (m *MockRepositories) GetRepositories() (site, list, job, assignment, item, sharing interface{}) {
	return m.Site, m.List, m.Job, m.Assignment, m.Item, m.Sharing
}

// ExpectSuccessfulSiteRetrieval sets up expectations for a successful site retrieval
func (m *MockRepositories) ExpectSuccessfulSiteRetrieval(siteID int64, site *sharepoint.Site) {
	m.Site.On("GetByID", mock.Anything, siteID).Return(site, nil)
}

// ExpectSuccessfulListRetrieval sets up expectations for successful list retrieval
func (m *MockRepositories) ExpectSuccessfulListRetrieval(siteID int64, lists []*sharepoint.List) {
	m.List.On("GetAllForSite", mock.Anything, siteID).Return(lists, nil)
}

// ExpectNoAuditHistory sets up expectations for no audit history
func (m *MockRepositories) ExpectNoAuditHistory(siteID int64) {
	m.Job.On("GetLastAuditDate", mock.Anything, siteID).Return((*time.Time)(nil), nil)
}

// ExpectAuditHistory sets up expectations for audit history with specific date
func (m *MockRepositories) ExpectAuditHistory(siteID int64, auditDate *time.Time) {
	m.Job.On("GetLastAuditDate", mock.Anything, siteID).Return(auditDate, nil)
}

// ExpectSuccessfulSiteWithMetadata sets up expectations for successful site with metadata retrieval
func (m *MockRepositories) ExpectSuccessfulSiteWithMetadata(siteID int64, result *contracts.SiteWithMetadata) {
	m.SiteContentAggregate.On("GetSiteWithMetadata", mock.Anything, siteID).Return(result, nil)
}

// ExpectSuccessfulListsForSite sets up expectations for successful lists retrieval
func (m *MockRepositories) ExpectSuccessfulListsForSite(siteID int64, lists []*sharepoint.List) {
	m.SiteContentAggregate.On("GetListsForSite", mock.Anything, siteID).Return(lists, nil)
}

// ExpectLastAuditDate sets up expectations for last audit date retrieval
func (m *MockRepositories) ExpectLastAuditDate(siteID int64, auditDate *time.Time) {
	m.SiteContentAggregate.On("GetLastAuditDate", mock.Anything, siteID).Return(auditDate, nil)
}

// AssertAllExpectations verifies all mock expectations were met
func (m *MockRepositories) AssertAllExpectations(t mock.TestingT) {
	m.Site.AssertExpectations(t)
	m.List.AssertExpectations(t)
	m.Job.AssertExpectations(t)
	m.Assignment.AssertExpectations(t)
	m.Item.AssertExpectations(t)
	m.Sharing.AssertExpectations(t)
	m.SiteContentAggregate.AssertExpectations(t)
}

// TestData provides simple builders for test data
type TestData struct{}

// NewTestData creates a test data builder
func NewTestData() *TestData {
	return &TestData{}
}

// SimpleSite creates a basic site for testing
func (td *TestData) SimpleSite(id int64, title string) *sharepoint.Site {
	return &sharepoint.Site{
		ID:    id,
		Title: title,
		URL:   "https://test.sharepoint.com",
	}
}

// SimpleList creates a basic list for testing
func (td *TestData) SimpleList(id string, hasUnique bool, itemCount int) *sharepoint.List {
	return &sharepoint.List{
		ID:        id,
		SiteID:    1,
		Title:     "List " + id,
		ItemCount: itemCount,
		HasUnique: hasUnique,
	}
}

// ListsWithStats creates a set of lists with known statistics for testing business logic
func (td *TestData) ListsWithStats(totalUnique int, totalItems int64) []*sharepoint.List {
	lists := make([]*sharepoint.List, 0)

	// Create lists with unique permissions up to totalUnique
	for i := 0; i < totalUnique; i++ {
		lists = append(lists, td.SimpleList(fmt.Sprintf("unique-%d", i), true, int(totalItems/int64(totalUnique))))
	}

	// Add one list without unique permissions if we have remaining items
	if len(lists) == 0 || totalItems > int64(totalUnique)*int64(lists[0].ItemCount) {
		remaining := int(totalItems) - (totalUnique * lists[0].ItemCount)
		if remaining > 0 {
			lists = append(lists, td.SimpleList("normal", false, remaining))
		}
	}

	return lists
}

// Helper for common test context
func TestContext() context.Context {
	return context.Background()
}

// Helper for time-based tests
func TestTime(daysAgo int) *time.Time {
	t := time.Now().AddDate(0, 0, -daysAgo)
	return &t
}
