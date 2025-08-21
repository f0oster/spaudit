package repositories

import (
	"context"
	"database/sql"
	"time"

	"spaudit/database"
	"spaudit/domain/contracts"
	"spaudit/domain/jobs"
	"spaudit/gen/db"
	"spaudit/infrastructure/serialization"
)

// SqlcJobRepository implements contracts.JobRepository using sqlc queries with read/write separation.
type SqlcJobRepository struct {
	*BaseRepository
	serializer *serialization.JobStateSerializer
}

// NewSqlcJobRepository creates a new job repository with read/write database separation.
func NewSqlcJobRepository(database *database.Database) contracts.JobRepository {
	return &SqlcJobRepository{
		BaseRepository: NewBaseRepository(database),
		serializer:     serialization.NewJobStateSerializer(),
	}
}

// GetLastAuditDate retrieves the last completed audit date for a site.
func (r *SqlcJobRepository) GetLastAuditDate(ctx context.Context, siteID int64) (*time.Time, error) {
	// Use read queries for this read-only operation
	readQueries := r.ReadQueries()

	// Get site info to use URL as fallback for job lookup
	siteRow, err := readQueries.GetSiteByID(ctx, siteID)
	if err != nil {
		return nil, err
	}

	lastJob, err := readQueries.GetLastCompletedJobForSite(ctx, db.GetLastCompletedJobForSiteParams{
		SiteID: sql.NullInt64{
			Int64: siteID,
			Valid: true,
		},
		SiteUrl: siteRow.SiteUrl,
	})
	if err != nil {
		// Return nil if no audit history found
		return nil, nil
	}

	if lastJob.CompletedAt.Valid {
		return &lastJob.CompletedAt.Time, nil
	}

	return nil, nil
}

// ListActiveJobs retrieves all active (pending/running) jobs.
func (r *SqlcJobRepository) ListActiveJobs(ctx context.Context) ([]*jobs.Job, error) {
	rows, err := r.ReadQueries().ListActiveJobs(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertRowsToJobs(rows), nil
}

// CreateJob creates a new job in the database.
func (r *SqlcJobRepository) CreateJob(ctx context.Context, job *jobs.Job) error {
	// Get progress percentage from state
	progressPercent := int64(job.State.Progress.Percentage)

	// Serialize job state to JSON
	stateJSON, err := r.serializer.SerializeState(job.State)
	if err != nil {
		return err
	}

	// Extract audit context data
	var siteURL, itemGUID string
	if auditCtx, ok := job.Context.(jobs.AuditJobContext); ok {
		siteURL = auditCtx.SiteURL
		itemGUID = auditCtx.ItemGUID
	}

	// Use write queries for this INSERT operation
	return r.WriteQueries().CreateJob(ctx, db.CreateJobParams{
		JobID:     job.ID,
		JobType:   string(job.Type),
		Status:    string(job.Status),
		SiteID:    sql.NullInt64{}, // TODO: Map from SiteURL if needed
		SiteUrl:   siteURL,
		ItemGuid:  sql.NullString{String: itemGUID, Valid: itemGUID != ""},
		Progress:  sql.NullInt64{Int64: progressPercent, Valid: true},
		StateJson: sql.NullString{String: stateJSON, Valid: stateJSON != ""},
		StartedAt: sql.NullTime{Time: job.StartedAt, Valid: true},
	})
}

// UpdateJobStatus updates a job's status and progress.
func (r *SqlcJobRepository) UpdateJobStatus(ctx context.Context, jobID string, status jobs.JobStatus, progress *jobs.JobProgress) error {
	progressPercent := int64(0)
	if progress != nil {
		progressPercent = int64(progress.Percentage)
	}

	return r.WriteQueries().UpdateJobStatus(ctx, db.UpdateJobStatusParams{
		JobID:     jobID,
		Status:    string(status),
		Progress:  sql.NullInt64{Int64: progressPercent, Valid: progress != nil},
		StateJson: sql.NullString{Valid: false}, // Empty state JSON for this method
	})
}

// CompleteJob marks a job as completed with a result
func (r *SqlcJobRepository) CompleteJob(ctx context.Context, jobID string, result string) error {
	return r.WriteQueries().CompleteJob(ctx, db.CompleteJobParams{
		JobID:  jobID,
		Result: sql.NullString{String: result, Valid: result != ""},
	})
}

// FailJob marks a job as failed with an error message
func (r *SqlcJobRepository) FailJob(ctx context.Context, jobID string, errorMsg string) error {
	return r.WriteQueries().FailJob(ctx, db.FailJobParams{
		JobID: jobID,
		Error: sql.NullString{String: errorMsg, Valid: errorMsg != ""},
	})
}

// GetJob retrieves a single job by ID
func (r *SqlcJobRepository) GetJob(ctx context.Context, jobID string) (*jobs.Job, error) {
	// Use read queries for this SELECT operation
	row, err := r.ReadQueries().GetJob(ctx, jobID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Job not found
		}
		return nil, err
	}

	return r.convertGetJobRowToJob(row), nil
}

// ListJobs retrieves all jobs
func (r *SqlcJobRepository) ListJobs(ctx context.Context) ([]*jobs.Job, error) {
	// Use read queries for this SELECT operation
	rows, err := r.ReadQueries().ListAllJobs(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertListAllJobsRowsToJobs(rows), nil
}

// ListJobsByType retrieves jobs filtered by type
func (r *SqlcJobRepository) ListJobsByType(ctx context.Context, jobType jobs.JobType) ([]*jobs.Job, error) {
	// Get all jobs and filter by type (since no specific query exists)
	allJobs, err := r.ListJobs(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*jobs.Job, 0)
	for _, job := range allJobs {
		if job.Type == jobType {
			filtered = append(filtered, job)
		}
	}

	return filtered, nil
}

// ListJobsByStatus retrieves jobs filtered by status
func (r *SqlcJobRepository) ListJobsByStatus(ctx context.Context, status jobs.JobStatus) ([]*jobs.Job, error) {
	// Get all jobs and filter by status (since no specific query exists)
	allJobs, err := r.ListJobs(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*jobs.Job, 0)
	for _, job := range allJobs {
		if job.Status == status {
			filtered = append(filtered, job)
		}
	}

	return filtered, nil
}

// UpdateJob updates a complete job record
func (r *SqlcJobRepository) UpdateJob(ctx context.Context, job *jobs.Job) error {
	// Get progress from state
	progressPercent := int64(job.State.Progress.Percentage)

	// Serialize job state to JSON
	stateJSON, err := r.serializer.SerializeState(job.State)
	if err != nil {
		return err
	}

	// Use write queries for this UPDATE operation
	err = r.WriteQueries().UpdateJobStatus(ctx, db.UpdateJobStatusParams{
		JobID:     job.ID,
		Status:    string(job.Status),
		Progress:  sql.NullInt64{Int64: progressPercent, Valid: true},
		StateJson: sql.NullString{String: stateJSON, Valid: stateJSON != ""},
	})

	if err != nil {
		return err
	}

	// If the job is completed or failed, use the specific completion methods
	if job.Status == jobs.JobStatusCompleted && job.Result != "" {
		return r.WriteQueries().CompleteJob(ctx, db.CompleteJobParams{
			JobID:  job.ID,
			Result: sql.NullString{String: job.Result, Valid: true},
		})
	}

	if job.Status == jobs.JobStatusFailed && job.Error != "" {
		return r.WriteQueries().FailJob(ctx, db.FailJobParams{
			JobID: job.ID,
			Error: sql.NullString{String: job.Error, Valid: true},
		})
	}

	return nil
}

// CancelJob marks a job as cancelled
func (r *SqlcJobRepository) CancelJob(ctx context.Context, jobID string) error {
	return r.WriteQueries().UpdateJobStatus(ctx, db.UpdateJobStatusParams{
		JobID:     jobID,
		Status:    string(jobs.JobStatusCancelled),
		Progress:  sql.NullInt64{Valid: false},
		StateJson: sql.NullString{Valid: false},
	})
}

// DeleteOldJobs deletes jobs older than the specified time
func (r *SqlcJobRepository) DeleteOldJobs(ctx context.Context, olderThan time.Time) error {
	// Note: The SQL query uses a hardcoded '-1 day' filter,
	// so this parameter is not directly used in the current implementation
	return r.WriteQueries().DeleteOldJobs(ctx)
}

// Helper function to convert sqlc job rows to domain jobs
func (r *SqlcJobRepository) convertRowsToJobs(rows []db.ListActiveJobsRow) []*jobs.Job {
	jobList := make([]*jobs.Job, len(rows))
	for i, row := range rows {
		jobList[i] = r.convertRowToJob(row)
	}
	return jobList
}

// Helper function to convert a single sqlc job row to domain job
func (r *SqlcJobRepository) convertRowToJob(row db.ListActiveJobsRow) *jobs.Job {
	// Create audit context from database fields
	auditContext := jobs.AuditJobContext{
		SiteURL:  row.SiteUrl,
		ItemGUID: r.nullableString(row.ItemGuid),
	}

	job := &jobs.Job{
		ID:      row.JobID,
		Type:    jobs.JobType(row.JobType),
		Status:  jobs.JobStatus(row.Status),
		Context: auditContext,
		Result:  r.nullableString(row.Result),
		Error:   r.nullableString(row.Error),
	}

	// Parse started_at
	if row.StartedAt.Valid {
		job.StartedAt = row.StartedAt.Time
	}

	// Parse completed_at
	if row.CompletedAt.Valid {
		job.CompletedAt = &row.CompletedAt.Time
	}

	// Deserialize JSON state if available
	if row.StateJson.Valid && row.StateJson.String != "" {
		if state, err := r.serializer.DeserializeState(row.StateJson.String); err == nil {
			job.State = state
		} else {
			// Initialize default state if deserialization fails
			job.InitializeState()
		}
	} else {
		// Initialize default state and set progress from database
		job.InitializeState()
		if row.Progress.Valid {
			job.State.Progress.Percentage = int(row.Progress.Int64)
		}
	}

	return job
}

// Helper function for nullable strings
func (r *SqlcJobRepository) nullableString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// Helper function to convert GetJob row to domain job
func (r *SqlcJobRepository) convertGetJobRowToJob(row db.GetJobRow) *jobs.Job {
	// Create audit context from database fields
	auditContext := jobs.AuditJobContext{
		SiteURL:  row.SiteUrl,
		ItemGUID: r.nullableString(row.ItemGuid),
	}

	job := &jobs.Job{
		ID:      row.JobID,
		Type:    jobs.JobType(row.JobType),
		Status:  jobs.JobStatus(row.Status),
		Context: auditContext,
		Result:  r.nullableString(row.Result),
		Error:   r.nullableString(row.Error),
	}

	// Parse started_at
	if row.StartedAt.Valid {
		job.StartedAt = row.StartedAt.Time
	}

	// Parse completed_at
	if row.CompletedAt.Valid {
		job.CompletedAt = &row.CompletedAt.Time
	}

	// Deserialize JSON state if available
	if row.StateJson.Valid && row.StateJson.String != "" {
		if state, err := r.serializer.DeserializeState(row.StateJson.String); err == nil {
			job.State = state
		} else {
			// Initialize default state if deserialization fails
			job.InitializeState()
		}
	} else {
		// Initialize default state and set progress from database
		job.InitializeState()
		if row.Progress.Valid {
			job.State.Progress.Percentage = int(row.Progress.Int64)
		}
	}

	return job
}

// Helper function to convert ListAllJobs rows to domain jobs
func (r *SqlcJobRepository) convertListAllJobsRowsToJobs(rows []db.ListAllJobsRow) []*jobs.Job {
	jobList := make([]*jobs.Job, len(rows))
	for i, row := range rows {
		jobList[i] = r.convertListAllJobsRowToJob(row)
	}
	return jobList
}

// Helper function to convert ListAllJobs row to domain job
func (r *SqlcJobRepository) convertListAllJobsRowToJob(row db.ListAllJobsRow) *jobs.Job {
	// Create audit context from database fields
	auditContext := jobs.AuditJobContext{
		SiteURL:  row.SiteUrl,
		ItemGUID: r.nullableString(row.ItemGuid),
	}

	job := &jobs.Job{
		ID:      row.JobID,
		Type:    jobs.JobType(row.JobType),
		Status:  jobs.JobStatus(row.Status),
		Context: auditContext,
		Result:  r.nullableString(row.Result),
		Error:   r.nullableString(row.Error),
	}

	// Parse started_at
	if row.StartedAt.Valid {
		job.StartedAt = row.StartedAt.Time
	}

	// Parse completed_at
	if row.CompletedAt.Valid {
		job.CompletedAt = &row.CompletedAt.Time
	}

	// Deserialize JSON state if available
	if row.StateJson.Valid && row.StateJson.String != "" {
		if state, err := r.serializer.DeserializeState(row.StateJson.String); err == nil {
			job.State = state
		} else {
			// Initialize default state if deserialization fails
			job.InitializeState()
		}
	} else {
		// Initialize default state and set progress from database
		job.InitializeState()
		if row.Progress.Valid {
			job.State.Progress.Percentage = int(row.Progress.Int64)
		}
	}

	return job
}
