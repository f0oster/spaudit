package presenters

import (
	"context"
	"strings"
	"time"

	"spaudit/domain/jobs"
	"spaudit/interfaces/web/templates/components/ui"
)

// ToastPresenter handles toast notification view logic and formatting.
type ToastPresenter struct{}

// NewToastPresenter creates a new toast presenter.
func NewToastPresenter() *ToastPresenter {
	return &ToastPresenter{}
}

// FormatToastNotification renders a toast notification using the proper template system.
func (p *ToastPresenter) FormatToastNotification(message, toastType string) (string, error) {
	ctx := context.Background()

	// Use the proper templ template system
	component := ui.ToastNotification(message, toastType)

	// Render template to string
	var buf strings.Builder
	if err := component.Render(ctx, &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// FormatRichJobToastNotification creates a rich toast notification from job data.
func (p *ToastPresenter) FormatRichJobToastNotification(job *jobs.Job) (string, error) {
	ctx := context.Background()

	// Create rich view model from job data
	toastView := p.createToastViewFromJob(job)

	// Use rich toast template
	component := ui.RichToastNotification(toastView)

	// Render template to string
	var buf strings.Builder
	if err := component.Render(ctx, &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// createToastViewFromJob transforms job domain data into toast view model.
func (p *ToastPresenter) createToastViewFromJob(job *jobs.Job) ui.ToastNotificationView {
	// Extract stats from job state
	stats := &ui.ToastStatsView{
		ListsProcessed:   job.State.Stats.ListsProcessed,
		ItemsProcessed:   job.State.Stats.ItemsProcessed,
		PermissionsFound: job.State.Stats.PermissionsAnalyzed,
		SharingLinks:     job.State.Stats.SharingLinksFound,
		ErrorsCount:      job.State.Stats.ErrorsEncountered,
	}

	// Create title and message based on job status
	var title, message string
	switch job.Status {
	case jobs.JobStatusCompleted:
		title = job.GetJobTypeDisplayName() + " Complete"
		message = "Successfully completed audit"
	case jobs.JobStatusFailed:
		title = job.GetJobTypeDisplayName() + " Failed"
		message = "Audit failed to complete"
		if job.Error != "" {
			message = job.Error
		}
	case jobs.JobStatusCancelled:
		title = job.GetJobTypeDisplayName() + " Cancelled"
		message = "Audit was cancelled"
	default:
		title = job.GetJobTypeDisplayName()
		message = string(job.Status)
	}

	// Calculate duration
	var duration string
	if job.CompletedAt != nil {
		duration = job.Duration().Round(time.Second).String()
	}

	return ui.ToastNotificationView{
		Title:     title,
		Message:   message,
		Type:      string(job.Status),
		JobType:   string(job.Type),
		Duration:  duration,
		SiteURL:   job.GetSiteURL(),
		Stats:     stats,
		Timestamp: time.Now(),
	}
}
