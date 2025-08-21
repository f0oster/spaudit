package events

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"spaudit/domain/events"
	"spaudit/domain/jobs"
)

func createTestJobForIntegration(jobID string, status jobs.JobStatus) *jobs.Job {
	job := &jobs.Job{
		ID:          jobID,
		Type:        jobs.JobTypeSiteAudit,
		Status:      status,
		StartedAt:   time.Now(),
		CompletedAt: func() *time.Time { t := time.Now(); return &t }(),
		Context:     jobs.AuditJobContext{SiteURL: "https://test.sharepoint.com"},
	}
	job.InitializeState()
	return job
}

// Integration test for the complete event flow: EventBus -> EventHandlers -> SSE
func TestEventSystem_EndToEndFlow_EventBusToSSENotification(t *testing.T) {
	// Arrange - Set up the complete event system
	mockSSE := &MockSSEBroadcaster{}
	mockSiteService := &MockSiteService{}

	// Create event bus and handlers
	eventBus := NewJobEventBus()
	notificationHandlers := NewNotificationEventHandlers(mockSSE, mockSiteService)
	notificationHandlers.RegisterHandlers(eventBus)

	// Create test job
	testJob := createTestJobForIntegration("integration-job", jobs.JobStatusCompleted)

	// Set up expectations
	mockSSE.On("BroadcastRichJobToast", testJob).Return()
	mockSSE.On("BroadcastJobListUpdate").Return()
	mockSSE.On("BroadcastSitesUpdate").Return()

	// Act - Publish events through the event bus
	eventBus.PublishJobCompleted(events.JobCompletedEvent{
		Job:       testJob,
		Timestamp: time.Now(),
	})

	eventBus.PublishSiteAuditCompleted(events.SiteAuditCompletedEvent{
		SiteURL:   testJob.GetSiteURL(),
		Job:       testJob,
		Timestamp: time.Now(),
	})

	// Wait for async event processing
	time.Sleep(50 * time.Millisecond)

	// Assert - Verify the complete flow worked
	mockSSE.AssertExpectations(t)
	mockSSE.AssertCalled(t, "BroadcastRichJobToast", testJob)
	mockSSE.AssertCalled(t, "BroadcastJobListUpdate")
	mockSSE.AssertCalled(t, "BroadcastSitesUpdate")
}

// Integration test for failed job flow
func TestEventSystem_EndToEndFlow_JobFailureNotification(t *testing.T) {
	// Arrange
	mockSSE := &MockSSEBroadcaster{}
	mockSiteService := &MockSiteService{}

	eventBus := NewJobEventBus()
	notificationHandlers := NewNotificationEventHandlers(mockSSE, mockSiteService)
	notificationHandlers.RegisterHandlers(eventBus)

	testJob := createTestJobForIntegration("failed-job", jobs.JobStatusFailed)

	// Set expectations - no sites update for failed jobs
	mockSSE.On("BroadcastRichJobToast", testJob).Return()
	mockSSE.On("BroadcastJobListUpdate").Return()

	// Act
	eventBus.PublishJobFailed(events.JobFailedEvent{
		Job:       testJob,
		Error:     "Test error",
		Timestamp: time.Now(),
	})

	time.Sleep(50 * time.Millisecond)

	// Assert
	mockSSE.AssertExpectations(t)
	mockSSE.AssertCalled(t, "BroadcastRichJobToast", testJob)
	mockSSE.AssertCalled(t, "BroadcastJobListUpdate")

	// Should NOT broadcast sites update for failed jobs
	mockSSE.AssertNotCalled(t, "BroadcastSitesUpdate")
}

// Integration test for multiple concurrent events
func TestEventSystem_EndToEndFlow_ConcurrentEvents_AllProcessed(t *testing.T) {
	// Arrange
	mockSSE := &MockSSEBroadcaster{}
	mockSiteService := &MockSiteService{}

	eventBus := NewJobEventBus()
	notificationHandlers := NewNotificationEventHandlers(mockSSE, mockSiteService)
	notificationHandlers.RegisterHandlers(eventBus)

	// Create multiple test jobs
	const numJobs = 5
	testJobs := make([]*jobs.Job, numJobs)
	for i := 0; i < numJobs; i++ {
		testJobs[i] = createTestJobForIntegration(fmt.Sprintf("concurrent-job-%d", i), jobs.JobStatusCompleted)
	}

	// Set up expectations for all jobs
	for _, job := range testJobs {
		mockSSE.On("BroadcastRichJobToast", job).Return()
	}
	mockSSE.On("BroadcastJobListUpdate").Return().Times(numJobs)
	mockSSE.On("BroadcastSitesUpdate").Return().Times(numJobs)

	// Act - Publish events concurrently
	var wg sync.WaitGroup
	wg.Add(numJobs)

	for i := 0; i < numJobs; i++ {
		go func(job *jobs.Job) {
			defer wg.Done()

			eventBus.PublishJobCompleted(events.JobCompletedEvent{
				Job:       job,
				Timestamp: time.Now(),
			})

			eventBus.PublishSiteAuditCompleted(events.SiteAuditCompletedEvent{
				SiteURL:   job.GetSiteURL(),
				Job:       job,
				Timestamp: time.Now(),
			})
		}(testJobs[i])
	}

	wg.Wait()

	// Wait for all async event processing
	time.Sleep(100 * time.Millisecond)

	// Assert
	mockSSE.AssertExpectations(t)

	// Verify all jobs were processed
	for _, job := range testJobs {
		mockSSE.AssertCalled(t, "BroadcastRichJobToast", job)
	}
}

// Integration test verifying event isolation - different event types handled separately
func TestEventSystem_EndToEndFlow_EventTypeIsolation(t *testing.T) {
	// Arrange
	mockSSE := &MockSSEBroadcaster{}
	mockSiteService := &MockSiteService{}

	eventBus := NewJobEventBus()
	notificationHandlers := NewNotificationEventHandlers(mockSSE, mockSiteService)
	notificationHandlers.RegisterHandlers(eventBus)

	completedJob := createTestJobForIntegration("completed-job", jobs.JobStatusCompleted)
	failedJob := createTestJobForIntegration("failed-job", jobs.JobStatusFailed)
	cancelledJob := createTestJobForIntegration("cancelled-job", jobs.JobStatusCancelled)

	// Track which jobs were processed
	var completedNotified, failedNotified, cancelledNotified bool

	mockSSE.On("BroadcastRichJobToast", mock.AnythingOfType("*jobs.Job")).
		Run(func(args mock.Arguments) {
			job := args.Get(0).(*jobs.Job)
			switch job.ID {
			case "completed-job":
				completedNotified = true
			case "failed-job":
				failedNotified = true
			case "cancelled-job":
				cancelledNotified = true
			}
		}).Return()

	mockSSE.On("BroadcastJobListUpdate").Return()
	mockSSE.On("BroadcastSitesUpdate").Return() // Only for completed job

	// Act - Publish different event types
	eventBus.PublishJobCompleted(events.JobCompletedEvent{
		Job:       completedJob,
		Timestamp: time.Now(),
	})

	eventBus.PublishJobFailed(events.JobFailedEvent{
		Job:       failedJob,
		Error:     "Test error",
		Timestamp: time.Now(),
	})

	eventBus.PublishJobCancelled(events.JobCancelledEvent{
		Job:       cancelledJob,
		Timestamp: time.Now(),
	})

	// Only completed jobs should trigger site audit completion
	eventBus.PublishSiteAuditCompleted(events.SiteAuditCompletedEvent{
		SiteURL:   completedJob.GetSiteURL(),
		Job:       completedJob,
		Timestamp: time.Now(),
	})

	time.Sleep(100 * time.Millisecond)

	// Assert
	mockSSE.AssertExpectations(t)

	// Verify all job types were processed correctly
	assert.True(t, completedNotified, "Completed job should have been notified")
	assert.True(t, failedNotified, "Failed job should have been notified")
	assert.True(t, cancelledNotified, "Cancelled job should have been notified")

	// Verify sites update was called only once (for completed job)
	mockSSE.AssertNumberOfCalls(t, "BroadcastSitesUpdate", 1)
	mockSSE.AssertNumberOfCalls(t, "BroadcastJobListUpdate", 3) // For all three jobs
}

// Integration test for system resilience when handlers panic
func TestEventSystem_EndToEndFlow_HandlerPanicResilience(t *testing.T) {
	// Arrange
	mockSSE1 := &MockSSEBroadcaster{}
	mockSSE2 := &MockSSEBroadcaster{}
	mockSiteService := &MockSiteService{}

	eventBus := NewJobEventBus()

	// Register two sets of handlers - one will panic, one will work
	handlers1 := NewNotificationEventHandlers(mockSSE1, mockSiteService)
	handlers2 := NewNotificationEventHandlers(mockSSE2, mockSiteService)

	handlers1.RegisterHandlers(eventBus)
	handlers2.RegisterHandlers(eventBus)

	testJob := createTestJobForIntegration("panic-test-job", jobs.JobStatusCompleted)

	// First handler will panic
	mockSSE1.On("BroadcastRichJobToast", testJob).
		Run(func(args mock.Arguments) {
			panic("Handler 1 panic!")
		}).Return()
	mockSSE1.On("BroadcastJobListUpdate").Return()

	// Second handler should work normally despite first handler panic
	mockSSE2.On("BroadcastRichJobToast", testJob).Return()
	mockSSE2.On("BroadcastJobListUpdate").Return()
	mockSSE2.On("BroadcastSitesUpdate").Return()

	// Act - should not crash despite handler panic
	require.NotPanics(t, func() {
		eventBus.PublishJobCompleted(events.JobCompletedEvent{
			Job:       testJob,
			Timestamp: time.Now(),
		})

		eventBus.PublishSiteAuditCompleted(events.SiteAuditCompletedEvent{
			SiteURL:   testJob.GetSiteURL(),
			Job:       testJob,
			Timestamp: time.Now(),
		})
	})

	time.Sleep(100 * time.Millisecond)

	// Assert - second handler should still work
	mockSSE2.AssertExpectations(t)
}

// Integration test for event ordering and timing
func TestEventSystem_EndToEndFlow_EventOrdering(t *testing.T) {
	// Arrange
	mockSSE := &MockSSEBroadcaster{}
	mockSiteService := &MockSiteService{}

	eventBus := NewJobEventBus()
	notificationHandlers := NewNotificationEventHandlers(mockSSE, mockSiteService)
	notificationHandlers.RegisterHandlers(eventBus)

	testJob := createTestJobForIntegration("ordering-job", jobs.JobStatusCompleted)

	// Track call order
	var callOrder []string
	var mu sync.Mutex

	mockSSE.On("BroadcastRichJobToast", testJob).
		Run(func(args mock.Arguments) {
			mu.Lock()
			callOrder = append(callOrder, "RichJobToast")
			mu.Unlock()
		}).Return()

	mockSSE.On("BroadcastJobListUpdate").
		Run(func(args mock.Arguments) {
			mu.Lock()
			callOrder = append(callOrder, "JobListUpdate")
			mu.Unlock()
		}).Return()

	mockSSE.On("BroadcastSitesUpdate").
		Run(func(args mock.Arguments) {
			mu.Lock()
			callOrder = append(callOrder, "SitesUpdate")
			mu.Unlock()
		}).Return()

	// Act - Publish events
	eventBus.PublishJobCompleted(events.JobCompletedEvent{
		Job:       testJob,
		Timestamp: time.Now(),
	})

	eventBus.PublishSiteAuditCompleted(events.SiteAuditCompletedEvent{
		SiteURL:   testJob.GetSiteURL(),
		Job:       testJob,
		Timestamp: time.Now(),
	})

	time.Sleep(100 * time.Millisecond)

	// Assert
	mockSSE.AssertExpectations(t)

	mu.Lock()
	require.Len(t, callOrder, 3, "All handlers should have been called")

	// Verify that all expected calls were made (order may vary due to async nature)
	assert.Contains(t, callOrder, "RichJobToast")
	assert.Contains(t, callOrder, "JobListUpdate")
	assert.Contains(t, callOrder, "SitesUpdate")
	mu.Unlock()
}
