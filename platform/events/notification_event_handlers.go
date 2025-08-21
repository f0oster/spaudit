package events

import (
	"spaudit/domain/events"
	"spaudit/domain/jobs"
	"spaudit/logging"
)

// SSEBroadcaster defines the interface for SSE broadcasting (same as application.SSEBroadcaster)
type SSEBroadcaster interface {
	BroadcastJobUpdate(jobID string, data string)
	BroadcastJobListUpdate()
	BroadcastSitesUpdate()
	BroadcastToast(message, toastType string)
	BroadcastRichJobToast(job *jobs.Job) // Use the actual job type
}

// SiteService defines interface for site-related notifications
type SiteService interface {
	// Add methods here if needed for site-specific event handling
}

// NotificationEventHandlers handles job events and converts them to appropriate notifications
type NotificationEventHandlers struct {
	sseBroadcaster SSEBroadcaster
	siteService    SiteService
	logger         *logging.Logger
}

// NewNotificationEventHandlers creates event handlers for notifications
func NewNotificationEventHandlers(sseBroadcaster SSEBroadcaster, siteService SiteService) *NotificationEventHandlers {
	return &NotificationEventHandlers{
		sseBroadcaster: sseBroadcaster,
		siteService:    siteService,
		logger:         logging.Default().WithComponent("notification_events"),
	}
}

// RegisterHandlers registers all notification event handlers with the event bus
func (h *NotificationEventHandlers) RegisterHandlers(eventBus *JobEventBus) {
	// Register handlers for each event type
	eventBus.OnJobCompleted(h.handleJobCompleted)
	eventBus.OnJobFailed(h.handleJobFailed)
	eventBus.OnJobCancelled(h.handleJobCancelled)
	eventBus.OnSiteAuditCompleted(h.handleSiteAuditCompleted)
}

// Event handler implementations

func (h *NotificationEventHandlers) handleJobCompleted(event events.JobCompletedEvent) {
	jobID := "unknown"
	if event.Job != nil {
		jobID = event.Job.ID
	}
	h.logger.Info("Handling job completed event", "job_id", jobID)

	// Send rich toast notification for job completion
	h.sseBroadcaster.BroadcastRichJobToast(event.Job)

	// Update job list for all connected clients
	h.sseBroadcaster.BroadcastJobListUpdate()
}

func (h *NotificationEventHandlers) handleJobFailed(event events.JobFailedEvent) {
	jobID := "unknown"
	if event.Job != nil {
		jobID = event.Job.ID
	}
	h.logger.Info("Handling job failed event", "job_id", jobID, "error", event.Error)

	// Send rich toast notification for job failure
	h.sseBroadcaster.BroadcastRichJobToast(event.Job)

	// Update job list for all connected clients
	h.sseBroadcaster.BroadcastJobListUpdate()
}

func (h *NotificationEventHandlers) handleJobCancelled(event events.JobCancelledEvent) {
	jobID := "unknown"
	if event.Job != nil {
		jobID = event.Job.ID
	}
	h.logger.Info("Handling job cancelled event", "job_id", jobID)

	// Send rich toast notification for job cancellation
	h.sseBroadcaster.BroadcastRichJobToast(event.Job)

	// Update job list for all connected clients
	h.sseBroadcaster.BroadcastJobListUpdate()
}

func (h *NotificationEventHandlers) handleSiteAuditCompleted(event events.SiteAuditCompletedEvent) {
	jobID := "unknown"
	if event.Job != nil {
		jobID = event.Job.ID
	}
	h.logger.Info("Handling site audit completed event", "site_url", event.SiteURL, "job_id", jobID)

	// Update sites table when any audit completes (metadata may have changed)
	h.sseBroadcaster.BroadcastSitesUpdate()
}
