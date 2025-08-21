package audit

import (
	"time"
)

// AuditStatus represents the status of an audit.
type AuditStatus string

const (
	AuditStatusPending   AuditStatus = "pending"
	AuditStatusRunning   AuditStatus = "running"
	AuditStatusCompleted AuditStatus = "completed"
	AuditStatusFailed    AuditStatus = "failed"
	AuditStatusCancelled AuditStatus = "cancelled"
)

// AuditRequest represents a request to perform an audit.
type AuditRequest struct {
	ID         string           `json:"id"`
	SiteURL    string           `json:"site_url"`
	ItemGUID   string           `json:"item_guid,omitempty"`
	Parameters *AuditParameters `json:"parameters"`
	Priority   int              `json:"priority"`
	CreatedAt  time.Time        `json:"created_at"`
	Retries    int              `json:"retries"`
}

// ActiveAudit represents an audit that is currently running.
type ActiveAudit struct {
	Request   *AuditRequest `json:"request"`
	Status    AuditStatus   `json:"status"`
	StartedAt time.Time     `json:"started_at"`
	JobID     string        `json:"job_id"`
	Progress  *Progress     `json:"progress,omitempty"`
}

// Progress represents the progress of an audit.
type Progress struct {
	Phase         string `json:"phase"`
	Message       string `json:"message"`
	ItemsAudited  int    `json:"items_audited"`
	ListsAudited  int    `json:"lists_audited"`
	TotalExpected int    `json:"total_expected"`
}

// AuditResult represents the result of a completed audit.
type AuditResult struct {
	RequestID   string      `json:"request_id"`
	SiteURL     string      `json:"site_url"`
	Status      AuditStatus `json:"status"`
	StartedAt   time.Time   `json:"started_at"`
	CompletedAt time.Time   `json:"completed_at"`
	Duration    int64       `json:"duration_ms"`
	Error       string      `json:"error,omitempty"`
	ItemsFound  int         `json:"items_found"`
	ListsFound  int         `json:"lists_found"`
}
