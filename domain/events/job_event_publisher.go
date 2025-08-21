package events

// JobEventPublisher defines the interface for publishing job-related events.
type JobEventPublisher interface {
	PublishJobCompleted(event JobCompletedEvent)
	PublishJobFailed(event JobFailedEvent)
	PublishJobCancelled(event JobCancelledEvent)
	PublishSiteAuditCompleted(event SiteAuditCompletedEvent)
}
