package application

import (
	"spaudit/domain/jobs"
)

// UpdateNotifier defines interface for update notifications.
type UpdateNotifier interface {
	NotifyUpdate()
	NotifyJobUpdate(jobID string, job *jobs.Job)
}

// JobParams represents parameters for creating different job types.
type JobParams map[string]interface{}

// JobService provides clean job management operations with the new simplified architecture.
type JobService interface {
	// Job creation and execution (unified operation)
	StartJob(jobType jobs.JobType, params JobParams) (*jobs.Job, error)

	// Job lifecycle operations
	CreateJob(jobType jobs.JobType, siteURL, description string) (*jobs.Job, error)
	GetJob(jobID string) (*jobs.Job, bool)
	CancelJob(jobID string) (*jobs.Job, error)

	// Job listing and filtering
	ListAllJobs() []*jobs.Job
	ListJobsByType(jobType jobs.JobType) []*jobs.Job
	ListJobsByStatus(status jobs.JobStatus) []*jobs.Job

	// Job progress tracking
	UpdateJobProgress(jobID string, stage, description string, percentage, itemsDone, itemsTotal int) error

	// Notifications
	SetUpdateNotifier(notifier UpdateNotifier)
}
