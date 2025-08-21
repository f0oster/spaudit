package events

import (
	"time"

	"spaudit/domain/jobs"
)

// JobCompletedEvent represents a job that has completed successfully
type JobCompletedEvent struct {
	Job       *jobs.Job
	Timestamp time.Time
}

// JobFailedEvent represents a job that has failed
type JobFailedEvent struct {
	Job       *jobs.Job
	Error     string
	Timestamp time.Time
}

// JobCancelledEvent represents a job that was cancelled
type JobCancelledEvent struct {
	Job       *jobs.Job
	Timestamp time.Time
}

// SiteAuditCompletedEvent represents completion of any audit on a site
type SiteAuditCompletedEvent struct {
	SiteURL   string
	Job       *jobs.Job
	Timestamp time.Time
}
