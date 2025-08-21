package events

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"spaudit/domain/events"
	"spaudit/domain/jobs"
)

func createTestJob(jobID string, status jobs.JobStatus) *jobs.Job {
	job := &jobs.Job{
		ID:        jobID,
		Type:      jobs.JobTypeSiteAudit,
		Status:    status,
		StartedAt: time.Now(),
		Context:   jobs.AuditJobContext{SiteURL: "https://test.sharepoint.com"},
	}
	job.InitializeState()
	return job
}

func TestJobEventBus_PublishJobCompleted_Success(t *testing.T) {
	// Arrange
	eventBus := NewJobEventBus()
	job := createTestJob("test-job-1", jobs.JobStatusCompleted)

	var receivedEvent events.JobCompletedEvent
	var handlerCalled bool

	// Subscribe to the event
	eventBus.OnJobCompleted(func(event events.JobCompletedEvent) {
		receivedEvent = event
		handlerCalled = true
	})

	// Act
	testEvent := events.JobCompletedEvent{
		Job:       job,
		Timestamp: time.Now(),
	}
	eventBus.PublishJobCompleted(testEvent)

	// Wait for async handler execution
	time.Sleep(10 * time.Millisecond)

	// Assert
	assert.True(t, handlerCalled, "Event handler should have been called")
	assert.Equal(t, testEvent.Job.ID, receivedEvent.Job.ID)
	assert.Equal(t, testEvent.Job.Status, receivedEvent.Job.Status)
	assert.False(t, receivedEvent.Timestamp.IsZero())
}

func TestJobEventBus_PublishJobFailed_Success(t *testing.T) {
	// Arrange
	eventBus := NewJobEventBus()
	job := createTestJob("test-job-2", jobs.JobStatusFailed)

	var receivedEvent events.JobFailedEvent
	var handlerCalled bool

	eventBus.OnJobFailed(func(event events.JobFailedEvent) {
		receivedEvent = event
		handlerCalled = true
	})

	// Act
	testEvent := events.JobFailedEvent{
		Job:       job,
		Error:     "Test error message",
		Timestamp: time.Now(),
	}
	eventBus.PublishJobFailed(testEvent)

	// Wait for async handler execution
	time.Sleep(10 * time.Millisecond)

	// Assert
	assert.True(t, handlerCalled)
	assert.Equal(t, "test-job-2", receivedEvent.Job.ID)
	assert.Equal(t, "Test error message", receivedEvent.Error)
}

func TestJobEventBus_PublishJobCancelled_Success(t *testing.T) {
	// Arrange
	eventBus := NewJobEventBus()
	job := createTestJob("test-job-3", jobs.JobStatusCancelled)

	var receivedEvent events.JobCancelledEvent
	var handlerCalled bool

	eventBus.OnJobCancelled(func(event events.JobCancelledEvent) {
		receivedEvent = event
		handlerCalled = true
	})

	// Act
	testEvent := events.JobCancelledEvent{
		Job:       job,
		Timestamp: time.Now(),
	}
	eventBus.PublishJobCancelled(testEvent)

	// Wait for async handler execution
	time.Sleep(10 * time.Millisecond)

	// Assert
	assert.True(t, handlerCalled)
	assert.Equal(t, "test-job-3", receivedEvent.Job.ID)
	assert.Equal(t, jobs.JobStatusCancelled, receivedEvent.Job.Status)
}

func TestJobEventBus_PublishSiteAuditCompleted_Success(t *testing.T) {
	// Arrange
	eventBus := NewJobEventBus()
	job := createTestJob("site-audit-job", jobs.JobStatusCompleted)

	var receivedEvent events.SiteAuditCompletedEvent
	var handlerCalled bool

	eventBus.OnSiteAuditCompleted(func(event events.SiteAuditCompletedEvent) {
		receivedEvent = event
		handlerCalled = true
	})

	// Act
	testEvent := events.SiteAuditCompletedEvent{
		SiteURL:   "https://test.sharepoint.com",
		Job:       job,
		Timestamp: time.Now(),
	}
	eventBus.PublishSiteAuditCompleted(testEvent)

	// Wait for async handler execution
	time.Sleep(10 * time.Millisecond)

	// Assert
	assert.True(t, handlerCalled)
	assert.Equal(t, "https://test.sharepoint.com", receivedEvent.SiteURL)
	assert.Equal(t, "site-audit-job", receivedEvent.Job.ID)
}

func TestJobEventBus_MultipleHandlers_AllCalled(t *testing.T) {
	// Arrange
	eventBus := NewJobEventBus()
	job := createTestJob("multi-handler-job", jobs.JobStatusCompleted)

	handler1Called := false
	handler2Called := false
	handler3Called := false

	// Subscribe multiple handlers to the same event
	eventBus.OnJobCompleted(func(event events.JobCompletedEvent) {
		handler1Called = true
	})

	eventBus.OnJobCompleted(func(event events.JobCompletedEvent) {
		handler2Called = true
	})

	eventBus.OnJobCompleted(func(event events.JobCompletedEvent) {
		handler3Called = true
	})

	// Act
	testEvent := events.JobCompletedEvent{
		Job:       job,
		Timestamp: time.Now(),
	}
	eventBus.PublishJobCompleted(testEvent)

	// Wait for all async handlers
	time.Sleep(20 * time.Millisecond)

	// Assert
	assert.True(t, handler1Called, "Handler 1 should have been called")
	assert.True(t, handler2Called, "Handler 2 should have been called")
	assert.True(t, handler3Called, "Handler 3 should have been called")
}

func TestJobEventBus_NoHandlers_DoesNotPanic(t *testing.T) {
	// Arrange
	eventBus := NewJobEventBus()
	job := createTestJob("no-handlers-job", jobs.JobStatusCompleted)

	// Act & Assert - should not panic
	require.NotPanics(t, func() {
		eventBus.PublishJobCompleted(events.JobCompletedEvent{
			Job:       job,
			Timestamp: time.Now(),
		})
	})
}

func TestJobEventBus_ConcurrentPublishing_ThreadSafe(t *testing.T) {
	// Arrange
	eventBus := NewJobEventBus()

	var receivedCount int32
	var mu sync.Mutex

	// Subscribe handler that increments counter
	eventBus.OnJobCompleted(func(event events.JobCompletedEvent) {
		mu.Lock()
		receivedCount++
		mu.Unlock()
	})

	// Act - publish events concurrently from multiple goroutines
	const numGoroutines = 10
	const eventsPerGoroutine = 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < eventsPerGoroutine; j++ {
				job := createTestJob("concurrent-job", jobs.JobStatusCompleted)
				eventBus.PublishJobCompleted(events.JobCompletedEvent{
					Job:       job,
					Timestamp: time.Now(),
				})
			}
		}(i)
	}

	wg.Wait()

	// Wait for all async handlers to complete
	time.Sleep(50 * time.Millisecond)

	// Assert
	mu.Lock()
	expectedCount := int32(numGoroutines * eventsPerGoroutine)
	assert.Equal(t, expectedCount, receivedCount, "All events should have been processed")
	mu.Unlock()
}

func TestJobEventBus_ConcurrentSubscription_ThreadSafe(t *testing.T) {
	// Arrange
	eventBus := NewJobEventBus()
	job := createTestJob("concurrent-sub-job", jobs.JobStatusCompleted)

	var handlerCount int32
	var mu sync.Mutex

	// Act - subscribe handlers concurrently
	const numHandlers = 20
	var wg sync.WaitGroup
	wg.Add(numHandlers)

	for i := 0; i < numHandlers; i++ {
		go func() {
			defer wg.Done()

			eventBus.OnJobCompleted(func(event events.JobCompletedEvent) {
				mu.Lock()
				handlerCount++
				mu.Unlock()
			})
		}()
	}

	wg.Wait()

	// Publish one event to trigger all handlers
	eventBus.PublishJobCompleted(events.JobCompletedEvent{
		Job:       job,
		Timestamp: time.Now(),
	})

	// Wait for all handlers to execute
	time.Sleep(50 * time.Millisecond)

	// Assert
	mu.Lock()
	assert.Equal(t, int32(numHandlers), handlerCount, "All handlers should have been called")
	mu.Unlock()
}

func TestJobEventBus_EventIsolation_HandlersNotCrossCalled(t *testing.T) {
	// Arrange
	eventBus := NewJobEventBus()
	job := createTestJob("isolation-job", jobs.JobStatusCompleted)

	completedCalled := false
	failedCalled := false
	cancelledCalled := false

	// Subscribe to different event types
	eventBus.OnJobCompleted(func(event events.JobCompletedEvent) {
		completedCalled = true
	})

	eventBus.OnJobFailed(func(event events.JobFailedEvent) {
		failedCalled = true
	})

	eventBus.OnJobCancelled(func(event events.JobCancelledEvent) {
		cancelledCalled = true
	})

	// Act - publish only JobCompleted event
	eventBus.PublishJobCompleted(events.JobCompletedEvent{
		Job:       job,
		Timestamp: time.Now(),
	})

	// Wait for handlers
	time.Sleep(20 * time.Millisecond)

	// Assert - only the correct handler should be called
	assert.True(t, completedCalled, "JobCompleted handler should be called")
	assert.False(t, failedCalled, "JobFailed handler should NOT be called")
	assert.False(t, cancelledCalled, "JobCancelled handler should NOT be called")
}
