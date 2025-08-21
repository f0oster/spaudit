package executors

import (
	"context"
	"encoding/json"

	"spaudit/application"
	"spaudit/domain/audit"
	"spaudit/domain/jobs"
	"spaudit/logging"
)

// SiteAuditExecutor handles site audit job execution
type SiteAuditExecutor struct {
	workflowFactory application.WorkflowFactory
	logger          *logging.Logger
}

// NewSiteAuditExecutor creates a new site audit executor
func NewSiteAuditExecutor(workflowFactory application.WorkflowFactory) *SiteAuditExecutor {
	return &SiteAuditExecutor{
		workflowFactory: workflowFactory,
		logger:          logging.Default().WithComponent("site_audit_executor"),
	}
}

// Execute implements the JobExecutor interface for site audit jobs
func (e *SiteAuditExecutor) Execute(ctx context.Context, job *jobs.Job, progressCallback application.ProgressCallback) error {
	siteURL := job.GetSiteURL()
	e.logger.Info("Starting site audit execution", "jobID", job.ID, "siteURL", siteURL)

	// Extract audit parameters from job context or use default
	parameters := e.extractParametersFromJob(job)

	// Create audit workflow using factory with parameters
	workflow, err := e.workflowFactory.CreateAuditWorkflow(siteURL, parameters)
	if err != nil {
		return err
	}

	// Set up progress reporting
	progressReporter := &ProgressAdapter{
		progressCallback: progressCallback,
		logger:           e.logger,
	}
	workflow.SetProgressReporter(progressReporter)

	// Execute the audit workflow
	result, err := workflow.ExecuteSiteAudit(ctx, job, siteURL)
	if err != nil {
		return err
	}

	// Store detailed results in the job
	if err := e.storeResultInJob(job, result); err != nil {
		e.logger.Warn("Failed to store detailed results in job", "job_id", job.ID, "error", err)
		// Don't fail the job for this
	}

	e.logger.Info("Site audit execution completed", "jobID", job.ID, "siteURL", siteURL)
	return nil
}

// ProgressAdapter adapts the workflow progress reporting to the job system's progress callback
type ProgressAdapter struct {
	progressCallback application.ProgressCallback
	logger           *logging.Logger
}

// ReportProgress implements the ProgressReporter interface
func (a *ProgressAdapter) ReportProgress(stage, description string, percentage int) {
	a.logger.Debug("Workflow progress", "stage", stage, "description", description, "percentage", percentage)
	a.progressCallback(stage, description, percentage, 0, 0)
}

// ReportItemProgress implements the ProgressReporter interface with item counts
func (a *ProgressAdapter) ReportItemProgress(stage, description string, percentage, itemsDone, itemsTotal int) {
	a.logger.Debug("Workflow item progress", "stage", stage, "description", description,
		"percentage", percentage, "itemsDone", itemsDone, "itemsTotal", itemsTotal)
	a.progressCallback(stage, description, percentage, itemsDone, itemsTotal)
}

// storeResultInJob stores the detailed workflow results in the job's Result field as JSON
func (e *SiteAuditExecutor) storeResultInJob(job *jobs.Job, result application.AuditWorkflowResult) error {
	// Convert workflow result to a serializable format
	resultData := map[string]interface{}{
		"contentRisk": map[string]interface{}{
			"level": result.GetContentRisk().GetRiskLevel(),
			"score": result.GetContentRisk().GetRiskScore(),
		},
		"sharingRisk": map[string]interface{}{
			"level":      result.GetSharingRisk().GetRiskLevel(),
			"totalLinks": result.GetSharingRisk().GetTotalLinks(),
		},
		"permissionRisk": map[string]interface{}{
			"level": result.GetPermissionAnalysis().GetRiskLevel(),
			"score": result.GetPermissionAnalysis().GetRiskScore(),
		},
		"duration":        result.GetDuration(),
		"totalLists":      result.GetTotalLists(),
		"totalItems":      result.GetTotalItems(),
		"itemsWithUnique": result.GetItemsWithUnique(),
	}

	// Convert to JSON and store in Result field
	resultJSON, err := json.Marshal(resultData)
	if err != nil {
		return err
	}

	job.Result = string(resultJSON)

	// Update job statistics
	job.State.Stats.ListsFound = result.GetTotalLists()
	job.State.Stats.ItemsFound = int(result.GetTotalItems())
	job.State.Stats.ItemsProcessed = int(result.GetTotalItems())
	job.State.Stats.PermissionsAnalyzed = int(result.GetItemsWithUnique())
	job.State.Stats.SharingLinksFound = result.GetSharingRisk().GetTotalLinks()

	return nil
}

// extractParametersFromJob extracts the audit parameters from the job context
func (e *SiteAuditExecutor) extractParametersFromJob(job *jobs.Job) *audit.AuditParameters {
	// Try to extract parameters from job using the existing method
	if parameters := job.GetAuditParameters(); parameters != nil {
		e.logger.Info("Using job-specific parameters", "jobID", job.ID,
			"batch_size", parameters.BatchSize, "include_sharing", parameters.IncludeSharing)
		return parameters
	}

	// Fall back to default parameters
	e.logger.Info("Using default parameters", "jobID", job.ID)
	return audit.DefaultParameters()
}
