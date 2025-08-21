package workflows

import (
	"context"
	"fmt"
	"time"

	"spaudit/database"
	"spaudit/domain/audit"
	"spaudit/domain/contracts"
	"spaudit/domain/jobs"
	"spaudit/domain/sharepoint"
	"spaudit/infrastructure/spauditor"
	"spaudit/infrastructure/spclient"
	"spaudit/logging"
)

// AuditWorkflowResult represents the comprehensive results of an audit workflow
type AuditWorkflowResult struct {
	SiteID          int64
	SiteURL         string
	TotalLists      int
	TotalItems      int64
	ItemsWithUnique int64

	// Content analysis results
	ContentAnalysis *sharepoint.ContentAnalysis
	ContentRisk     *sharepoint.ContentRiskAssessment

	// Sharing analysis results
	SharingAnalysis *sharepoint.SharingPatternAnalysis
	SharingRisk     *sharepoint.SharingRiskAssessment

	// Permission analysis results
	PermissionAnalysis *sharepoint.SharePointRiskAssessment

	// Overall audit metadata
	StartedAt   time.Time
	CompletedAt time.Time
	Duration    time.Duration
}

// AuditWorkflow orchestrates the complete audit process using domain services
type AuditWorkflow struct {
	// Domain services (pure business logic)
	contentService     *sharepoint.ContentService
	sharingService     *sharepoint.SharingService
	permissionsService *sharepoint.PermissionsService

	// Existing audit services (for data collection)
	sharingDataCollector *spauditor.SharingDataCollector

	// Repository interfaces (for data access after collection)
	auditRepo   contracts.SharePointAuditRepository
	sharingRepo contracts.SharingRepository
	itemRepo    contracts.ItemRepository

	// Infrastructure dependencies
	spClient         spclient.SharePointClient
	db               *database.Database
	logger           *logging.Logger
	progressReporter audit.ProgressReporter
}

// NewAuditWorkflow creates a new audit workflow with domain services
func NewAuditWorkflow(
	auditRepo contracts.SharePointAuditRepository,
	sharingRepo contracts.SharingRepository,
	itemRepo contracts.ItemRepository,
	spClient spclient.SharePointClient,
	db *database.Database,
) *AuditWorkflow {
	// Create existing audit services for data collection
	sharingDataCollector := spauditor.NewSharingDataCollector(spClient, auditRepo)

	return &AuditWorkflow{
		contentService:       sharepoint.NewContentService(),
		sharingService:       sharepoint.NewSharingService(),
		permissionsService:   sharepoint.NewPermissionsService(),
		sharingDataCollector: sharingDataCollector,
		auditRepo:            auditRepo,
		sharingRepo:          sharingRepo,
		itemRepo:             itemRepo,
		spClient:             spClient,
		db:                   db,
		logger:               logging.Default().WithComponent("audit_workflow"),
	}
}

// SetProgressReporter sets the progress reporter for workflow progress updates
func (w *AuditWorkflow) SetProgressReporter(reporter audit.ProgressReporter) {
	w.progressReporter = reporter
}

// ExecuteSiteAudit orchestrates a complete site audit using domain services
func (w *AuditWorkflow) ExecuteSiteAudit(ctx context.Context, job *jobs.Job, siteURL string) (*AuditWorkflowResult, error) {
	// Get audit run ID from job
	auditRunID := job.GetAuditRunID()
	if auditRunID == 0 {
		return nil, fmt.Errorf("job must have an associated audit run")
	}
	startTime := time.Now()
	w.logger.Audit("Starting platform audit workflow for site", siteURL)

	result := &AuditWorkflowResult{
		SiteURL:   siteURL,
		StartedAt: startTime,
	}

	// Get parameters from job, use defaults if not provided
	parameters := job.GetAuditParameters()
	if parameters == nil {
		parameters = audit.DefaultParameters()
		w.logger.Info("No parameters provided in job, using defaults", "job_id", job.ID)
	}

	// Phase 1: Full Site Data Collection using proven auditor
	w.reportProgress(audit.StandardStages.WebDiscovery, "Performing comprehensive site audit", 10)
	siteID, err := w.performFullSiteAudit(ctx, auditRunID, siteURL, parameters)
	if err != nil {
		return nil, fmt.Errorf("full site audit: %w", err)
	}
	result.SiteID = siteID

	// Phase 2: Content Collection and Analysis
	w.reportProgress(audit.StandardStages.ListDiscovery, "Analyzing content structure", 30)
	if err := w.analyzeContent(ctx, siteID, result); err != nil {
		return nil, fmt.Errorf("content analysis: %w", err)
	}

	// Phase 3: Sharing Links Analysis
	w.reportProgress(audit.StandardStages.Permissions, "Analyzing sharing patterns", 50)
	if err := w.analyzeSharing(ctx, auditRunID, siteID, result); err != nil {
		return nil, fmt.Errorf("sharing analysis: %w", err)
	}

	// Phase 4: Permission Analysis
	w.reportProgress(audit.StandardStages.Permissions, "Analyzing permission risks", 70)
	if err := w.analyzePermissions(ctx, siteID, result); err != nil {
		return nil, fmt.Errorf("permission analysis: %w", err)
	}

	// Phase 5: Finalization
	w.reportProgress(audit.StandardStages.Finalization, "Completing audit analysis", 90)
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	w.reportProgress(audit.StandardStages.Finalization, "Audit workflow completed", 100)
	w.logger.Info("Platform audit workflow completed", "siteURL", siteURL, "duration", result.Duration.String())

	return result, nil
}

// Private orchestration methods

// analyzeContent performs comprehensive content analysis using domain services
func (w *AuditWorkflow) analyzeContent(ctx context.Context, siteID int64, result *AuditWorkflowResult) error {
	// Get lists directly from SharePoint (simple approach - sites typically have few lists)
	// For a full workflow, we'd want to get a web first, but for analysis we can use empty webID
	lists, err := w.spClient.GetWebLists(ctx, "") // Get all lists
	if err != nil {
		return fmt.Errorf("get lists from SharePoint: %w", err)
	}
	result.TotalLists = len(lists)

	var allItems []*sharepoint.Item
	var totalItems int64

	// Collect all items across all lists
	for _, list := range lists {
		items, err := w.itemRepo.GetItemsForList(ctx, siteID, list.ID, 0, 10000) // Large limit for analysis
		if err != nil {
			w.logger.Warn("Failed to get items for list", "listID", list.ID, "error", err)
			continue
		}
		allItems = append(allItems, items...)
		totalItems += int64(len(items))
	}

	result.TotalItems = totalItems

	// Use content service for analysis
	contentAnalysis := w.contentService.AnalyzeItems(allItems)
	result.ContentAnalysis = contentAnalysis
	result.ItemsWithUnique = contentAnalysis.ItemsWithUnique

	// Assess content risk using domain service
	contentRisk := w.contentService.AssessContentRisk(contentAnalysis)
	result.ContentRisk = contentRisk

	w.logger.Info("Content analysis completed",
		"totalItems", result.TotalItems,
		"itemsWithUnique", result.ItemsWithUnique,
		"riskLevel", contentRisk.RiskLevel)

	return nil
}

// analyzeSharing performs sharing link analysis using domain services
func (w *AuditWorkflow) analyzeSharing(ctx context.Context, auditRunID int64, siteID int64, result *AuditWorkflowResult) error {
	// Set up progress reporting for sharing data collector
	w.sharingDataCollector.SetProgressReporter(w.progressReporter)
	
	// Use the existing sharing data collector to perform comprehensive site sharing collection
	// This will collect all sharing data using the proven approach
	if err := w.sharingDataCollector.AuditSiteSharing(ctx, auditRunID, siteID, ""); err != nil {
		w.logger.Warn("Sharing audit failed, proceeding with available data", "error", err)
		// Don't fail the workflow - proceed with any available data
	}

	// Now get all sharing links that were collected/exist in the database
	allSharingLinks, err := w.sharingRepo.GetSharingLinksForList(ctx, siteID, "") // This should work after audit
	if err != nil {
		// If this fails, we can't do sharing analysis
		w.logger.Warn("Could not retrieve sharing links for analysis", "error", err)
		result.SharingRisk = &sharepoint.SharingRiskAssessment{
			RiskScore:       0,
			RiskLevel:       "Low",
			RiskFactors:     []string{"No sharing data available"},
			Recommendations: []string{},
		}
		return nil
	}

	// Use sharing service for risk analysis
	sharingRisk := w.sharingService.AnalyzeSharingRisk(allSharingLinks)
	result.SharingRisk = sharingRisk

	// Analyze sharing patterns across items if we have content analysis
	if result.ContentAnalysis != nil {
		// Build sharing data map for pattern analysis
		sharingData := make(map[string][]*sharepoint.SharingLink)
		for _, link := range allSharingLinks {
			sharingData[link.ItemGUID] = append(sharingData[link.ItemGUID], link)
		}

		// Get items for pattern analysis (we could optimize this by reusing from content analysis)
		items, err := w.itemRepo.GetItemsForList(ctx, siteID, "", 0, 10000) // Get all items
		if err == nil {
			sharingPatterns := w.sharingService.AnalyzeSharingPatterns(items, sharingData)
			result.SharingAnalysis = sharingPatterns
		}
	}

	w.logger.Info("Sharing analysis completed",
		"totalLinks", sharingRisk.TotalLinks,
		"riskLevel", sharingRisk.RiskLevel)

	return nil
}

// analyzePermissions performs permission risk analysis using domain services
func (w *AuditWorkflow) analyzePermissions(ctx context.Context, siteID int64, result *AuditWorkflowResult) error {
	// Build permission risk data from our collected information
	riskData := &sharepoint.SharePointRiskData{
		TotalItems:         result.TotalItems,
		ItemsWithUnique:    result.ItemsWithUnique,
		TotalAssignments:   0, // Would need to collect assignment data
		LimitedAccessCount: 0, // Would need assignment analysis
		FullControlCount:   0, // Would need assignment analysis
		ContributeCount:    0, // Would need assignment analysis
		SharingLinkCount:   0, // Use sharing analysis results
	}

	// Add sharing link count from sharing analysis
	if result.SharingRisk != nil {
		riskData.SharingLinkCount = result.SharingRisk.TotalLinks
	}

	// Use permissions service
	permissionAssessment := w.permissionsService.CalculateSharePointRiskAssessment(riskData)
	result.PermissionAnalysis = permissionAssessment

	w.logger.Info("Permission analysis completed",
		"riskScore", permissionAssessment.RiskScore,
		"riskLevel", permissionAssessment.RiskLevel)

	return nil
}

// performFullSiteAudit uses the existing proven auditor to perform comprehensive data collection
func (w *AuditWorkflow) performFullSiteAudit(ctx context.Context, auditRunID int64, siteURL string, parameters *audit.AuditParameters) (int64, error) {
	// Use the provided parameters from the web UI
	// Apply ValidateAndSetDefaults to ensure all fields have reasonable values
	if err := parameters.ValidateAndSetDefaults(audit.DefaultApiConstraints()); err != nil {
		return 0, fmt.Errorf("invalid audit configuration: %w", err)
	}

	// Create the proven data collector with progress reporting
	dataCollector := spauditor.NewSharePointDataCollectorWithProgress(parameters, w.spClient, w.auditRepo, w.db, w.progressReporter)

	// Run the full site data collection
	if err := dataCollector.CollectSiteData(ctx, auditRunID, siteURL); err != nil {
		return 0, fmt.Errorf("data collector site collection failed: %w", err)
	}

	// The data collector has created and populated the site with proper title
	// Query the database for the site that the data collector created
	site, err := w.auditRepo.GetSiteByURL(ctx, siteURL)
	if err != nil {
		return 0, fmt.Errorf("get site by URL after data collection: %w", err)
	}
	if site == nil {
		return 0, fmt.Errorf("site not found after successful data collection: %s", siteURL)
	}

	return site.ID, nil
}

// reportProgress reports workflow progress if a reporter is configured
func (w *AuditWorkflow) reportProgress(stage, description string, percentage int) {
	if w.progressReporter != nil {
		w.progressReporter.ReportProgress(stage, description, percentage)
	}
}
