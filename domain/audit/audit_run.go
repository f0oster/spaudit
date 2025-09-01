package audit

import (
	"time"
)

// AuditRun represents a single audit run execution
type AuditRun struct {
	ID          int64
	JobID       string
	SiteID      int64
	StartedAt   time.Time
	CompletedAt *time.Time
	Status      string
	Trigger     string
}

// IsCompleted returns true if the audit run has completed
func (ar *AuditRun) IsCompleted() bool {
	return ar.CompletedAt != nil
}

// GetStatus returns the display status of the audit run
func (ar *AuditRun) GetStatus() string {
	if ar.IsCompleted() {
		return "completed"
	}
	return "running"
}