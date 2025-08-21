package spauditor

import (
	"time"

	"spaudit/logging"
)

// PerformanceMetrics tracks detailed performance data for audit operations
type PerformanceMetrics struct {
	// Timing metrics
	SiteDiscoveryDuration   time.Duration
	WebAnalysisDuration     time.Duration
	RoleDefinitionsDuration time.Duration
	WebPermissionsDuration  time.Duration
	ListProcessingDuration  time.Duration
	ItemProcessingDuration  time.Duration
	SharingAnalysisDuration time.Duration
	TotalDuration           time.Duration

	// Throughput metrics
	TotalListsProcessed  int
	TotalItemsProcessed  int
	ItemsWithUniquePerms int
	SharingLinksFound    int
	PermissionsCollected int

	// API call metrics
	SharePointAPICallsCount int
	DatabaseOperationsCount int

	// Error metrics
	ErrorsEncountered   int
	WarningsEncountered int

	// Resource usage
	PeakMemoryUsageMB     int64
	AverageProcessingRate float64 // items per second
}

// NewPerformanceMetrics creates a new metrics collection instance
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{}
}

// StartTiming begins timing for a specific operation
func (m *PerformanceMetrics) StartTiming() time.Time {
	return time.Now()
}

// RecordSiteDiscovery records site discovery timing
func (m *PerformanceMetrics) RecordSiteDiscovery(start time.Time) {
	m.SiteDiscoveryDuration = time.Since(start)
}

// RecordWebAnalysis records web analysis timing
func (m *PerformanceMetrics) RecordWebAnalysis(start time.Time) {
	m.WebAnalysisDuration = time.Since(start)
}

// RecordRoleDefinitions records role definitions collection timing
func (m *PerformanceMetrics) RecordRoleDefinitions(start time.Time) {
	m.RoleDefinitionsDuration = time.Since(start)
}

// RecordWebPermissions records web permissions collection timing
func (m *PerformanceMetrics) RecordWebPermissions(start time.Time) {
	m.WebPermissionsDuration = time.Since(start)
}

// RecordListProcessing records list processing timing
func (m *PerformanceMetrics) RecordListProcessing(start time.Time, listsProcessed int) {
	m.ListProcessingDuration = time.Since(start)
	m.TotalListsProcessed = listsProcessed
}

// RecordItemProcessing records item processing timing
func (m *PerformanceMetrics) RecordItemProcessing(start time.Time, itemsProcessed int) {
	m.ItemProcessingDuration = time.Since(start)
	m.TotalItemsProcessed = itemsProcessed
}

// RecordSharingAnalysis records sharing analysis timing
func (m *PerformanceMetrics) RecordSharingAnalysis(start time.Time, sharingLinksFound int) {
	m.SharingAnalysisDuration = time.Since(start)
	m.SharingLinksFound = sharingLinksFound
}

// RecordAPICall increments the API call counter
func (m *PerformanceMetrics) RecordAPICall() {
	m.SharePointAPICallsCount++
}

// RecordDatabaseOperation increments the database operation counter
func (m *PerformanceMetrics) RecordDatabaseOperation() {
	m.DatabaseOperationsCount++
}

// RecordError increments the error counter
func (m *PerformanceMetrics) RecordError() {
	m.ErrorsEncountered++
}

// RecordWarning increments the warning counter
func (m *PerformanceMetrics) RecordWarning() {
	m.WarningsEncountered++
}

// CalculateTotalDuration calculates and stores the total duration
func (m *PerformanceMetrics) CalculateTotalDuration(start time.Time) {
	m.TotalDuration = time.Since(start)

	// Calculate processing rate
	if m.TotalDuration > 0 && m.TotalItemsProcessed > 0 {
		seconds := m.TotalDuration.Seconds()
		m.AverageProcessingRate = float64(m.TotalItemsProcessed) / seconds
	}
}

// LogPerformanceMetrics outputs comprehensive performance metrics
func (m *PerformanceMetrics) LogPerformanceMetrics(logger *logging.Logger, siteURL string) {
	logger.Info("=== Audit Performance Metrics ===",
		"site_url", siteURL,
		"total_duration_ms", m.TotalDuration.Milliseconds(),
		"total_duration_human", m.TotalDuration.Round(time.Millisecond).String())

	// Timing breakdown
	logger.Info("Timing Breakdown",
		"site_discovery_ms", m.SiteDiscoveryDuration.Milliseconds(),
		"web_analysis_ms", m.WebAnalysisDuration.Milliseconds(),
		"role_definitions_ms", m.RoleDefinitionsDuration.Milliseconds(),
		"web_permissions_ms", m.WebPermissionsDuration.Milliseconds(),
		"list_processing_ms", m.ListProcessingDuration.Milliseconds(),
		"item_processing_ms", m.ItemProcessingDuration.Milliseconds(),
		"sharing_analysis_ms", m.SharingAnalysisDuration.Milliseconds())

	// Throughput metrics
	logger.Info("Throughput Metrics",
		"lists_processed", m.TotalListsProcessed,
		"items_processed", m.TotalItemsProcessed,
		"items_with_unique", m.ItemsWithUniquePerms,
		"sharing_links_found", m.SharingLinksFound,
		"permissions_collected", m.PermissionsCollected,
		"processing_rate_items_per_sec", m.AverageProcessingRate)

	// Operation counts
	logger.Info("Operation Counts",
		"sharepoint_api_calls", m.SharePointAPICallsCount,
		"database_operations", m.DatabaseOperationsCount,
		"errors", m.ErrorsEncountered,
		"warnings", m.WarningsEncountered)

	// Performance insights
	if m.TotalDuration > 0 {
		listPercent := float64(m.ListProcessingDuration.Milliseconds()) / float64(m.TotalDuration.Milliseconds()) * 100
		itemPercent := float64(m.ItemProcessingDuration.Milliseconds()) / float64(m.TotalDuration.Milliseconds()) * 100
		sharingPercent := float64(m.SharingAnalysisDuration.Milliseconds()) / float64(m.TotalDuration.Milliseconds()) * 100

		logger.Info("Performance Insights",
			"list_processing_percent", listPercent,
			"item_processing_percent", itemPercent,
			"sharing_analysis_percent", sharingPercent)
	}

	logger.Info("Performance metrics summary complete", "site_url", siteURL)
}
