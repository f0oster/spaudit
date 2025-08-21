package application

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"spaudit/database"
	"spaudit/domain/audit"
	"spaudit/domain/jobs"
	"spaudit/logging"
)

// AuditService defines the interface for audit operations used by job and handler layers.
type AuditService interface {
	// Methods needed by AuditHandlers.
	QueueAudit(ctx context.Context, siteURL string, parameters *audit.AuditParameters) (*audit.AuditRequest, error)
	GetAuditStatus(siteURL string) (*audit.ActiveAudit, bool)
	GetActiveAudits() []*audit.ActiveAudit
	GetAuditHistory(limit int) []*audit.AuditResult
	CancelAudit(siteURL string) error

	// Methods needed by other services.
	IsSiteBeingAudited(siteURL string) bool
	BuildAuditParametersFromFormData(formData map[string][]string) *audit.AuditParameters
}

// AuditServiceImpl is the production implementation of AuditService using JobService for management.
type AuditServiceImpl struct {
	jobService JobService
	db         *database.Database
	logger     *logging.Logger
}

// NewAuditService creates a new audit service using platform workflows.
func NewAuditService(
	jobService JobService,
	db *database.Database,
) AuditService {
	return &AuditServiceImpl{
		jobService: jobService,
		db:         db,
		logger:     logging.Default().WithComponent("audit_service"),
	}
}

// IsSiteBeingAudited checks if a site is currently being audited.
func (s *AuditServiceImpl) IsSiteBeingAudited(siteURL string) bool {
	// Check for running audit jobs for this site
	runningJobs := s.jobService.ListJobsByStatus(jobs.JobStatusRunning)
	for _, job := range runningJobs {
		if job.GetSiteURL() == siteURL && job.Type == jobs.JobTypeSiteAudit {
			return true
		}
	}

	// Check for pending audit jobs for this site
	pendingJobs := s.jobService.ListJobsByStatus(jobs.JobStatusPending)
	for _, job := range pendingJobs {
		if job.GetSiteURL() == siteURL && job.Type == jobs.JobTypeSiteAudit {
			return true
		}
	}

	return false
}

// BuildAuditParametersFromFormData creates audit parameters from form data
func (s *AuditServiceImpl) BuildAuditParametersFromFormData(formData map[string][]string) *audit.AuditParameters {
	// Start with default parameters
	parameters := audit.DefaultParameters()

	// Helper function to check if form field is "on" or explicitly set
	hasFormValue := func(key string) bool {
		if values, exists := formData[key]; exists && len(values) > 0 {
			return values[0] == "on" || values[0] == "true"
		}
		return false
	}

	// Helper function to get integer form value
	getIntValue := func(key string) int {
		if values, exists := formData[key]; exists && len(values) > 0 && values[0] != "" {
			if val, err := strconv.Atoi(values[0]); err == nil {
				return val
			}
		}
		return 0
	}

	// Override with form parameters if provided
	if hasFormValue("scan_individual_items") {
		parameters.ScanIndividualItems = true
	} else if _, exists := formData["scan_individual_items"]; exists {
		parameters.ScanIndividualItems = false
	}

	if hasFormValue("skip_hidden") {
		parameters.SkipHidden = true
	} else {
		// If checkbox not present in form, user unchecked it
		parameters.SkipHidden = false
	}

	if hasFormValue("include_sharing") {
		parameters.IncludeSharing = true
	} else if _, exists := formData["include_sharing"]; exists {
		parameters.IncludeSharing = false
	}

	// Handle numeric parameters
	if batchSize := getIntValue("batch_size"); batchSize > 0 {
		parameters.BatchSize = batchSize
	}

	if timeout := getIntValue("timeout"); timeout > 0 {
		parameters.Timeout = timeout
	}

	return parameters
}

// QueueAudit queues a new audit request with deduplication
func (s *AuditServiceImpl) QueueAudit(ctx context.Context, siteURL string, parameters *audit.AuditParameters) (*audit.AuditRequest, error) {
	s.logger.Debug("Checking for duplicate audits", "site_url", siteURL)

	// Check if audit is already running or pending for this site
	if s.IsSiteBeingAudited(siteURL) {
		s.logger.Info("Rejecting duplicate audit request", "site_url", siteURL)
		return nil, fmt.Errorf("audit already running or queued for site: %s", siteURL)
	}

	// Use the StartJob method which creates AND starts the job
	params := JobParams{
		"siteURL":     siteURL,
		"description": fmt.Sprintf("Audit: %s", siteURL),
		"parameters":  parameters,
	}

	job, err := s.jobService.StartJob(jobs.JobTypeSiteAudit, params)
	if err != nil {
		s.logger.Error("Failed to start audit job", "site_url", siteURL, "error", err)
		return nil, fmt.Errorf("failed to start job: %w", err)
	}

	// Create audit request (contains the job ID for tracking)
	request := &audit.AuditRequest{
		ID:         job.ID, // Use job ID as request ID
		SiteURL:    siteURL,
		ItemGUID:   "",
		Parameters: parameters,
		Priority:   0,
		CreatedAt:  time.Now(),
		Retries:    0,
	}

	s.logger.Info("Audit queued successfully", "job_id", job.ID, "site_url", siteURL)
	return request, nil
}

// GetAuditStatus retrieves the current status of an audit for a site
func (s *AuditServiceImpl) GetAuditStatus(siteURL string) (*audit.ActiveAudit, bool) {
	// Find the most recent audit job for this site
	allJobs := s.jobService.ListAllJobs()
	var latestJob *jobs.Job

	for _, job := range allJobs {
		if job.GetSiteURL() == siteURL && job.Type == jobs.JobTypeSiteAudit {
			if latestJob == nil || job.StartedAt.After(latestJob.StartedAt) {
				latestJob = job
			}
		}
	}

	if latestJob == nil || (!latestJob.IsActive() && !latestJob.IsComplete()) {
		return nil, false
	}

	// Convert job to active audit format
	activeAudit := &audit.ActiveAudit{
		Request: &audit.AuditRequest{
			ID:        latestJob.ID,
			SiteURL:   siteURL,
			ItemGUID:  "", // We don't store this in job, could enhance later
			CreatedAt: latestJob.StartedAt,
		},
		Status:    audit.AuditStatus(latestJob.Status), // Convert job status to audit status
		StartedAt: latestJob.StartedAt,
		JobID:     latestJob.ID,
	}

	return activeAudit, true
}

// GetActiveAudits returns all currently active audits
func (s *AuditServiceImpl) GetActiveAudits() []*audit.ActiveAudit {
	activeJobs := s.jobService.ListJobsByStatus(jobs.JobStatusRunning)
	var activeAudits []*audit.ActiveAudit

	for _, job := range activeJobs {
		if job.Type == jobs.JobTypeSiteAudit {
			activeAudit := &audit.ActiveAudit{
				Request: &audit.AuditRequest{
					ID:        job.ID,
					SiteURL:   job.GetSiteURL(),
					ItemGUID:  "", // We don't store this in job, could enhance later
					CreatedAt: job.StartedAt,
				},
				Status:    audit.AuditStatus(job.Status),
				StartedAt: job.StartedAt,
				JobID:     job.ID,
			}
			activeAudits = append(activeAudits, activeAudit)
		}
	}

	return activeAudits
}

// GetAuditHistory returns recent audit history
func (s *AuditServiceImpl) GetAuditHistory(limit int) []*audit.AuditResult {
	// Get completed, failed, and cancelled audit jobs
	allJobs := s.jobService.ListAllJobs()
	var auditResults []*audit.AuditResult

	for _, job := range allJobs {
		if job.Type == jobs.JobTypeSiteAudit &&
			(job.Status == jobs.JobStatusCompleted || job.Status == jobs.JobStatusFailed || job.Status == jobs.JobStatusCancelled) {

			duration := int64(0)
			completedAt := time.Time{} // Default to zero time
			if job.CompletedAt != nil {
				duration = job.CompletedAt.Sub(job.StartedAt).Milliseconds()
				completedAt = *job.CompletedAt
			}

			result := &audit.AuditResult{
				RequestID:   job.ID,
				SiteURL:     job.GetSiteURL(),
				Status:      audit.AuditStatus(job.Status),
				StartedAt:   job.StartedAt,
				CompletedAt: completedAt,
				Duration:    duration,
				Error:       job.Error,
				ItemsFound:  0, // Would need to enhance job to track these
				ListsFound:  0,
			}
			auditResults = append(auditResults, result)
		}
	}

	// Sort by completed time (most recent first) and limit
	if len(auditResults) > limit {
		auditResults = auditResults[:limit]
	}

	return auditResults
}

// CancelAudit cancels a running audit
func (s *AuditServiceImpl) CancelAudit(siteURL string) error {
	// Find the active audit job for this site
	runningJobs := s.jobService.ListJobsByStatus(jobs.JobStatusRunning)
	var targetJob *jobs.Job

	for _, job := range runningJobs {
		if job.GetSiteURL() == siteURL && job.Type == jobs.JobTypeSiteAudit {
			targetJob = job
			break
		}
	}

	if targetJob == nil {
		return fmt.Errorf("no active audit found for site: %s", siteURL)
	}

	// Cancel the job through the job service
	if _, err := s.jobService.CancelJob(targetJob.ID); err != nil {
		s.logger.Error("Failed to cancel job", "site_url", siteURL, "job_id", targetJob.ID, "error", err)
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	s.logger.Info("Audit cancelled", "site_url", siteURL, "job_id", targetJob.ID)
	return nil
}
