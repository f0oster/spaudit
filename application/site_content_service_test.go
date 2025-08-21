package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/test/helpers"
)

func TestSiteContentService_GetSiteWithLists_Success(t *testing.T) {
	// Arrange
	mocks := helpers.NewMockRepositories()
	testData := helpers.NewTestData()

	expectedSite := testData.SimpleSite(1, "Test Site")
	expectedLists := []*sharepoint.List{
		testData.SimpleList("list1", true, 10), // Has unique, 10 items
		testData.SimpleList("list2", false, 5), // No unique, 5 items
	}
	auditDate := helpers.TestTime(3) // 3 days ago

	expectedSiteWithMeta := &contracts.SiteWithMetadata{
		Site:             expectedSite,
		TotalLists:       2,
		ListsWithUnique:  1,
		LastAuditDate:    auditDate,
		LastAuditDaysAgo: 3,
	}

	mocks.ExpectSuccessfulSiteWithMetadata(1, expectedSiteWithMeta)
	mocks.ExpectSuccessfulListsForSite(1, expectedLists)
	mocks.ExpectLastAuditDate(1, auditDate)

	service := NewSiteContentService(mocks.SiteContentAggregate)

	// Act
	result, err := service.GetSiteWithLists(context.Background(), 1)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)

	// Test business outcomes (not implementation details)
	assert.Equal(t, expectedSite.Title, result.Site.Title)
	assert.Equal(t, 2, result.TotalLists)
	assert.Equal(t, 1, result.ListsWithUnique)    // Only list1 has unique
	assert.Equal(t, int64(15), result.TotalItems) // 10 + 5
	assert.Equal(t, 3, result.LastAuditDaysAgo)

	mocks.AssertAllExpectations(t)
}

func TestSiteContentService_GetSiteWithLists_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*helpers.MockRepositories)
		expectedError string
	}{
		{
			name: "site_not_found",
			setupMocks: func(m *helpers.MockRepositories) {
				m.SiteContentAggregate.On("GetSiteWithMetadata", context.Background(), int64(1)).Return((*contracts.SiteWithMetadata)(nil), errors.New("site not found"))
			},
			expectedError: "site not found",
		},
		{
			name: "lists_fetch_fails",
			setupMocks: func(m *helpers.MockRepositories) {
				expectedSiteWithMeta := &contracts.SiteWithMetadata{
					Site: helpers.NewTestData().SimpleSite(1, "Test"),
				}
				m.SiteContentAggregate.On("GetSiteWithMetadata", context.Background(), int64(1)).Return(expectedSiteWithMeta, nil)
				m.SiteContentAggregate.On("GetListsForSite", context.Background(), int64(1)).Return(([]*sharepoint.List)(nil), errors.New("database error"))
			},
			expectedError: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := helpers.NewMockRepositories()
			tt.setupMocks(mocks)

			service := NewSiteContentService(mocks.SiteContentAggregate)
			result, err := service.GetSiteWithLists(context.Background(), 1)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestSiteContentService_GetSiteWithLists_BusinessLogic(t *testing.T) {
	tests := []struct {
		name           string
		lists          []*sharepoint.List
		expectedTotal  int
		expectedUnique int
		expectedItems  int64
	}{
		{
			name: "mixed_lists",
			lists: []*sharepoint.List{
				helpers.NewTestData().SimpleList("1", true, 10),
				helpers.NewTestData().SimpleList("2", false, 5),
				helpers.NewTestData().SimpleList("3", true, 0), // Edge case: unique but empty
			},
			expectedTotal: 3, expectedUnique: 2, expectedItems: 15,
		},
		{
			name: "all_unique",
			lists: []*sharepoint.List{
				helpers.NewTestData().SimpleList("1", true, 100),
				helpers.NewTestData().SimpleList("2", true, 200),
			},
			expectedTotal: 2, expectedUnique: 2, expectedItems: 300,
		},
		{
			name: "no_unique",
			lists: []*sharepoint.List{
				helpers.NewTestData().SimpleList("1", false, 50),
			},
			expectedTotal: 1, expectedUnique: 0, expectedItems: 50,
		},
		{
			name:          "empty_site",
			lists:         []*sharepoint.List{},
			expectedTotal: 0, expectedUnique: 0, expectedItems: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := helpers.NewMockRepositories()

			expectedSiteWithMeta := &contracts.SiteWithMetadata{
				Site: helpers.NewTestData().SimpleSite(1, "Test"),
			}
			mocks.ExpectSuccessfulSiteWithMetadata(1, expectedSiteWithMeta)
			mocks.ExpectSuccessfulListsForSite(1, tt.lists)
			mocks.ExpectLastAuditDate(1, nil)

			service := NewSiteContentService(mocks.SiteContentAggregate)
			result, err := service.GetSiteWithLists(context.Background(), 1)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedTotal, result.TotalLists)
			assert.Equal(t, tt.expectedUnique, result.ListsWithUnique)
			assert.Equal(t, tt.expectedItems, result.TotalItems)
		})
	}
}

func TestSiteContentService_GetSiteWithLists_AuditDateHandling(t *testing.T) {
	tests := []struct {
		name            string
		setupAuditDate  func(*helpers.MockRepositories)
		expectedDaysAgo int
		hasAuditDate    bool
	}{
		{
			name: "recent_audit",
			setupAuditDate: func(m *helpers.MockRepositories) {
				m.ExpectAuditHistory(1, helpers.TestTime(1)) // 1 day ago
			},
			expectedDaysAgo: 1,
			hasAuditDate:    true,
		},
		{
			name: "old_audit",
			setupAuditDate: func(m *helpers.MockRepositories) {
				m.ExpectAuditHistory(1, helpers.TestTime(30)) // 30 days ago
			},
			expectedDaysAgo: 30,
			hasAuditDate:    true,
		},
		{
			name: "no_audit_history",
			setupAuditDate: func(m *helpers.MockRepositories) {
				m.ExpectNoAuditHistory(1)
			},
			expectedDaysAgo: 0,
			hasAuditDate:    false,
		},
		{
			name: "audit_query_fails_gracefully",
			setupAuditDate: func(m *helpers.MockRepositories) {
				m.Job.On("GetLastAuditDate", context.Background(), int64(1)).Return((*time.Time)(nil), errors.New("query failed"))
			},
			expectedDaysAgo: 0,
			hasAuditDate:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := helpers.NewMockRepositories()

			lists := []*sharepoint.List{
				helpers.NewTestData().SimpleList("1", false, 5),
			}

			var auditDate *time.Time
			if tt.hasAuditDate {
				if tt.name == "recent_audit" {
					auditDate = helpers.TestTime(1)
				} else if tt.name == "old_audit" {
					auditDate = helpers.TestTime(30)
				}
			}

			expectedSiteWithMeta := &contracts.SiteWithMetadata{
				Site: helpers.NewTestData().SimpleSite(1, "Test"),
			}
			mocks.ExpectSuccessfulSiteWithMetadata(1, expectedSiteWithMeta)
			mocks.ExpectSuccessfulListsForSite(1, lists)
			mocks.ExpectLastAuditDate(1, auditDate)

			service := NewSiteContentService(mocks.SiteContentAggregate)
			result, err := service.GetSiteWithLists(context.Background(), 1)

			require.NoError(t, err) // Should not fail even if audit date query fails
			assert.Equal(t, tt.expectedDaysAgo, result.LastAuditDaysAgo)

			if tt.hasAuditDate {
				assert.NotNil(t, result.LastAuditDate)
			} else {
				assert.Nil(t, result.LastAuditDate)
			}
		})
	}
}
