package repositories

import (
	"context"
	"time"

	"spaudit/domain/contracts"
	"spaudit/domain/jobs"
	"spaudit/gen/db"
)

// ScopedJobRepository wraps a JobRepository with automatic site and audit run scoping
type ScopedJobRepository struct {
	*BaseRepository
	queries     *db.Queries
	siteID      int64
	auditRunID  int64
}

// NewScopedJobRepository creates a new scoped job repository
func NewScopedJobRepository(baseRepo *BaseRepository, queries *db.Queries, siteID, auditRunID int64) contracts.JobRepository {
	return &ScopedJobRepository{
		BaseRepository: baseRepo,
		queries:       queries,
		siteID:        siteID,
		auditRunID:    auditRunID,
	}
}

// GetLastAuditDate retrieves the audit date for the scoped audit run
func (r *ScopedJobRepository) GetLastAuditDate(ctx context.Context, siteID int64) (*time.Time, error) {
	// Verify the requested siteID matches our scoped siteID
	if siteID != r.siteID {
		return nil, contracts.ErrSiteScopeMismatch
	}

	// Get the audit run date directly since we're scoped to a specific audit run
	auditRun, err := r.queries.GetAuditRun(ctx, r.auditRunID)
	if err != nil {
		return nil, err
	}

	return &auditRun.StartedAt, nil
}

// All other job operations are not supported in scoped repository
// These methods panic because scoped repositories are for reading audit data, not managing jobs

func (r *ScopedJobRepository) GetJob(ctx context.Context, jobID string) (*jobs.Job, error) {
	panic("GetJob not supported on scoped repository - use unscoped repository for job management")
}

func (r *ScopedJobRepository) ListJobs(ctx context.Context) ([]*jobs.Job, error) {
	panic("ListJobs not supported on scoped repository - use unscoped repository for job management")
}

func (r *ScopedJobRepository) ListJobsByType(ctx context.Context, jobType jobs.JobType) ([]*jobs.Job, error) {
	panic("ListJobsByType not supported on scoped repository - use unscoped repository for job management")
}

func (r *ScopedJobRepository) ListJobsByStatus(ctx context.Context, status jobs.JobStatus) ([]*jobs.Job, error) {
	panic("ListJobsByStatus not supported on scoped repository - use unscoped repository for job management")
}

func (r *ScopedJobRepository) ListActiveJobs(ctx context.Context) ([]*jobs.Job, error) {
	panic("ListActiveJobs not supported on scoped repository - use unscoped repository for job management")
}

func (r *ScopedJobRepository) CreateJob(ctx context.Context, job *jobs.Job) error {
	panic("CreateJob not supported on scoped repository - use unscoped repository for job management")
}

func (r *ScopedJobRepository) UpdateJob(ctx context.Context, job *jobs.Job) error {
	panic("UpdateJob not supported on scoped repository - use unscoped repository for job management")
}

func (r *ScopedJobRepository) UpdateJobStatus(ctx context.Context, jobID string, status jobs.JobStatus, progress *jobs.JobProgress) error {
	panic("UpdateJobStatus not supported on scoped repository - use unscoped repository for job management")
}

func (r *ScopedJobRepository) CompleteJob(ctx context.Context, jobID string, result string) error {
	panic("CompleteJob not supported on scoped repository - use unscoped repository for job management")
}

func (r *ScopedJobRepository) FailJob(ctx context.Context, jobID string, errorMsg string) error {
	panic("FailJob not supported on scoped repository - use unscoped repository for job management")
}

func (r *ScopedJobRepository) CancelJob(ctx context.Context, jobID string) error {
	panic("CancelJob not supported on scoped repository - use unscoped repository for job management")
}

func (r *ScopedJobRepository) DeleteOldJobs(ctx context.Context, olderThan time.Time) error {
	panic("DeleteOldJobs not supported on scoped repository - use unscoped repository for job management")
}