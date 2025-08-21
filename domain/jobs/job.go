package jobs

import (
	"fmt"
	"time"

	"spaudit/domain/audit"
)

// JobStatus represents the status of a job.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// JobType represents the type of job.
type JobType string

const (
	JobTypeSiteAudit JobType = "site_audit"
)

// JobProgress represents detailed progress information.
type JobProgress struct {
	Stage       string `json:"stage"`       // Current stage (e.g., "Authenticating", "Scanning Lists")
	Description string `json:"description"` // Detailed description
	Percentage  int    `json:"percentage"`  // Progress percentage (0-100)
	ItemsTotal  int    `json:"items_total"` // Total items to process (if known)
	ItemsDone   int    `json:"items_done"`  // Items processed so far
}

// JobStageInfo represents information about a stage in the job timeline.
type JobStageInfo struct {
	Stage     string     `json:"stage"`
	Started   time.Time  `json:"started"`
	Completed *time.Time `json:"completed,omitempty"`
	Duration  string     `json:"duration,omitempty"`
}

// JobContext represents contextual information about what's being processed.
type JobContext struct {
	CurrentListID    string `json:"current_list_id,omitempty"`
	CurrentListTitle string `json:"current_list_title,omitempty"`
	CurrentItemID    string `json:"current_item_id,omitempty"`
	CurrentItemName  string `json:"current_item_name,omitempty"`
	SiteID           string `json:"site_id,omitempty"`
	SiteTitle        string `json:"site_title,omitempty"`
}

// JobStats represents statistics about the job execution.
type JobStats struct {
	ListsFound          int `json:"lists_found"`
	ListsProcessed      int `json:"lists_processed"`
	ListsSkipped        int `json:"lists_skipped"`        // Lists filtered out (hidden, etc.)
	ItemsFound          int `json:"items_found"`
	ItemsProcessed      int `json:"items_processed"`
	PermissionsAnalyzed int `json:"permissions_analyzed"`
	SharingLinksFound   int `json:"sharing_links_found"`
	ErrorsEncountered   int `json:"errors_encountered"`
}

// JobContextData represents the generic interface for job-specific context data.
type JobContextData interface {
	GetType() string
}

// AuditJobContext represents context specific to audit jobs.
type AuditJobContext struct {
	SiteURL    string                 `json:"site_url"`
	ItemGUID   string                 `json:"item_guid,omitempty"`
	Parameters *audit.AuditParameters `json:"parameters,omitempty"`
}

// GetType implements JobContextData interface.
func (c AuditJobContext) GetType() string {
	return "audit"
}

// JobState represents the complete rich state of a job stored as JSON.
type JobState struct {
	Stage            string         `json:"stage"`
	StageStartedAt   time.Time      `json:"stage_started_at"`
	CurrentOperation string         `json:"current_operation"`
	CurrentItem      string         `json:"current_item,omitempty"`
	Progress         JobProgress    `json:"progress"`
	Context          JobContext     `json:"context"`
	Timeline         []JobStageInfo `json:"timeline"`
	Stats            JobStats       `json:"stats"`
	Messages         []string       `json:"messages,omitempty"` // Recent status messages
}

// Job represents a background job with progress tracking and state management.
type Job struct {
	ID          string
	Type        JobType
	Status      JobStatus
	AuditRunID  *int64 // Associated audit run ID for tracking audit results
	StartedAt   time.Time
	CompletedAt *time.Time
	State       JobState // Job state information (always initialized)
	Result      string
	Error       string
	Context     JobContextData // Generic context for job-specific data
}

// IsActive returns true if the job is still running or pending.
func (j *Job) IsActive() bool {
	return j.Status == JobStatusPending || j.Status == JobStatusRunning
}

// IsComplete returns true if the job has finished (successfully, with error, or cancelled).
func (j *Job) IsComplete() bool {
	return j.Status == JobStatusCompleted || j.Status == JobStatusFailed || j.Status == JobStatusCancelled
}

// GetAuditParameters returns the audit parameters from context, or nil if not available.
func (j *Job) GetAuditParameters() *audit.AuditParameters {
	if auditCtx, ok := j.Context.(AuditJobContext); ok {
		return auditCtx.Parameters
	}
	return nil
}

// GetSiteURL returns the site URL from audit context, or empty string if not available.
func (j *Job) GetSiteURL() string {
	if auditCtx, ok := j.Context.(AuditJobContext); ok {
		return auditCtx.SiteURL
	}
	return ""
}

// SetItemGUID sets the ItemGUID in audit context.
func (j *Job) SetItemGUID(itemGUID string) {
	if auditCtx, ok := j.Context.(AuditJobContext); ok {
		auditCtx.ItemGUID = itemGUID
		j.Context = auditCtx
	}
}

// UpdateParameters updates the audit parameters in the job context.
// This allows users to specify custom batch sizes, timeouts, and other parameters.
func (j *Job) UpdateParameters(parameters *audit.AuditParameters, constraints *audit.SharePointApiConstraints) {
	if parameters == nil {
		return
	}

	// Validate the parameters before applying
	if err := parameters.ValidateAndSetDefaults(constraints); err != nil {
		// If invalid, don't update - keep existing parameters
		return
	}

	if auditCtx, ok := j.Context.(AuditJobContext); ok {
		auditCtx.Parameters = parameters
		j.Context = auditCtx
	}
}

// GetJobTypeDisplayName returns a human-readable display name for the job type.
func (j *Job) GetJobTypeDisplayName() string {
	switch j.Type {
	case JobTypeSiteAudit:
		return "Site Audit"
	default:
		return string(j.Type)
	}
}

// Duration returns how long the job has been running, or total duration if complete.
func (j *Job) Duration() time.Duration {
	if j.CompletedAt != nil {
		return j.CompletedAt.Sub(j.StartedAt)
	}
	return time.Since(j.StartedAt)
}

// UpdateProgress updates the job progress with detailed information.
func (j *Job) UpdateProgress(stage, description string, percentage, itemsDone, itemsTotal int) {
	// Update state progress directly
	j.State.Progress = JobProgress{
		Stage:       stage,
		Description: description,
		Percentage:  percentage,
		ItemsTotal:  itemsTotal,
		ItemsDone:   itemsDone,
	}

	// Update current operation in state
	j.State.CurrentOperation = description

	// Handle stage transitions
	if j.State.Stage != stage {
		j.State.Stage = stage
		j.State.StageStartedAt = time.Now()

		// Update timeline if initialized
		if len(j.State.Timeline) > 0 {
			// Complete the previous stage
			lastStage := &j.State.Timeline[len(j.State.Timeline)-1]
			if lastStage.Completed == nil {
				now := time.Now()
				lastStage.Completed = &now
				lastStage.Duration = now.Sub(lastStage.Started).String()
			}
		}

		// Add new stage to timeline
		j.State.Timeline = append(j.State.Timeline, JobStageInfo{
			Stage:   stage,
			Started: time.Now(),
		})
	}
}

// GetProgressString returns a human-readable progress string.
func (j *Job) GetProgressString() string {
	if j.State.CurrentItem != "" {
		return fmt.Sprintf("%s: %s - %s (%d%%)",
			j.State.Stage,
			j.State.CurrentOperation,
			j.State.CurrentItem,
			j.State.Progress.Percentage)
	}

	// Show item counts if available, otherwise percentage
	if j.State.Progress.ItemsTotal > 0 {
		// Show skipped count if we have filtering happening
		if j.State.Stats.ListsSkipped > 0 && j.State.Stage == "List Processing" {
			return fmt.Sprintf("%s: %s (%d/%d lists, %d skipped)",
				j.State.Stage,
				j.State.CurrentOperation,
				j.State.Progress.ItemsDone,
				j.State.Stats.ListsProcessed + j.State.Stats.ListsSkipped, // total discovered
				j.State.Stats.ListsSkipped)
		}
		return fmt.Sprintf("%s: %s (%d/%d items)",
			j.State.Stage,
			j.State.CurrentOperation,
			j.State.Progress.ItemsDone,
			j.State.Progress.ItemsTotal)
	}

	// Show stage, description and percentage as fallback
	return fmt.Sprintf("%s: %s (%d%%)",
		j.State.Stage,
		j.State.CurrentOperation,
		j.State.Progress.Percentage)
}

// InitializeState initializes the job state with basic information and timeline.
func (j *Job) InitializeState() {
	j.State = JobState{
		Stage:            "initializing",
		StageStartedAt:   time.Now(),
		CurrentOperation: "Preparing audit...",
		Progress: JobProgress{
			Stage:       "initializing",
			Description: "Preparing audit...",
			Percentage:  0,
		},
		Context: JobContext{}, // Job context information
		Timeline: []JobStageInfo{
			{
				Stage:   "initializing",
				Started: time.Now(),
			},
		},
		Stats:    JobStats{},
		Messages: []string{},
	}
}

// UpdateState updates the job's rich state information with stage management and timeline tracking.
func (j *Job) UpdateState(stage, operation, currentItem string, percentage int, context JobContext, stats JobStats) {
	// Handle stage transitions with timeline management
	if j.State.Stage != stage {
		// Complete the previous stage in timeline
		if len(j.State.Timeline) > 0 {
			lastStage := &j.State.Timeline[len(j.State.Timeline)-1]
			if lastStage.Completed == nil {
				now := time.Now()
				lastStage.Completed = &now
				lastStage.Duration = now.Sub(lastStage.Started).String()
			}
		}

		// Add new stage to timeline
		j.State.Timeline = append(j.State.Timeline, JobStageInfo{
			Stage:   stage,
			Started: time.Now(),
		})
		j.State.Stage = stage
		j.State.StageStartedAt = time.Now()
	}

	j.State.CurrentOperation = operation
	j.State.CurrentItem = currentItem
	j.State.Progress = JobProgress{
		Stage:       stage,
		Description: operation,
		Percentage:  percentage,
		ItemsTotal:  stats.ItemsFound,
		ItemsDone:   stats.ItemsProcessed,
	}
	j.State.Context = context
	j.State.Stats = stats

	// Maintain rolling buffer of last 10 status messages
	message := fmt.Sprintf("[%s] %s", stage, operation)
	j.State.Messages = append(j.State.Messages, message)
	if len(j.State.Messages) > 10 {
		j.State.Messages = j.State.Messages[1:]
	}
}

// SetAuditRunID sets the audit run ID for this job
func (j *Job) SetAuditRunID(auditRunID int64) {
	j.AuditRunID = &auditRunID
}

// GetAuditRunID returns the audit run ID for this job, or 0 if not set
func (j *Job) GetAuditRunID() int64 {
	if j.AuditRunID == nil {
		return 0
	}
	return *j.AuditRunID
}

// HasAuditRun returns true if this job has an associated audit run
func (j *Job) HasAuditRun() bool {
	return j.AuditRunID != nil && *j.AuditRunID > 0
}

