package presenters

import (
	"fmt"
	"time"

	"spaudit/domain/jobs"
)

// Job-related view data structures

// JobStatusView represents the status of a job for API responses
type JobStatusView struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	SiteURL     string `json:"site_url"`
	Progress    string `json:"progress"`
	Percentage  int    `json:"percentage"`
	Stage       string `json:"stage"`
	Description string `json:"description"`
	StartedAt   string `json:"started_at"`
	IsActive    bool   `json:"is_active"`
	IsComplete  bool   `json:"is_complete"`
	Error       string `json:"error,omitempty"`

	// Enhanced fields from JSON state
	CurrentItem    string            `json:"current_item,omitempty"`
	CurrentList    string            `json:"current_list,omitempty"`
	SiteTitle      string            `json:"site_title,omitempty"`
	Timeline       []JobStageDisplay `json:"timeline,omitempty"`
	Stats          JobStatsDisplay   `json:"stats,omitempty"`
	RecentMessages []string          `json:"recent_messages,omitempty"`
	StageStartedAt string            `json:"stage_started_at,omitempty"`
	StageDuration  string            `json:"stage_duration,omitempty"`
}

// JobStageDisplay represents a stage in the job timeline for UI display
type JobStageDisplay struct {
	Stage     string `json:"stage"`
	Started   string `json:"started"`
	Completed string `json:"completed,omitempty"`
	Duration  string `json:"duration,omitempty"`
}

// JobStatsDisplay represents job statistics for UI display
type JobStatsDisplay struct {
	ListsFound          int `json:"lists_found"`
	ListsProcessed      int `json:"lists_processed"`
	ItemsFound          int `json:"items_found"`
	ItemsProcessed      int `json:"items_processed"`
	PermissionsAnalyzed int `json:"permissions_analyzed"`
	SharingLinksFound   int `json:"sharing_links_found"`
	ErrorsEncountered   int `json:"errors_encountered"`
}

// JobListView represents a list of jobs
type JobListView struct {
	Jobs []*JobStatusView `json:"jobs"`
}

// JobPresenter transforms job domain data into UI-ready formats including JSON and HTML.
type JobPresenter struct{}

// NewJobPresenter creates a job presenter.
func NewJobPresenter() *JobPresenter {
	return &JobPresenter{}
}

// FormatJobStatus converts job data to view model with progress, timeline, and stats.
func (p *JobPresenter) FormatJobStatus(job *jobs.Job) *JobStatusView {
	if job == nil {
		return nil
	}

	percentage := 0
	stage := "initializing"
	description := "Job created"

	// Use rich state information
	percentage = job.State.Progress.Percentage
	if job.State.Stage != "" {
		stage = job.State.Stage
	}
	if job.State.CurrentOperation != "" {
		description = job.State.CurrentOperation
	}

	view := &JobStatusView{
		ID:          job.ID,
		Type:        string(job.Type),
		Status:      string(job.Status),
		SiteURL:     job.GetSiteURL(),
		Progress:    job.GetProgressString(),
		Percentage:  percentage,
		Stage:       stage,
		Description: description,
		StartedAt:   job.StartedAt.Format("2006-01-02 15:04:05"),
		IsActive:    job.IsActive(),
		IsComplete:  job.IsComplete(),
		Error:       job.Error,
	}

	// Add rich state details
	view.CurrentItem = job.State.Context.CurrentItemName
	view.CurrentList = job.State.Context.CurrentListTitle
	view.SiteTitle = job.State.Context.SiteTitle
	view.RecentMessages = job.State.Messages
	view.StageStartedAt = job.State.StageStartedAt.Format("2006-01-02 15:04:05")

	// Calculate how long current stage has been running
	if job.IsActive() {
		view.StageDuration = time.Since(job.State.StageStartedAt).Truncate(time.Second).String()
	}

	// Transform job timeline for display
	view.Timeline = make([]JobStageDisplay, len(job.State.Timeline))
	for i, stage := range job.State.Timeline {
		stageDisplay := JobStageDisplay{
			Stage:   stage.Stage,
			Started: stage.Started.Format("15:04:05"),
		}
		if stage.Completed != nil {
			stageDisplay.Completed = stage.Completed.Format("15:04:05")
			stageDisplay.Duration = stage.Duration
		}
		view.Timeline[i] = stageDisplay
	}

	// Transform job statistics for display
	view.Stats = JobStatsDisplay{
		ListsFound:          job.State.Stats.ListsFound,
		ListsProcessed:      job.State.Stats.ListsProcessed,
		ItemsFound:          job.State.Stats.ItemsFound,
		ItemsProcessed:      job.State.Stats.ItemsProcessed,
		PermissionsAnalyzed: job.State.Stats.PermissionsAnalyzed,
		SharingLinksFound:   job.State.Stats.SharingLinksFound,
		ErrorsEncountered:   job.State.Stats.ErrorsEncountered,
	}

	return view
}

// FormatJobNotFound creates a "not found" error view model.
func (p *JobPresenter) FormatJobNotFound() *JobStatusView {
	return &JobStatusView{
		ID:     "",
		Status: "not_found",
		Error:  "Job not found",
	}
}

// FormatJobList converts multiple jobs to list view model.
func (p *JobPresenter) FormatJobList(jobs []*jobs.Job) *JobListView {
	jobViews := make([]*JobStatusView, 0, len(jobs))

	for _, job := range jobs {
		jobView := p.FormatJobStatus(job)
		if jobView != nil {
			jobViews = append(jobViews, jobView)
		}
	}

	return &JobListView{
		Jobs: jobViews,
	}
}

// FormatJobListHTML generates HTMX-compatible HTML for job list with real-time updates.
func (p *JobPresenter) FormatJobListHTML(jobs []*jobs.Job, isPartialUpdate bool) string {
	if len(jobs) == 0 {
		content := `<div class="px-6 py-8 text-center">
			<div class="text-slate-400 text-3xl mb-3">⏱️</div>
			<h3 class="text-lg font-medium text-slate-900 mb-2">No jobs yet</h3>
			<p class="text-slate-500 text-sm">Start an audit above to see jobs here</p>
		</div>`

		if !isPartialUpdate {
			content = p.wrapWithSSEContainer(content)
		}

		return content
	}

	html := ""
	for _, job := range jobs {
		html += p.formatJobItemHTML(job)
	}

	// Add real-time update container for full page loads
	if !isPartialUpdate {
		html = p.wrapWithSSEContainer(html)
	}

	return html
}

// formatJobItemHTML formats a single job as HTML with status, progress, and context information.
func (p *JobPresenter) formatJobItemHTML(job *jobs.Job) string {
	statusClass, statusIcon := p.getJobStatusDisplay(job.Status)
	jobTypeDisplay := p.getJobTypeDisplay(job.Type)
	cancelButton := p.getCancelButtonHTML(job)
	statusDisplay := p.getJobStatusText(job.Status)

	// Build contextual information and progress details from rich state
	contextInfo := p.getJobContextHTML(job)
	progressDetail := p.getJobProgressDetailHTML(job)

	return fmt.Sprintf(`<div class="px-6 py-4 border-b border-slate-100">
		<div class="flex items-center justify-between">
			<div class="flex-1">
				<div class="font-medium text-slate-900">%s</div>
				<div class="text-sm text-slate-500">%s</div>
				<div class="text-xs text-slate-400">Job ID: %s</div>
				%s
				%s
				%s
			</div>
			<div class="text-right ml-4">
				<div class="text-sm">
					<span class="%s">%s %s</span>
					<div class="text-xs text-slate-500 mt-1">%s</div>
				</div>
			</div>
		</div>
	</div>`, jobTypeDisplay, job.GetSiteURL(), job.ID, contextInfo, progressDetail, cancelButton, statusClass, statusIcon, statusDisplay, job.GetProgressString())
}

// getJobContextHTML returns contextual information HTML badges for site, list, and item.
func (p *JobPresenter) getJobContextHTML(job *jobs.Job) string {
	contextParts := make([]string, 0, 3)

	// Add site title badge if available
	if job.State.Context.SiteTitle != "" && job.State.Context.SiteTitle != job.GetSiteURL() {
		contextParts = append(contextParts, fmt.Sprintf(`<span class="inline-flex items-center px-2 py-1 rounded-full text-xs bg-blue-100 text-blue-800">🌐 %s</span>`, job.State.Context.SiteTitle))
	}

	// Add current list badge if available
	if job.State.Context.CurrentListTitle != "" {
		contextParts = append(contextParts, fmt.Sprintf(`<span class="inline-flex items-center px-2 py-1 rounded-full text-xs bg-green-100 text-green-800">📋 %s</span>`, job.State.Context.CurrentListTitle))
	}

	// Add current item badge if available
	if job.State.Context.CurrentItemName != "" {
		contextParts = append(contextParts, fmt.Sprintf(`<span class="inline-flex items-center px-2 py-1 rounded-full text-xs bg-purple-100 text-purple-800">📄 %s</span>`, job.State.Context.CurrentItemName))
	}

	if len(contextParts) > 0 {
		contextHTML := contextParts[0]
		for i := 1; i < len(contextParts); i++ {
			contextHTML += " " + contextParts[i]
		}
		return fmt.Sprintf(`<div class="mt-2 space-x-1">%s</div>`, contextHTML)
	}

	return ""
}

// getJobProgressDetailHTML returns detailed progress statistics HTML for active jobs.
func (p *JobPresenter) getJobProgressDetailHTML(job *jobs.Job) string {
	if !job.IsActive() {
		return ""
	}

	stats := job.State.Stats
	details := make([]string, 0, 4)

	// Add lists progress if available
	if stats.ListsFound > 0 {
		details = append(details, fmt.Sprintf("Lists: %d/%d", stats.ListsProcessed, stats.ListsFound))
	}

	// Add items progress if available
	if stats.ItemsFound > 0 {
		details = append(details, fmt.Sprintf("Items: %d/%d", stats.ItemsProcessed, stats.ItemsFound))
	}

	// Add permissions count if available
	if stats.PermissionsAnalyzed > 0 {
		details = append(details, fmt.Sprintf("Permissions: %d", stats.PermissionsAnalyzed))
	}

	// Add sharing links count if available
	if stats.SharingLinksFound > 0 {
		details = append(details, fmt.Sprintf("Links: %d", stats.SharingLinksFound))
	}

	// Add errors count if any encountered
	if stats.ErrorsEncountered > 0 {
		details = append(details, fmt.Sprintf(`<span class="text-red-600">Errors: %d</span>`, stats.ErrorsEncountered))
	}

	if len(details) > 0 {
		detailsHTML := details[0]
		for i := 1; i < len(details); i++ {
			detailsHTML += " • " + details[i]
		}
		return fmt.Sprintf(`<div class="mt-1 text-xs text-slate-600">%s</div>`, detailsHTML)
	}

	return ""
}

// getJobStatusDisplay returns CSS class and icon for job status visualization.
func (p *JobPresenter) getJobStatusDisplay(status jobs.JobStatus) (string, string) {
	switch status {
	case jobs.JobStatusPending:
		return "text-gray-600", "⏳"
	case jobs.JobStatusRunning:
		return "text-blue-600", "🔄"
	case jobs.JobStatusCompleted:
		return "text-green-600", "✅"
	case jobs.JobStatusFailed:
		return "text-red-600", "❌"
	case jobs.JobStatusCancelled:
		return "text-orange-600", "⏹️"
	default:
		return "text-gray-600", "❓"
	}
}

// getJobStatusText returns user-friendly status text for display.
func (p *JobPresenter) getJobStatusText(status jobs.JobStatus) string {
	switch status {
	case jobs.JobStatusPending:
		return "Pending"
	case jobs.JobStatusRunning:
		return "Running"
	case jobs.JobStatusCompleted:
		return "Completed"
	case jobs.JobStatusFailed:
		return "Failed"
	case jobs.JobStatusCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// getJobTypeDisplay returns human-readable job type for display.
func (p *JobPresenter) getJobTypeDisplay(jobType jobs.JobType) string {
	switch jobType {
	case jobs.JobTypeSiteAudit:
		return "Site Audit"
	default:
		return string(jobType)
	}
}

// getCancelButtonHTML returns HTMX-enabled cancel button HTML for active jobs.
func (p *JobPresenter) getCancelButtonHTML(job *jobs.Job) string {
	if !job.IsActive() {
		return ""
	}

	return fmt.Sprintf(`<div class="mt-2">
		<button class="text-xs px-2 py-1 bg-red-100 hover:bg-red-200 text-red-700 rounded border border-red-300 transition-colors"
			hx-post="/jobs/%s/cancel"
			hx-target="#cancel-status-%s"
			hx-swap="innerHTML"
			hx-on::after-request="if (event.detail.xhr.status === 200) { htmx.trigger('#jobs-list', 'sse:jobs-updated'); }">
			🗑️ Cancel
		</button>
		<div id="cancel-status-%s" class="mt-1"></div>
	</div>`, job.ID, job.ID, job.ID)
}

// wrapWithSSEContainer wraps content with SSE container for HTMX real-time updates.
func (p *JobPresenter) wrapWithSSEContainer(content string) string {
	return fmt.Sprintf(`<div id="job-list" 
		       hx-ext="sse" 
		       sse-connect="/events"
		       hx-get="/jobs" 
		       hx-trigger="sse:jobs:updated"
		       hx-swap="outerHTML">%s</div>`, content)
}

// FormatCancelSuccessMessage formats success message for job cancellation.
func (p *JobPresenter) FormatCancelSuccessMessage() string {
	return `<div class="text-green-600 text-sm">✅ Job cancelled successfully</div>`
}

// FormatCancelErrorMessage formats error message for job cancellation.
func (p *JobPresenter) FormatCancelErrorMessage(err error) string {
	return fmt.Sprintf(`<div class="text-red-600 text-sm">❌ Failed to cancel job: %s</div>`, err.Error())
}

// FormatJobNotActiveMessage formats message for jobs that can't be cancelled.
func (p *JobPresenter) FormatJobNotActiveMessage() string {
	return `<div class="text-orange-600 text-sm">⚠️ Job is no longer active and cannot be cancelled</div>`
}

// FormatAuditQueuedSuccessMessage formats success message for queued audit jobs.
func (p *JobPresenter) FormatAuditQueuedSuccessMessage() string {
	return `<div class="text-green-600 text-sm">✅ Background audit queued successfully! Check the jobs section below for real-time progress.</div>`
}

// FormatAuditQueuedErrorMessage formats error message for audit queue failures.
func (p *JobPresenter) FormatAuditQueuedErrorMessage(err error) string {
	return fmt.Sprintf(`<div class="text-red-600 text-sm">❌ Failed to queue audit: %s</div>`, err.Error())
}

// FormatAuditAlreadyRunningMessage formats message when audit is already running.
func (p *JobPresenter) FormatAuditAlreadyRunningMessage() string {
	return `<div class="text-orange-600 text-sm">⚠️ An audit is already running or queued for this site. Please wait for it to complete.</div>`
}
