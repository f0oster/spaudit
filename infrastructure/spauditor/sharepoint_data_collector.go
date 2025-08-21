package spauditor

import (
	"context"
	"fmt"

	"spaudit/database"
	"spaudit/domain/audit"
	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/infrastructure/spclient"
	"spaudit/logging"

	"github.com/koltyakov/gosip/api"
)

// SharePointDataCollector orchestrates the complete SharePoint data collection process using SharePoint APIs.
type SharePointDataCollector struct {
	parameters           *audit.AuditParameters
	spClient             spclient.SharePointClient
	repo                 contracts.SharePointAuditRepository
	permissionCollector  *PermissionCollector
	sharingDataCollector *SharingDataCollector
	logger               *logging.Logger
	progressReporter     audit.ProgressReporter
	metrics              *PerformanceMetrics
}

// NewSharePointDataCollector creates a new data collector with all dependencies
func NewSharePointDataCollector(
	parameters *audit.AuditParameters,
	spClient spclient.SharePointClient,
	repo contracts.SharePointAuditRepository,
	db *database.Database,
) *SharePointDataCollector {
	return NewSharePointDataCollectorWithProgress(parameters, spClient, repo, db, audit.NewNoOpProgressReporter())
}

// NewSharePointDataCollectorWithProgress creates a new data collector with progress reporting
func NewSharePointDataCollectorWithProgress(
	parameters *audit.AuditParameters,
	spClient spclient.SharePointClient,
	repo contracts.SharePointAuditRepository,
	db *database.Database,
	progressReporter audit.ProgressReporter,
) *SharePointDataCollector {
	permissionCollector := NewPermissionCollector(spClient, repo)
	sharingDataCollector := NewSharingDataCollector(spClient, repo)
	
	// Set up progress reporting for sharing data collector
	sharingDataCollector.SetProgressReporter(progressReporter)

	return &SharePointDataCollector{
		parameters:           parameters,
		spClient:             spClient,
		repo:                 repo,
		permissionCollector:  permissionCollector,
		sharingDataCollector: sharingDataCollector,
		logger:               logging.Default().WithComponent("audit_service"),
		progressReporter:     progressReporter,
		metrics:              NewPerformanceMetrics(),
	}
}

// CollectSiteData performs complete data collection from a SharePoint site
func (s *SharePointDataCollector) CollectSiteData(ctx context.Context, auditRunID int64, siteURL string) error {
	// Defensive checks
	if s == nil {
		return fmt.Errorf("SharePointDataCollector cannot be nil")
	}
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if siteURL == "" {
		return fmt.Errorf("site URL cannot be empty")
	}
	if s.parameters == nil {
		return fmt.Errorf("config cannot be nil")
	}
	if s.spClient == nil {
		return fmt.Errorf("SharePoint client cannot be nil")
	}
	if s.repo == nil {
		return fmt.Errorf("repository cannot be nil")
	}
	if s.metrics == nil {
		return fmt.Errorf("metrics cannot be nil")
	}

	// Start overall timing
	overallStart := s.metrics.StartTiming()
	defer func() {
		s.metrics.CalculateTotalDuration(overallStart)
		s.metrics.LogPerformanceMetrics(s.logger, siteURL)
	}()

	// Validate configuration before starting
	if err := s.parameters.Validate(audit.DefaultApiConstraints()); err != nil {
		s.metrics.RecordError()
		return fmt.Errorf("invalid configuration: %w", err)
	}

	s.logger.Audit("Starting site data collection", siteURL)
	s.logger.Debug("Using configuration",
		"batch_size", s.parameters.BatchSize,
		"max_retries", s.parameters.MaxRetries,
		"timeout", s.parameters.Timeout,
		"scan_individual_items", s.parameters.ScanIndividualItems,
		"include_sharing", s.parameters.IncludeSharing,
		"skip_hidden", s.parameters.SkipHidden)
	s.progressReporter.ReportProgress(audit.StandardStages.WebDiscovery, "Starting site data collection", 10)

	// Step 1: Save site entry and get site ID
	siteStart := s.metrics.StartTiming()
	site, err := s.saveSiteEntry(ctx, auditRunID, siteURL)
	if err != nil {
		s.metrics.RecordError()
		return fmt.Errorf("save site entry: %w", err)
	}
	s.metrics.RecordSiteDiscovery(siteStart)
	s.metrics.RecordDatabaseOperation()

	// Step 2: Audit web
	s.progressReporter.ReportProgress(audit.StandardStages.WebDiscovery, "Discovering web information", 15)
	webStart := s.metrics.StartTiming()
	web, err := s.auditWeb(ctx, auditRunID, site.ID, siteURL)
	if err != nil {
		s.metrics.RecordError()
		return fmt.Errorf("audit web: %w", err)
	}
	s.metrics.RecordWebAnalysis(webStart)
	s.metrics.RecordAPICall()
	s.metrics.RecordDatabaseOperation()

	// Step 3: Cache role definitions
	s.progressReporter.ReportProgress(audit.StandardStages.Permissions, "Collecting role definitions", 20)
	roleDefsStart := s.metrics.StartTiming()
	if err := s.permissionCollector.CollectRoleDefinitions(ctx, auditRunID, site.ID); err != nil {
		s.metrics.RecordError()
		return fmt.Errorf("collect role definitions: %w", err)
	}
	s.metrics.RecordRoleDefinitions(roleDefsStart)
	s.metrics.RecordAPICall()
	s.metrics.RecordDatabaseOperation()

	// Step 4: Collect web role assignments
	s.progressReporter.ReportProgress(audit.StandardStages.Permissions, "Collecting web permissions", 25)
	webPermsStart := s.metrics.StartTiming()
	if err := s.permissionCollector.CollectWebRoleAssignments(ctx, auditRunID, site.ID, web.ID); err != nil {
		s.metrics.RecordError()
		return fmt.Errorf("collect web role assignments: %w", err)
	}
	s.metrics.RecordWebPermissions(webPermsStart)
	s.metrics.RecordAPICall()
	s.metrics.RecordDatabaseOperation()

	// Step 5: Audit lists
	s.progressReporter.ReportProgress(audit.StandardStages.ListDiscovery, "Discovering and auditing lists", 30)
	if err := s.auditLists(ctx, auditRunID, site.ID, web.ID); err != nil {
		s.metrics.RecordError()
		return fmt.Errorf("audit lists: %w", err)
	}
	// auditLists will record its own metrics internally

	// Step 6: Comprehensive sharing audit (if enabled)
	if s.parameters.IncludeSharing {
		s.progressReporter.ReportProgress(audit.StandardStages.Sharing, "Starting comprehensive sharing audit", 80)
		s.logger.Audit("Starting comprehensive sharing audit", siteURL)
		sharingStart := s.metrics.StartTiming()
		if err := s.sharingDataCollector.AuditSiteSharing(ctx, auditRunID, site.ID, siteURL); err != nil {
			s.logger.AuditError("Comprehensive sharing audit failed", err, siteURL)
			s.metrics.RecordError()
			// Don't fail the entire audit for sharing issues
		} else {
			s.logger.Audit("Completed comprehensive sharing audit", siteURL)
			s.progressReporter.ReportProgress(audit.StandardStages.Sharing, "Comprehensive sharing audit complete", 90)
		}
		s.metrics.RecordSharingAnalysis(sharingStart, 0) // TODO: Get actual sharing links count
	}

	s.progressReporter.ReportProgress(audit.StandardStages.Finalization, "Data collection completed successfully", 100)
	s.logger.Audit("Completed site data collection", siteURL)
	return nil
}

// CollectSiteSharingData performs comprehensive sharing data collection by scanning for sharing links
func (s *SharePointDataCollector) CollectSiteSharingData(ctx context.Context, auditRunID int64, siteID int64, siteURL string) error {
	// Defensive checks
	if s == nil {
		return fmt.Errorf("SharePointDataCollector cannot be nil")
	}
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if siteID <= 0 {
		return fmt.Errorf("site ID must be positive, got: %d", siteID)
	}
	if siteURL == "" {
		return fmt.Errorf("site URL cannot be empty")
	}
	if s.sharingDataCollector == nil {
		return fmt.Errorf("sharing data collector cannot be nil")
	}

	return s.sharingDataCollector.AuditSiteSharing(ctx, auditRunID, siteID, siteURL)
}

// Private helper methods

// saveSiteEntry creates the initial site entry and returns it with populated ID
func (s *SharePointDataCollector) saveSiteEntry(ctx context.Context, auditRunID int64, siteURL string) (*sharepoint.Site, error) {
	site := &sharepoint.Site{
		URL: siteURL,
		// Title will be updated when web is processed
	}
	if err := s.repo.SaveSite(ctx, site); err != nil {
		return nil, err
	}
	return site, nil
}

// auditWeb audits the web and returns web information
func (s *SharePointDataCollector) auditWeb(ctx context.Context, auditRunID int64, siteID int64, siteURL string) (*sharepoint.Web, error) {
	web, err := s.spClient.GetSiteWeb(ctx)
	if err != nil {
		return nil, fmt.Errorf("get web: %w", err)
	}

	// Set site ID for the web
	web.SiteID = siteID

	// Update the site title with the web title
	site := &sharepoint.Site{
		ID:    siteID,
		URL:   siteURL,
		Title: web.Title,
	}
	if err := s.repo.SaveSite(ctx, site); err != nil {
		return nil, fmt.Errorf("update site title: %w", err)
	}

	if err := s.repo.SaveWeb(ctx, auditRunID, web); err != nil {
		return nil, fmt.Errorf("save web: %w", err)
	}

	return web, nil
}

// auditLists audits all lists in the web using simple approach (no pagination needed)
func (s *SharePointDataCollector) auditLists(ctx context.Context, auditRunID int64, siteID int64, webID string) error {
	// Check for context cancellation
	if ctx.Err() != nil {
		return fmt.Errorf("context canceled before list auditing: %w", ctx.Err())
	}

	// Get all lists in one call (sites typically have few lists, metadata is lightweight)
	lists, err := s.spClient.GetWebLists(ctx, webID)
	if err != nil {
		return fmt.Errorf("failed to get lists for web %s (site_id=%d): %w", webID, siteID, err)
	}

	s.logger.Info("Retrieved lists for processing", "count", len(lists), "web_id", webID)
	s.metrics.RecordAPICall() // GetLists API call

	// Start timing for list processing
	listsStart := s.metrics.StartTiming()
	
	// Track skipped lists for better progress reporting
	var skippedCount int
	var processedCount int  // Track actually processed lists

	// Calculate total lists that will be processed (excluding hidden lists)
	totalListsToProcess := 0
	hiddenCount := 0
	
	s.logger.Info("Analyzing list visibility", 
		"total_discovered", len(lists),
		"skip_hidden_enabled", s.parameters.SkipHidden)
		
	if s.parameters.SkipHidden {
		for _, list := range lists {
			isHidden := s.spClient.CheckListVisibility(list.ID)
			if isHidden {
				hiddenCount++
				s.logger.Debug("Found hidden list",
					"list_title", list.Title,
					"list_id", list.ID)
			} else {
				totalListsToProcess++
			}
		}
	} else {
		totalListsToProcess = len(lists)
	}
	
	s.logger.Info("List visibility analysis complete",
		"total_discovered", len(lists),
		"visible_lists", totalListsToProcess,
		"hidden_lists", hiddenCount,
		"skip_hidden_enabled", s.parameters.SkipHidden)

	// Process all lists
	for i, list := range lists {
		// Check for context cancellation during processing
		if ctx.Err() != nil {
			return fmt.Errorf("context canceled during list processing: %w", ctx.Err())
		}

		// Skip hidden lists entirely if configured to do so
		if s.parameters.SkipHidden && s.spClient.CheckListVisibility(list.ID) {
			skippedCount++
			s.logger.Debug("Skipping hidden list due to configuration",
				"list_title", list.Title,
				"list_id", list.ID,
				"total_skipped", skippedCount)
			continue
		}

		// Increment processed count for non-skipped lists
		processedCount++
		
		// Calculate overall progress for this list (30-80% range)
		percentage := 30 + int(float64(i+1)/float64(len(lists))*50)

		// Set site ID for the list
		list.SiteID = siteID
		if err := s.auditList(ctx, auditRunID, siteID, list, percentage, processedCount, totalListsToProcess); err != nil {
			s.logger.Warn("Failed to audit list",
				"list_title", list.Title,
				"list_id", list.ID,
				"error", err.Error())
			continue
		}

		// Report overall progress after list completion
		s.progressReporter.ReportItemProgress(audit.StandardStages.ListProcessing,
			fmt.Sprintf("List %d/%d completed: %s", processedCount, totalListsToProcess, list.Title),
			percentage, processedCount, totalListsToProcess)
	}

	// Record list processing metrics
	s.metrics.RecordListProcessing(listsStart, len(lists))
	s.logger.Info("Completed lists processing", 
		"total_discovered", len(lists),
		"processed", processedCount,
		"skipped", skippedCount, 
		"web_id", webID)
	return nil
}

// auditList audits a single list
func (s *SharePointDataCollector) auditList(ctx context.Context, auditRunID int64, siteID int64, list *sharepoint.List, overallPercentage int, currentListNumber int, totalLists int) error {
	// Substate 1: Save list metadata
	s.progressReporter.ReportProgress(audit.StandardStages.ListProcessing,
		fmt.Sprintf("List %d/%d - Saving metadata: %s", currentListNumber, totalLists, list.Title), overallPercentage)
	
	if err := s.repo.SaveList(ctx, auditRunID, list); err != nil {
		return fmt.Errorf("save list %s (site_id=%d, list_id=%s): %w", list.Title, siteID, list.ID, err)
	}

	// Substate 2: Collect list permissions
	s.progressReporter.ReportProgress(audit.StandardStages.ListProcessing,
		fmt.Sprintf("List %d/%d - Collecting list permissions: %s", currentListNumber, totalLists, list.Title), overallPercentage)
		
	if err := s.permissionCollector.CollectListRoleAssignments(ctx, auditRunID, siteID, list.ID); err != nil {
		s.logger.Warn("Failed to collect list role assignments", "list_title", list.Title, "error", err.Error())
	}

	// Substate 3: Audit individual items (documents/folders) if individual item scanning is enabled
	if s.parameters.ScanIndividualItems {
		if list.ItemCount > 0 {
			s.progressReporter.ReportProgress(audit.StandardStages.ListProcessing,
				fmt.Sprintf("List %d/%d - Preparing to scan items: %s (~%d items)", currentListNumber, totalLists, list.Title, list.ItemCount), overallPercentage)
		} else {
			s.progressReporter.ReportProgress(audit.StandardStages.ListProcessing,
				fmt.Sprintf("List %d/%d - Preparing to scan items: %s (empty list)", currentListNumber, totalLists, list.Title), overallPercentage)
		}
			
		if err := s.auditListItems(ctx, auditRunID, siteID, list.ID, list.Title, overallPercentage, currentListNumber, totalLists, list.ItemCount); err != nil {
			s.logger.Warn("Failed to audit individual items in list", "list_title", list.Title, "error", err.Error())
			// Continue processing other lists - don't return error
		}
	}

	return nil
}

// auditListItems performs deep scanning of individual items (documents, folders, files)
// within a SharePoint list. This includes collecting permissions and metadata for each item.
// Uses Gosip's native pagination to efficiently handle lists with thousands of items.
func (s *SharePointDataCollector) auditListItems(ctx context.Context, auditRunID int64, siteID int64, listID string, listTitle string, overallPercentage int, currentListNumber int, totalLists int, expectedItemCount int) error {
	// Check for context cancellation at the start
	if ctx.Err() != nil {
		return fmt.Errorf("context canceled before auditing items for list %s: %w", listID, ctx.Err())
	}

	// Start timing for item processing
	itemsStart := s.metrics.StartTiming()

	batchSize := s.parameters.BatchSize
	totalProcessed := 0
	itemsWithUniquePerms := 0

	// Create the items query (*api.Items)
	itemsQuery := s.spClient.CreateListItemsQuery(ctx, listID, batchSize)
	s.metrics.RecordAPICall() // GetItemsQuery preparation

	err := s.walkListItems(ctx, itemsQuery, func(itemResp api.ItemResp) error {
		// Process each individual SharePoint item (document, folder, etc.) and extract sensitivity label in single parse
		domainItem, sensitivityLabel, err := s.spClient.ConvertItemWithSensitivityLabel(ctx, itemResp, listID, siteID)
		if err != nil {
			s.logger.Warn("Failed to process individual item response", "error", err.Error())
			s.metrics.RecordError()
			return nil // Continue processing other items
		}

		// Save sensitivity label information if present
		if sensitivityLabel != nil {
			if err := s.repo.SaveItemSensitivityLabel(ctx, sensitivityLabel); err != nil {
				s.logger.Warn("Failed to save sensitivity label", "item_guid", domainItem.GUID, "error", err.Error())
				s.metrics.RecordError()
			} else {
				s.logger.Debug("Sensitivity label saved successfully", "item_guid", domainItem.GUID, "label_id", sensitivityLabel.LabelID)
				s.metrics.RecordDatabaseOperation()
			}
		}

		// Set site ID and audit this individual item's permissions and metadata
		domainItem.SiteID = siteID
		if err := s.auditIndividualItem(ctx, auditRunID, siteID, domainItem); err != nil {
			s.logger.Warn("Failed to audit individual item permissions", "item_guid", domainItem.GUID, "error", err.Error())
		}

		// Track items with unique permissions
		if domainItem.HasUnique {
			itemsWithUniquePerms++
		}

		totalProcessed++
		
		// Report progress every batch or every 50 items for better UX feedback
		progressInterval := batchSize
		if progressInterval > 50 {
			progressInterval = 50
		}
		
		if totalProcessed%progressInterval == 0 {
			// Show progress with expected count if available
			if expectedItemCount > 0 {
				percentage := int(float64(totalProcessed) / float64(expectedItemCount) * 100)
				if percentage > 100 {
					percentage = 100 // Cap at 100% in case we find more items than expected
				}
				s.progressReporter.ReportProgress(audit.StandardStages.ListProcessing,
					fmt.Sprintf("List %d/%d - Scanning items: %s (%d/%d items, %d%%)", currentListNumber, totalLists, listTitle, totalProcessed, expectedItemCount, percentage), overallPercentage)
			} else {
				s.progressReporter.ReportProgress(audit.StandardStages.ListProcessing,
					fmt.Sprintf("List %d/%d - Scanning items: %s (%d items processed)", currentListNumber, totalLists, listTitle, totalProcessed), overallPercentage)
			}
			s.logger.Debug("Deep item scanning progress", "items_processed", totalProcessed, "expected_count", expectedItemCount, "list_id", listID)
		}

		return nil
	})

	if err != nil {
		s.metrics.RecordError()
		return fmt.Errorf("failed to walk list items for list %s (site_id=%d, batch_size=%d): %w",
			listID, siteID, batchSize, err)
	}

	// Final progress update with completion
	if totalProcessed > 0 {
		if itemsWithUniquePerms > 0 {
			s.progressReporter.ReportProgress(audit.StandardStages.ListProcessing,
				fmt.Sprintf("List %d/%d - Collecting permissions: %s (%d items with unique permissions)", currentListNumber, totalLists, listTitle, itemsWithUniquePerms), overallPercentage)
		}
		
		// Show actual vs expected count in completion message
		if expectedItemCount > 0 && totalProcessed != expectedItemCount {
			s.progressReporter.ReportProgress(audit.StandardStages.ListProcessing,
				fmt.Sprintf("List %d/%d - Completed items scan: %s (%d items processed, expected %d)", currentListNumber, totalLists, listTitle, totalProcessed, expectedItemCount), overallPercentage)
		} else {
			s.progressReporter.ReportProgress(audit.StandardStages.ListProcessing,
				fmt.Sprintf("List %d/%d - Completed items scan: %s (%d items processed)", currentListNumber, totalLists, listTitle, totalProcessed), overallPercentage)
		}
	} else {
		if expectedItemCount > 0 {
			s.progressReporter.ReportProgress(audit.StandardStages.ListProcessing,
				fmt.Sprintf("List %d/%d - Completed: %s (empty list, expected %d items)", currentListNumber, totalLists, listTitle, expectedItemCount), overallPercentage)
		} else {
			s.progressReporter.ReportProgress(audit.StandardStages.ListProcessing,
				fmt.Sprintf("List %d/%d - Completed: %s (empty list)", currentListNumber, totalLists, listTitle), overallPercentage)
		}
	}

	// Record item processing metrics
	s.metrics.RecordItemProcessing(itemsStart, totalProcessed)
	s.metrics.ItemsWithUniquePerms += itemsWithUniquePerms

	s.logger.Info("Completed deep item scanning", "total_items", totalProcessed, "unique_perms", itemsWithUniquePerms, "list_id", listID)
	return nil
}

// walkListItems iterates through all items in a SharePoint list using Gosip's native pagination.
// It calls the onItem callback for each individual item (document, folder, etc.) found in the list.
// This efficiently handles lists with thousands of items by processing them in pages.
func (s *SharePointDataCollector) walkListItems(ctx context.Context, items *api.Items, onItem func(api.ItemResp) error) error {
	// Defensive check: ensure items is not nil
	if items == nil {
		return fmt.Errorf("items query cannot be nil")
	}

	page, err := items.GetPaged()
	if err != nil {
		s.metrics.RecordError()
		return err
	}
	if page == nil { // empty list
		return nil
	}

	s.metrics.RecordAPICall() // GetPaged API call

	for p := page; ; {
		// Check for context cancellation before processing each page
		if ctx.Err() != nil {
			return fmt.Errorf("context canceled during pagination: %w", ctx.Err())
		}

		// Defensive check: ensure page has Items
		if p.Items == nil {
			s.logger.Warn("Page has nil Items collection, skipping")
			break
		}

		// page.Items.Data() returns []api.ItemResp (each ItemResp is []byte with generated methods)
		for _, ir := range p.Items.Data() { // ir: api.ItemResp
			// Check for context cancellation more frequently within the page
			if ctx.Err() != nil {
				return fmt.Errorf("context canceled during item processing: %w", ctx.Err())
			}

			// Defensive check: ensure callback is not nil
			if onItem == nil {
				return fmt.Errorf("onItem callback cannot be nil")
			}

			if err := onItem(ir); err != nil {
				s.metrics.RecordError()
				return err
			}
		}

		if !p.HasNextPage() {
			return nil
		}

		// Check for context cancellation before fetching next page
		if ctx.Err() != nil {
			return fmt.Errorf("context canceled before next page: %w", ctx.Err())
		}

		p, err = p.GetNextPage()
		if err != nil {
			s.metrics.RecordError()
			return err
		}
		s.metrics.RecordAPICall() // GetNextPage API call
	}

	return nil
}

// auditIndividualItem audits a single SharePoint item (document, folder, or file).
// This includes saving the item metadata and collecting its unique permissions if it has any.
func (s *SharePointDataCollector) auditIndividualItem(ctx context.Context, auditRunID int64, siteID int64, item *sharepoint.Item) error {

	// Defensive checks
	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}
	if item.GUID == "" {
		return fmt.Errorf("item GUID cannot be empty")
	}
	if siteID <= 0 {
		return fmt.Errorf("site ID must be positive, got: %d", siteID)
	}

	// Check if context was canceled before processing this item
	if ctx.Err() != nil {
		return fmt.Errorf("context canceled before auditing item %s: %w", item.GUID, ctx.Err())
	}

	// Save item
	if err := s.repo.SaveItem(ctx, auditRunID, item); err != nil {
		s.metrics.RecordError()
		return fmt.Errorf("save item %s (site_id=%d, list_id=%s, item_id=%d): %w",
			item.GUID, siteID, item.ListID, item.ID, err)
	}
	s.metrics.RecordDatabaseOperation()

	// Collect item role assignments if it has unique permissions
	if item.HasUnique {
		if err := s.permissionCollector.CollectItemRoleAssignments(ctx, auditRunID, siteID, item.ListID, item.GUID, item.ID); err != nil {
			s.metrics.RecordWarning()
			s.logger.Warn("Failed to collect item role assignments", "item_guid", item.GUID, "error", err.Error())
		} else {
			s.metrics.PermissionsCollected++
		}
	}

	return nil
}
