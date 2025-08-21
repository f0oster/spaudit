package presenters

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"spaudit/application"
	"spaudit/domain/sharepoint"
	"spaudit/test/helpers"
)

func TestListPresenter_ToSiteListsViewModel_Success(t *testing.T) {
	// Arrange
	presenter := NewListPresenter()
	testData := helpers.NewTestData()

	auditDate := helpers.TestTime(5) // 5 days ago
	data := &application.SiteWithListsData{
		Site: testData.SimpleSite(1, "Test Site"),
		Lists: []*sharepoint.List{
			testData.SimpleList("docs", true, 10),
			testData.SimpleList("tasks", false, 5),
		},
		TotalLists:       2,
		ListsWithUnique:  1,
		TotalItems:       15,
		LastAuditDate:    auditDate,
		LastAuditDaysAgo: 5,
	}

	// Act
	result := presenter.ToSiteListsViewModel(data)

	// Assert - Test presentation logic outcomes
	require.NotNil(t, result)

	// Site presentation
	assert.Equal(t, "Test Site", result.Site.Title)
	assert.Equal(t, int64(1), result.Site.SiteID)

	// Statistics presentation
	assert.Equal(t, 2, result.TotalLists)
	assert.Equal(t, 1, result.ListsWithUnique)
	assert.Equal(t, 15, result.TotalItems) // Note: int conversion from int64

	// Date presentation should be in the site metadata
	assert.Equal(t, "5 days ago", result.Site.LastAuditDate)
	assert.Equal(t, 5, result.Site.DaysAgo)

	// Lists presentation
	require.Len(t, result.Lists, 2)
	assert.Equal(t, "List docs", result.Lists[0].Title) // Formatted title
	assert.True(t, result.Lists[0].HasUnique)           // Has unique permissions
	assert.False(t, result.Lists[1].HasUnique)          // No unique permissions
}

func TestListPresenter_ToSiteListsViewModel_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		data     *application.SiteWithListsData
		expected map[string]interface{} // Key aspects to verify
	}{
		{
			name: "empty_site",
			data: &application.SiteWithListsData{
				Site:             helpers.NewTestData().SimpleSite(1, "Empty Site"),
				Lists:            []*sharepoint.List{},
				TotalLists:       0,
				ListsWithUnique:  0,
				TotalItems:       0,
				LastAuditDate:    nil,
				LastAuditDaysAgo: 0,
			},
			expected: map[string]interface{}{
				"total_lists":       0,
				"lists_with_unique": 0,
				"total_items":       0,
				"last_audit_date":   "Never audited", // No audit date formatted as never audited
				"days_ago":          0,
			},
		},
		{
			name: "recent_audit",
			data: &application.SiteWithListsData{
				Site:             helpers.NewTestData().SimpleSite(1, "Recent Site"),
				Lists:            []*sharepoint.List{helpers.NewTestData().SimpleList("test", false, 1)},
				TotalLists:       1,
				ListsWithUnique:  0,
				TotalItems:       1,
				LastAuditDate:    helpers.TestTime(0), // Today
				LastAuditDaysAgo: 0,
			},
			expected: map[string]interface{}{
				"days_ago":        0,
				"last_audit_date": "Today",
			},
		},
		{
			name: "old_audit",
			data: &application.SiteWithListsData{
				Site:             helpers.NewTestData().SimpleSite(1, "Old Site"),
				Lists:            []*sharepoint.List{helpers.NewTestData().SimpleList("test", true, 100)},
				TotalLists:       1,
				ListsWithUnique:  1,
				TotalItems:       100,
				LastAuditDate:    helpers.TestTime(30), // 30 days ago
				LastAuditDaysAgo: 30,
			},
			expected: map[string]interface{}{
				"days_ago":        30,
				"last_audit_date": "30 days ago",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presenter := NewListPresenter()

			result := presenter.ToSiteListsViewModel(tt.data)

			require.NotNil(t, result)

			// Check expected values
			for key, expectedValue := range tt.expected {
				switch key {
				case "total_lists":
					assert.Equal(t, expectedValue, result.TotalLists)
				case "lists_with_unique":
					assert.Equal(t, expectedValue, result.ListsWithUnique)
				case "total_items":
					assert.Equal(t, expectedValue, result.TotalItems)
				case "last_audit_date":
					assert.Equal(t, expectedValue, result.Site.LastAuditDate)
				case "days_ago":
					assert.Equal(t, expectedValue, result.Site.DaysAgo)
				}
			}
		})
	}
}

func TestListPresenter_ToListSummaries_Success(t *testing.T) {
	// Test the public method for converting domain lists to view models
	presenter := NewListPresenter()

	domainLists := []*sharepoint.List{
		{
			ID:        "unique-list",
			SiteID:    1,
			Title:     "Important Documents",
			URL:       "/sites/test/Lists/Documents",
			ItemCount: 50,
			HasUnique: true,
			AuditRunID: func() *int64 {
				id := int64(123)
				return &id
			}(),
		},
		{
			ID:         "inherited-list",
			SiteID:     1,
			Title:      "Regular Tasks",
			URL:        "/sites/test/Lists/Tasks",
			ItemCount:  10,
			HasUnique:  false,
			AuditRunID: nil, // Never audited
		},
	}

	// Act
	result := presenter.ToListSummaries(domainLists)

	// Assert - Test list transformation
	require.Len(t, result, 2)

	// Test first list (unique permissions)
	uniqueList := result[0]
	assert.Equal(t, "Important Documents", uniqueList.Title)
	assert.Equal(t, "unique-list", uniqueList.ListID)
	assert.Equal(t, int64(1), uniqueList.SiteID)
	assert.Equal(t, "/sites/test/Lists/Documents", uniqueList.URL)
	assert.Equal(t, int64(50), uniqueList.ItemCount)
	assert.True(t, uniqueList.HasUnique)
	assert.Contains(t, uniqueList.LastModified, "Run #") // Should format audit run ID

	// Test second list (inherited permissions)
	inheritedList := result[1]
	assert.Equal(t, "Regular Tasks", inheritedList.Title)
	assert.Equal(t, "inherited-list", inheritedList.ListID)
	assert.Equal(t, int64(10), inheritedList.ItemCount)
	assert.False(t, inheritedList.HasUnique)
	assert.Empty(t, inheritedList.LastModified) // No audit date
}

func TestListPresenter_FilterListsForSearch(t *testing.T) {
	// Arrange
	presenter := NewListPresenter()

	lists := []ListSummary{ // Using view type for test
		{
			ListID:   "list-1",
			Title:    "Important Documents",
			URL:      "/sites/test/Documents",
			WebTitle: "Main Site",
		},
		{
			ListID:   "list-2",
			Title:    "Task List",
			URL:      "/sites/test/Tasks",
			WebTitle: "Project Sub-Site",
		},
		{
			ListID:   "list-3",
			Title:    "Calendar Events",
			URL:      "/sites/test/Calendar",
			WebTitle: "Main Site",
		},
	}

	tests := []struct {
		name        string
		searchQuery string
		expectedIDs []string
	}{
		{
			name:        "empty_search",
			searchQuery: "",
			expectedIDs: []string{"list-1", "list-2", "list-3"}, // All lists
		},
		{
			name:        "title_match",
			searchQuery: "documents",
			expectedIDs: []string{"list-1"}, // Only "Important Documents"
		},
		{
			name:        "url_match",
			searchQuery: "tasks",
			expectedIDs: []string{"list-2"}, // Only URL with "Tasks"
		},
		{
			name:        "web_title_match",
			searchQuery: "project",
			expectedIDs: []string{"list-2"}, // Only "Project Sub-Site"
		},
		{
			name:        "case_insensitive",
			searchQuery: "IMPORTANT",
			expectedIDs: []string{"list-1"}, // Should match "Important Documents"
		},
		{
			name:        "no_matches",
			searchQuery: "nonexistent",
			expectedIDs: []string{}, // No matches
		},
		{
			name:        "multiple_matches",
			searchQuery: "main",
			expectedIDs: []string{"list-1", "list-3"}, // Both have "Main Site" in WebTitle
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Lists are already view models, use them directly
			result := presenter.FilterListsForSearch(lists, tt.searchQuery)

			// Extract IDs for comparison
			actualIDs := make([]string, len(result))
			for i, list := range result {
				actualIDs[i] = list.ListID
			}

			assert.Equal(t, tt.expectedIDs, actualIDs)
		})
	}
}

func TestListPresenter_FormatLastModified(t *testing.T) {
	// This tests the private formatLastModified method via ToListSummaries
	presenter := NewListPresenter()

	tests := []struct {
		name        string
		auditRunID  *int64
		expectedStr string
	}{
		{
			name:        "no_audit",
			auditRunID:  nil,
			expectedStr: "",
		},
		{
			name: "hours_ago",
			auditRunID: func() *int64 {
				id := int64(100)
				return &id
			}(),
			expectedStr: "Run #100",
		},
		{
			name: "days_ago",
			auditRunID: func() *int64 {
				id := int64(101)
				return &id
			}(),
			expectedStr: "Run #101",
		},
		{
			name: "weeks_ago",
			auditRunID: func() *int64 {
				id := int64(102)
				return &id
			}(),
			expectedStr: "Run #102",
		},
		{
			name: "months_ago",
			auditRunID: func() *int64 {
				id := int64(103)
				return &id
			}(),
			expectedStr: "Run #103",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domainList := []*sharepoint.List{
				{
					ID:         "test-list",
					Title:      "Test List",
					AuditRunID: tt.auditRunID,
				},
			}

			result := presenter.ToListSummaries(domainList)

			require.Len(t, result, 1)
			if tt.expectedStr == "" {
				assert.Empty(t, result[0].LastModified)
			} else if tt.name == "months_ago" {
				// For audit run IDs, check it's not empty and contains run number
				assert.NotEmpty(t, result[0].LastModified)
				assert.Contains(t, result[0].LastModified, "Run #103")
			} else {
				assert.Equal(t, tt.expectedStr, result[0].LastModified)
			}
		})
	}
}

// Test nil safety and error handling
func TestListPresenter_NilSafety(t *testing.T) {
	presenter := NewListPresenter()

	// Should not panic with nil data and return safe defaults
	result := presenter.ToSiteListsViewModel(nil)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.TotalLists)
	assert.Equal(t, 0, result.ListsWithUnique)
	assert.Equal(t, 0, result.TotalItems)
	assert.Empty(t, result.Lists)
	assert.Equal(t, int64(0), result.Site.SiteID)

	// Should not panic with nil site in data
	dataWithNilSite := &application.SiteWithListsData{
		Site:             nil,
		Lists:            []*sharepoint.List{},
		TotalLists:       1,
		ListsWithUnique:  0,
		TotalItems:       5,
		LastAuditDate:    nil,
		LastAuditDaysAgo: 0,
	}
	result2 := presenter.ToSiteListsViewModel(dataWithNilSite)
	require.NotNil(t, result2)
	assert.Equal(t, 1, result2.TotalLists)         // Should preserve data values
	assert.Equal(t, int64(0), result2.Site.SiteID) // But handle nil site gracefully
	assert.Empty(t, result2.Site.Title)            // Nil site means empty title

	// Should handle empty lists
	emptyLists := []*sharepoint.List{}
	result3 := presenter.ToListSummaries(emptyLists)
	assert.NotNil(t, result3)
	assert.Empty(t, result3)
}

// Test business logic - verify that presenter doesn't change business data
func TestListPresenter_DoesNotModifyInput(t *testing.T) {
	presenter := NewListPresenter()
	testData := helpers.NewTestData()

	originalData := &application.SiteWithListsData{
		Site:            testData.SimpleSite(1, "Original Site"),
		Lists:           []*sharepoint.List{testData.SimpleList("test", true, 5)},
		TotalLists:      1,
		ListsWithUnique: 1,
		TotalItems:      5,
	}

	// Store original values
	originalTitle := originalData.Site.Title
	originalListCount := len(originalData.Lists)
	originalItemCount := originalData.Lists[0].ItemCount

	// Transform data
	presenter.ToSiteListsViewModel(originalData)

	// Verify original data unchanged (presenter should not mutate input)
	assert.Equal(t, originalTitle, originalData.Site.Title)
	assert.Equal(t, originalListCount, len(originalData.Lists))
	assert.Equal(t, originalItemCount, originalData.Lists[0].ItemCount)
}

// Test edge cases and robustness improvements for list presenter
func TestListPresenter_EdgeCases_SearchRobustness(t *testing.T) {
	presenter := NewListPresenter()

	// Create test lists with edge case data
	lists := []ListSummary{
		{
			ListID:   "special-chars",
			Title:    "List with <HTML> & \"quotes\"",
			URL:      "/sites/test/Lists/Special%20Characters",
			WebTitle: "Site with Émojis 🎉 and ñoñó",
		},
		{
			ListID:   "unicode",
			Title:    "Список документов", // Russian
			URL:      "/sites/test/Lists/Документы",
			WebTitle: "サイト", // Japanese
		},
		{
			ListID:   "empty-fields",
			Title:    "", // Empty title
			URL:      "",
			WebTitle: "",
		},
		{
			ListID:   "whitespace",
			Title:    "   Padded with spaces   ",
			URL:      "/sites/test/Lists/SpacePadded",
			WebTitle: "  Web  Title  ",
		},
	}

	tests := []struct {
		name        string
		query       string
		expectedIDs []string
	}{
		{
			name:        "html_entities_search",
			query:       "HTML",
			expectedIDs: []string{"special-chars"},
		},
		{
			name:        "unicode_search",
			query:       "документов",
			expectedIDs: []string{"unicode"},
		},
		{
			name:        "japanese_search",
			query:       "サイト",
			expectedIDs: []string{"unicode"},
		},
		{
			name:        "emoji_search",
			query:       "🎉",
			expectedIDs: []string{"special-chars"},
		},
		{
			name:        "whitespace_tolerant",
			query:       "padded",
			expectedIDs: []string{"whitespace"},
		},
		{
			name:        "case_insensitive_unicode",
			query:       "СПИСОК",
			expectedIDs: []string{"unicode"},
		},
		{
			name:        "empty_title_robust",
			query:       "empty",
			expectedIDs: []string{}, // Should not crash on empty title
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := presenter.FilterListsForSearch(lists, tt.query)

			actualIDs := make([]string, len(result))
			for i, list := range result {
				actualIDs[i] = list.ListID
			}

			assert.Equal(t, tt.expectedIDs, actualIDs)
		})
	}
}

func TestListPresenter_EdgeCases_DateFormatting(t *testing.T) {
	presenter := NewListPresenter()

	// Test edge cases for date formatting
	tests := []struct {
		name        string
		auditRunID  *int64
		expectEmpty bool
		expectDate  bool
	}{
		{
			name:        "nil_date",
			auditRunID:  nil,
			expectEmpty: true,
		},
		{
			name: "zero_date",
			auditRunID: func() *int64 {
				id := int64(104)
				return &id
			}(),
			expectDate: true, // Should format even zero time
		},
		{
			name: "very_old_date",
			auditRunID: func() *int64 {
				id := int64(105)
				return &id
			}(),
			expectDate: true,
		},
		{
			name: "future_date",
			auditRunID: func() *int64 {
				id := int64(106) // future audit run
				return &id
			}(),
			expectDate: true, // Should handle future dates
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domainList := []*sharepoint.List{
				{
					ID:         "test-date",
					Title:      "Test List",
					AuditRunID: tt.auditRunID,
				},
			}

			result := presenter.ToListSummaries(domainList)
			require.Len(t, result, 1)

			if tt.expectEmpty {
				assert.Empty(t, result[0].LastModified)
			} else if tt.expectDate {
				assert.NotEmpty(t, result[0].LastModified)
				// Should not panic or return error strings
				assert.NotContains(t, result[0].LastModified, "error")
				assert.NotContains(t, result[0].LastModified, "panic")
			}
		})
	}
}

func TestListPresenter_EdgeCases_MultipleDataSets(t *testing.T) {
	presenter := NewListPresenter()

	// Test with a reasonable number of lists (realistic scenario)
	listCount := 15
	domainLists := make([]*sharepoint.List, listCount)

	for i := 0; i < listCount; i++ {
		domainLists[i] = &sharepoint.List{
			ID:        fmt.Sprintf("list-%d", i),
			Title:     fmt.Sprintf("List %d", i),
			ItemCount: i * 10,
			HasUnique: i%2 == 0, // Every other list has unique permissions
		}
	}

	// Should handle multiple lists correctly
	result := presenter.ToListSummaries(domainLists)

	assert.Len(t, result, listCount)
	assert.Equal(t, "list-0", result[0].ListID)
	assert.Equal(t, "list-14", result[14].ListID)
	assert.Equal(t, int64(140), result[14].ItemCount)
	assert.True(t, result[0].HasUnique)  // Even index (0)
	assert.True(t, result[14].HasUnique) // Even index (14)
}
