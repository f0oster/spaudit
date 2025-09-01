package application

import (
	"context"
	"spaudit/domain/audit"
	"spaudit/domain/jobs"
)

// JobExecutor defines the interface for executing specific job types
type JobExecutor interface {
	Execute(ctx context.Context, job *jobs.Job, progressCallback ProgressCallback) error
}

// ProgressCallback is called during job execution to report progress
type ProgressCallback func(stage, description string, percentage, itemsDone, itemsTotal int)

// WorkflowFactory defines the interface for creating workflows
type WorkflowFactory interface {
	CreateAuditWorkflow(siteURL string, auditRunID int64, parameters *audit.AuditParameters) (AuditWorkflow, error)
}

// AuditWorkflow defines the interface for audit workflow operations
type AuditWorkflow interface {
	ExecuteSiteAudit(ctx context.Context, job *jobs.Job, siteURL string) (AuditWorkflowResult, error)
	SetProgressReporter(reporter ProgressReporter)
}

// ProgressReporter defines the interface for progress reporting in workflows
type ProgressReporter interface {
	ReportProgress(stage, description string, percentage int)
	ReportItemProgress(stage, description string, percentage, itemsDone, itemsTotal int)
}

// AuditWorkflowResult represents the result of an audit workflow
type AuditWorkflowResult interface {
	GetDuration() string
	GetTotalLists() int
	GetTotalItems() int64
	GetItemsWithUnique() int64
	GetContentRisk() RiskAssessment
	GetSharingRisk() SharingRiskAssessment
	GetPermissionAnalysis() PermissionRiskAssessment
}

// Risk assessment interfaces
type RiskAssessment interface {
	GetRiskLevel() string
	GetRiskScore() float64
}

type SharingRiskAssessment interface {
	GetRiskLevel() string
	GetTotalLinks() int
}

type PermissionRiskAssessment interface {
	GetRiskLevel() string
	GetRiskScore() float64
}
