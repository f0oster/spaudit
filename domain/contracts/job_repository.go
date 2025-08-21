package contracts

import (
	"context"
	"time"

	"spaudit/domain/jobs"
)

// JobRepository defines operations for job/audit data.
type JobRepository interface {
	// GetLastAuditDate retrieves the last completed audit date for a site.
	GetLastAuditDate(ctx context.Context, siteID int64) (*time.Time, error)

	// Job management operations
	GetJob(ctx context.Context, jobID string) (*jobs.Job, error)
	ListJobs(ctx context.Context) ([]*jobs.Job, error)
	ListJobsByType(ctx context.Context, jobType jobs.JobType) ([]*jobs.Job, error)
	ListJobsByStatus(ctx context.Context, status jobs.JobStatus) ([]*jobs.Job, error)
	ListActiveJobs(ctx context.Context) ([]*jobs.Job, error)
	CreateJob(ctx context.Context, job *jobs.Job) error
	UpdateJob(ctx context.Context, job *jobs.Job) error
	UpdateJobStatus(ctx context.Context, jobID string, status jobs.JobStatus, progress *jobs.JobProgress) error
	CompleteJob(ctx context.Context, jobID string, result string) error
	FailJob(ctx context.Context, jobID string, errorMsg string) error
	CancelJob(ctx context.Context, jobID string) error
	DeleteOldJobs(ctx context.Context, olderThan time.Time) error
}
