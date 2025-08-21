package events

import (
	"sync"

	"spaudit/domain/events"
	"spaudit/logging"
)

// JobEventBus provides type-safe event publishing and subscription for job-related events
type JobEventBus struct {
	mu     sync.RWMutex
	logger *logging.Logger

	// Event handler slices for each event type
	jobCompletedHandlers       []func(events.JobCompletedEvent)
	jobFailedHandlers          []func(events.JobFailedEvent)
	jobCancelledHandlers       []func(events.JobCancelledEvent)
	siteAuditCompletedHandlers []func(events.SiteAuditCompletedEvent)
}

// NewJobEventBus creates a new typed job event bus
func NewJobEventBus() *JobEventBus {
	return &JobEventBus{
		logger:                     logging.Default().WithComponent("job_event_bus"),
		jobCompletedHandlers:       make([]func(events.JobCompletedEvent), 0),
		jobFailedHandlers:          make([]func(events.JobFailedEvent), 0),
		jobCancelledHandlers:       make([]func(events.JobCancelledEvent), 0),
		siteAuditCompletedHandlers: make([]func(events.SiteAuditCompletedEvent), 0),
	}
}

// Subscribe methods for each event type

func (bus *JobEventBus) OnJobCompleted(handler func(events.JobCompletedEvent)) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.jobCompletedHandlers = append(bus.jobCompletedHandlers, handler)
}

func (bus *JobEventBus) OnJobFailed(handler func(events.JobFailedEvent)) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.jobFailedHandlers = append(bus.jobFailedHandlers, handler)
}

func (bus *JobEventBus) OnJobCancelled(handler func(events.JobCancelledEvent)) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.jobCancelledHandlers = append(bus.jobCancelledHandlers, handler)
}

func (bus *JobEventBus) OnSiteAuditCompleted(handler func(events.SiteAuditCompletedEvent)) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.siteAuditCompletedHandlers = append(bus.siteAuditCompletedHandlers, handler)
}

// Publish methods for each event type

func (bus *JobEventBus) PublishJobCompleted(event events.JobCompletedEvent) {
	bus.mu.RLock()
	handlers := make([]func(events.JobCompletedEvent), len(bus.jobCompletedHandlers))
	copy(handlers, bus.jobCompletedHandlers)
	bus.mu.RUnlock()

	// Execute handlers asynchronously to avoid blocking the publisher
	for _, handler := range handlers {
		go func(h func(events.JobCompletedEvent)) {
			defer func() {
				if r := recover(); r != nil {
					bus.logger.Error("Event handler panicked in JobCompleted",
						"job_id", event.Job.ID,
						"panic", r)
				}
			}()
			h(event)
		}(handler)
	}
}

func (bus *JobEventBus) PublishJobFailed(event events.JobFailedEvent) {
	bus.mu.RLock()
	handlers := make([]func(events.JobFailedEvent), len(bus.jobFailedHandlers))
	copy(handlers, bus.jobFailedHandlers)
	bus.mu.RUnlock()

	for _, handler := range handlers {
		go func(h func(events.JobFailedEvent)) {
			defer func() {
				if r := recover(); r != nil {
					bus.logger.Error("Event handler panicked in JobFailed",
						"job_id", event.Job.ID,
						"error", event.Error,
						"panic", r)
				}
			}()
			h(event)
		}(handler)
	}
}

func (bus *JobEventBus) PublishJobCancelled(event events.JobCancelledEvent) {
	bus.mu.RLock()
	handlers := make([]func(events.JobCancelledEvent), len(bus.jobCancelledHandlers))
	copy(handlers, bus.jobCancelledHandlers)
	bus.mu.RUnlock()

	for _, handler := range handlers {
		go func(h func(events.JobCancelledEvent)) {
			defer func() {
				if r := recover(); r != nil {
					bus.logger.Error("Event handler panicked in JobCancelled",
						"job_id", event.Job.ID,
						"panic", r)
				}
			}()
			h(event)
		}(handler)
	}
}

func (bus *JobEventBus) PublishSiteAuditCompleted(event events.SiteAuditCompletedEvent) {
	bus.mu.RLock()
	handlers := make([]func(events.SiteAuditCompletedEvent), len(bus.siteAuditCompletedHandlers))
	copy(handlers, bus.siteAuditCompletedHandlers)
	bus.mu.RUnlock()

	for _, handler := range handlers {
		go func(h func(events.SiteAuditCompletedEvent)) {
			defer func() {
				if r := recover(); r != nil {
					bus.logger.Error("Event handler panicked in SiteAuditCompleted",
						"site_url", event.SiteURL,
						"job_id", event.Job.ID,
						"panic", r)
				}
			}()
			h(event)
		}(handler)
	}
}
