package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"spaudit/domain/audit"
	"spaudit/domain/contracts"
	"spaudit/domain/events"
	"spaudit/domain/jobs"
	"spaudit/domain/sharepoint"
	"spaudit/gen/db"
	"spaudit/infrastructure/repositories"
	"spaudit/logging"
)

// EventPublisher publishes domain events.
type EventPublisher interface {
	PublishJobCompleted(event events.JobCompletedEvent)
	PublishJobFailed(event events.JobFailedEvent)
	PublishJobCancelled(event events.JobCancelledEvent)
	PublishSiteAuditCompleted(event events.SiteAuditCompletedEvent)
}

// JobServiceImpl implements job orchestration.
type JobServiceImpl struct {
	jobRepo      contracts.JobRepository
	auditRepo    contracts.AuditRepository
	registry     *JobExecutorRegistry
	notifier     UpdateNotifier
	eventBus     EventPublisher
	logger       *logging.Logger
	
	// Context cancellation for running jobs
	runningJobs map[string]context.CancelFunc
	jobsMutex   sync.RWMutex
}

// NewJobService creates a new job service
func NewJobService(
	jobRepo contracts.JobRepository,
	auditRepo contracts.AuditRepository,
	registry *JobExecutorRegistry,
	notifier UpdateNotifier,
	eventBus EventPublisher,
) JobService {
	return &JobServiceImpl{
		jobRepo:     jobRepo,
		auditRepo:   auditRepo,
		registry:    registry,
		notifier:    notifier,
		eventBus:    eventBus,
		logger:      logging.Default().WithComponent("job_service"),
		runningJobs: make(map[string]context.CancelFunc),
	}
}

// StartJob creates and starts a job.
func (s *JobServiceImpl) StartJob(jobType jobs.JobType, params JobParams) (*jobs.Job, error) {
	// Get executor for this job type
	executor, err := s.registry.GetExecutor(jobType)
	if err != nil {
		return nil, fmt.Errorf("cannot start job: %w", err)
	}

	// Extract common parameters
	siteURL, _ := params["siteURL"].(string)
	description, _ := params["description"].(string)
	if description == "" {
		description = fmt.Sprintf("%s job for %s", jobType, siteURL)
	}

	// Create job
	job, err := s.CreateJob(jobType, siteURL, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// Set additional job parameters
	if itemGUID, ok := params["itemGUID"].(string); ok {
		job.SetItemGUID(itemGUID)
	}

	// Set audit parameters if provided
	if auditParams, ok := params["parameters"].(*audit.AuditParameters); ok {
		constraints := audit.DefaultApiConstraints()
		job.UpdateParameters(auditParams, constraints)
		s.logger.Info("Updated job with audit parameters", "job_id", job.ID,
			"batch_size", auditParams.BatchSize, "include_sharing", auditParams.IncludeSharing)
	}

	// Start execution asynchronously
	go s.executeJobAsync(job, executor)

	s.logger.Info("Job started successfully", "job_id", job.ID, "type", jobType)
	return job, nil
}

// CreateJob creates a new job using domain factory
func (s *JobServiceImpl) CreateJob(jobType jobs.JobType, siteURL, description string) (*jobs.Job, error) {
	s.logger.Info("CreateJob called", "jobType", jobType, "siteURL", siteURL)

	// Use domain factory to create job
	jobFactory := &jobs.JobFactory{}
	job := jobFactory.CreateJob(jobType, siteURL, description)

	// Persist using repository
	ctx := context.Background()
	if err := s.jobRepo.CreateJob(ctx, job); err != nil {
		s.logger.Error("Failed to create job", "job_id", job.ID, "error", err)
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	s.logger.Info("Job created", "job_id", job.ID, "type", jobType)
	return job, nil
}

// executeJobAsync executes the job asynchronously
func (s *JobServiceImpl) executeJobAsync(job *jobs.Job, executor JobExecutor) {
	// Create cancellable context for this job
	ctx, cancel := context.WithCancel(context.Background())
	
	// Store cancel function for this job
	s.jobsMutex.Lock()
	s.runningJobs[job.ID] = cancel
	s.jobsMutex.Unlock()
	
	// Ensure cleanup on completion
	defer func() {
		s.jobsMutex.Lock()
		delete(s.runningJobs, job.ID)
		s.jobsMutex.Unlock()
	}()

	// Start job
	jobLifecycle := &jobs.JobLifecycle{}
	if err := jobLifecycle.StartJob(job); err != nil {
		s.logger.Error("Failed to start job", "job_id", job.ID, "error", err)
		s.failJob(job, err.Error())
		return
	}

	// Create audit run for audit jobs
	if job.Type == jobs.JobTypeSiteAudit {
		auditRunID, err := s.createAuditRun(ctx, job)
		if err != nil {
			s.logger.Error("Failed to create audit run", "job_id", job.ID, "error", err)
			s.failJob(job, fmt.Sprintf("Failed to create audit run: %v", err))
			return
		}
		job.SetAuditRunID(auditRunID)
		s.logger.Info("Created audit run", "job_id", job.ID, "audit_run_id", auditRunID)
	}

	// Update repository with running status
	if err := s.jobRepo.UpdateJob(ctx, job); err != nil {
		s.logger.Error("Failed to update job to running", "job_id", job.ID, "error", err)
	}

	// Notify job started
	s.notifyJobUpdate(job.ID, job)

	// Execute using the specific executor
	progressCallback := s.createProgressCallback(job)
	err := executor.Execute(ctx, job, progressCallback)

	// Handle completion
	if err != nil {
		// Check if job was cancelled via context
		if ctx.Err() == context.Canceled {
			s.logger.Info("Job was cancelled", "job_id", job.ID)
			// Job status already set to cancelled in CancelJob method
		} else {
			s.logger.Error("Job execution failed", "job_id", job.ID, "error", err)
			s.failJob(job, err.Error())
		}
	} else {
		s.logger.Info("Job execution completed", "job_id", job.ID)
		s.completeJob(job)
	}

	// Save final state and notify
	if updateErr := s.jobRepo.UpdateJob(ctx, job); updateErr != nil {
		s.logger.Error("Failed to update job final status", "job_id", job.ID, "error", updateErr)
	}

	s.notifyJobUpdate(job.ID, job)
}

// createProgressCallback creates a progress callback for job execution
func (s *JobServiceImpl) createProgressCallback(job *jobs.Job) ProgressCallback {
	return func(stage, description string, percentage, itemsDone, itemsTotal int) {
		// Update job progress
		job.UpdateProgress(stage, description, percentage, itemsDone, itemsTotal)

		// Update in repository
		ctx := context.Background()
		if err := s.jobRepo.UpdateJob(ctx, job); err != nil {
			s.logger.Error("Failed to update job progress", "job_id", job.ID, "error", err)
		}

		// Notify clients of progress update
		s.notifyJobUpdate(job.ID, job)
	}
}

// completeJob completes a job successfully
func (s *JobServiceImpl) completeJob(job *jobs.Job) {
	jobLifecycle := &jobs.JobLifecycle{}
	jobLifecycle.CompleteJob(job)
	s.logger.Info("Job completed", "job_id", job.ID)

	// Publish job completion event
	if s.eventBus != nil {
		s.eventBus.PublishJobCompleted(events.JobCompletedEvent{
			Job: job,
		})

		// If this is a site audit, also publish site audit completion event
		if job.Type == jobs.JobTypeSiteAudit {
			s.eventBus.PublishSiteAuditCompleted(events.SiteAuditCompletedEvent{
				Job:     job,
				SiteURL: job.GetSiteURL(),
			})
		}
	}
}

// failJob fails a job with an error message
func (s *JobServiceImpl) failJob(job *jobs.Job, errorMsg string) {
	jobLifecycle := &jobs.JobLifecycle{}
	jobLifecycle.FailJob(job, errorMsg)
	s.logger.Error("Job failed", "job_id", job.ID, "error", errorMsg)

	// Publish job failure event
	if s.eventBus != nil {
		s.eventBus.PublishJobFailed(events.JobFailedEvent{
			Job:   job,
			Error: errorMsg,
		})
	}
}

// GetJob retrieves job by ID
func (s *JobServiceImpl) GetJob(jobID string) (*jobs.Job, bool) {
	ctx := context.Background()
	job, err := s.jobRepo.GetJob(ctx, jobID)
	if err != nil {
		s.logger.Error("Failed to get job from repository", "job_id", jobID, "error", err)
		return nil, false
	}

	if job == nil {
		return nil, false
	}

	return job, true
}

// CancelJob cancels a running job
func (s *JobServiceImpl) CancelJob(jobID string) (*jobs.Job, error) {
	// Get job from repository
	ctx := context.Background()
	job, err := s.jobRepo.GetJob(ctx, jobID)
	if err != nil || job == nil {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	// Cancel the running context if job is active
	s.jobsMutex.Lock()
	if cancelFunc, exists := s.runningJobs[jobID]; exists {
		cancelFunc() // This cancels the context and stops the goroutine
		s.logger.Info("Cancelled running job context", "job_id", jobID)
	}
	s.jobsMutex.Unlock()

	// Use domain service to cancel job
	jobLifecycle := &jobs.JobLifecycle{}
	if err := jobLifecycle.CancelJob(job); err != nil {
		return nil, err
	}

	// Update repository
	if err := s.jobRepo.UpdateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	// Publish job cancellation event
	if s.eventBus != nil {
		s.eventBus.PublishJobCancelled(events.JobCancelledEvent{
			Job: job,
		})
	}

	// Notify clients
	s.notifyJobUpdate(job.ID, job)

	return job, nil
}

// ListAllJobs returns all jobs from repository
func (s *JobServiceImpl) ListAllJobs() []*jobs.Job {
	ctx := context.Background()
	jobList, err := s.jobRepo.ListJobs(ctx)
	if err != nil {
		s.logger.Error("Failed to list all jobs", "error", err)
		return []*jobs.Job{}
	}

	return jobList
}

// ListJobsByType returns jobs filtered by type
func (s *JobServiceImpl) ListJobsByType(jobType jobs.JobType) []*jobs.Job {
	ctx := context.Background()
	jobList, err := s.jobRepo.ListJobsByType(ctx, jobType)
	if err != nil {
		s.logger.Error("Failed to list jobs by type", "type", jobType, "error", err)
		return []*jobs.Job{}
	}

	return jobList
}

// ListJobsByStatus returns jobs filtered by status
func (s *JobServiceImpl) ListJobsByStatus(status jobs.JobStatus) []*jobs.Job {
	ctx := context.Background()
	jobList, err := s.jobRepo.ListJobsByStatus(ctx, status)
	if err != nil {
		s.logger.Error("Failed to list jobs by status", "status", status, "error", err)
		return []*jobs.Job{}
	}

	return jobList
}

// UpdateJobProgress updates job progress and notifies clients
func (s *JobServiceImpl) UpdateJobProgress(jobID string, stage, description string, percentage, itemsDone, itemsTotal int) error {
	// Get job for state update
	job, exists := s.GetJob(jobID)
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Update job progress and state
	job.UpdateProgress(stage, description, percentage, itemsDone, itemsTotal)

	// Update in repository
	ctx := context.Background()
	if err := s.jobRepo.UpdateJob(ctx, job); err != nil {
		s.logger.Error("Failed to update job progress", "job_id", jobID, "error", err)
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	// Notify clients of progress update
	s.notifyJobUpdate(job.ID, job)

	return nil
}

// SetUpdateNotifier sets the update notifier for job changes
func (s *JobServiceImpl) SetUpdateNotifier(notifier UpdateNotifier) {
	s.notifier = notifier
}

// notifyJobUpdate notifies clients of job updates
func (s *JobServiceImpl) notifyJobUpdate(jobID string, job *jobs.Job) {
	if s.notifier != nil {
		s.notifier.NotifyJobUpdate(jobID, job)
	}
}

// createAuditRun creates a new audit run using database autoincrement
func (s *JobServiceImpl) createAuditRun(ctx context.Context, job *jobs.Job) (int64, error) {
	// Get site URL from job
	siteURL := job.GetSiteURL()
	if siteURL == "" {
		return 0, fmt.Errorf("job must have a site URL")
	}

	// Get or create site first
	siteID, err := s.getOrCreateSite(ctx, siteURL)
	if err != nil {
		return 0, fmt.Errorf("failed to get/create site: %w", err)
	}

	// Create audit run with database autoincrement
	baseRepo := s.auditRepo.(*repositories.SqlcAuditRepository)
	auditRunID, err := baseRepo.WriteQueries().CreateAuditRun(ctx, db.CreateAuditRunParams{
		JobID:     job.ID,
		SiteID:    siteID,
		StartedAt: time.Now(),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create audit run: %w", err)
	}

	return auditRunID, nil
}

// getOrCreateSite gets or creates a site for the given URL
func (s *JobServiceImpl) getOrCreateSite(ctx context.Context, siteURL string) (int64, error) {
	// Check if site already exists
	site, err := s.auditRepo.GetSiteByURL(ctx, siteURL)
	if err != nil {
		return 0, fmt.Errorf("failed to query existing site: %w", err)
	}

	if site != nil {
		s.logger.Info("Site already exists in database", "site_url", siteURL, "site_id", site.ID)
		return site.ID, nil
	}

	// Create new site with placeholder title - will be updated during audit
	newSite := &sharepoint.Site{
		URL:   siteURL,
		Title: "Discovering...",
	}

	if err := s.auditRepo.SaveSite(ctx, newSite); err != nil {
		return 0, fmt.Errorf("failed to create site: %w", err)
	}

	s.logger.Info("Created new site in database", "site_url", siteURL, "site_id", newSite.ID)
	return newSite.ID, nil
}
