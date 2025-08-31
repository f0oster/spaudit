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

	done := make(chan events.JobCompletedEvent, 1)

	// Subscribe to the event
	eventBus.OnJobCompleted(func(event events.JobCompletedEvent) {
		done <- event
	})

	// Act
	testEvent := events.JobCompletedEvent{
		Job:       job,
		Timestamp: time.Now(),
	}
	eventBus.PublishJobCompleted(testEvent)

	// Assert
	select {
	case receivedEvent := <-done:
		assert.Equal(t, testEvent.Job.ID, receivedEvent.Job.ID)
		assert.Equal(t, testEvent.Job.Status, receivedEvent.Job.Status)
		assert.False(t, receivedEvent.Timestamp.IsZero())
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Handler was not called within timeout")
	}
}

func TestJobEventBus_PublishJobFailed_Success(t *testing.T) {
	// Arrange
	eventBus := NewJobEventBus()
	job := createTestJob("test-job-2", jobs.JobStatusFailed)

	done := make(chan events.JobFailedEvent, 1)

	eventBus.OnJobFailed(func(event events.JobFailedEvent) {
		done <- event
	})

	// Act
	testEvent := events.JobFailedEvent{
		Job:       job,
		Error:     "Test error message",
		Timestamp: time.Now(),
	}
	eventBus.PublishJobFailed(testEvent)

	// Assert
	select {
	case receivedEvent := <-done:
		assert.Equal(t, "test-job-2", receivedEvent.Job.ID)
		assert.Equal(t, "Test error message", receivedEvent.Error)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Handler was not called within timeout")
	}
}

func TestJobEventBus_PublishJobCancelled_Success(t *testing.T) {
	// Arrange
	eventBus := NewJobEventBus()
	job := createTestJob("test-job-3", jobs.JobStatusCancelled)

	done := make(chan events.JobCancelledEvent, 1)

	eventBus.OnJobCancelled(func(event events.JobCancelledEvent) {
		done <- event
	})

	// Act
	testEvent := events.JobCancelledEvent{
		Job:       job,
		Timestamp: time.Now(),
	}
	eventBus.PublishJobCancelled(testEvent)

	// Assert
	select {
	case receivedEvent := <-done:
		assert.Equal(t, "test-job-3", receivedEvent.Job.ID)
		assert.Equal(t, jobs.JobStatusCancelled, receivedEvent.Job.Status)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Handler was not called within timeout")
	}
}

func TestJobEventBus_PublishSiteAuditCompleted_Success(t *testing.T) {
	// Arrange
	eventBus := NewJobEventBus()
	job := createTestJob("site-audit-job", jobs.JobStatusCompleted)

	done := make(chan events.SiteAuditCompletedEvent, 1)

	eventBus.OnSiteAuditCompleted(func(event events.SiteAuditCompletedEvent) {
		done <- event
	})

	// Act
	testEvent := events.SiteAuditCompletedEvent{
		SiteURL:   "https://test.sharepoint.com",
		Job:       job,
		Timestamp: time.Now(),
	}
	eventBus.PublishSiteAuditCompleted(testEvent)

	// Assert
	select {
	case receivedEvent := <-done:
		assert.Equal(t, "https://test.sharepoint.com", receivedEvent.SiteURL)
		assert.Equal(t, "site-audit-job", receivedEvent.Job.ID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Handler was not called within timeout")
	}
}

func TestJobEventBus_MultipleHandlers_AllCalled(t *testing.T) {
	// Arrange
	eventBus := NewJobEventBus()
	job := createTestJob("multi-handler-job", jobs.JobStatusCompleted)

	var wg sync.WaitGroup
	wg.Add(3)

	handler1Done := make(chan bool, 1)
	handler2Done := make(chan bool, 1)
	handler3Done := make(chan bool, 1)

	// Subscribe multiple handlers to the same event
	eventBus.OnJobCompleted(func(event events.JobCompletedEvent) {
		handler1Done <- true
		wg.Done()
	})

	eventBus.OnJobCompleted(func(event events.JobCompletedEvent) {
		handler2Done <- true
		wg.Done()
	})

	eventBus.OnJobCompleted(func(event events.JobCompletedEvent) {
		handler3Done <- true
		wg.Done()
	})

	// Act
	testEvent := events.JobCompletedEvent{
		Job:       job,
		Timestamp: time.Now(),
	}
	eventBus.PublishJobCompleted(testEvent)

	// Wait for all handlers to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Assert all handlers were called within timeout
	select {
	case <-done:
		// Verify each handler was called
		assert.True(t, len(handler1Done) == 1, "Handler 1 should have been called")
		assert.True(t, len(handler2Done) == 1, "Handler 2 should have been called")
		assert.True(t, len(handler3Done) == 1, "Handler 3 should have been called")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Not all handlers were called within timeout")
	}
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

	completedDone := make(chan bool, 1)
	failedDone := make(chan bool, 1)
	cancelledDone := make(chan bool, 1)

	// Subscribe to different event types
	eventBus.OnJobCompleted(func(event events.JobCompletedEvent) {
		completedDone <- true
	})

	eventBus.OnJobFailed(func(event events.JobFailedEvent) {
		failedDone <- true
	})

	eventBus.OnJobCancelled(func(event events.JobCancelledEvent) {
		cancelledDone <- true
	})

	// Act - publish only JobCompleted event
	eventBus.PublishJobCompleted(events.JobCompletedEvent{
		Job:       job,
		Timestamp: time.Now(),
	})

	// Assert - only the correct handler should be called
	select {
	case <-completedDone:
		// Expected - JobCompleted handler was called
	case <-time.After(100 * time.Millisecond):
		t.Fatal("JobCompleted handler was not called within timeout")
	}

	// Verify other handlers were NOT called
	select {
	case <-failedDone:
		t.Fatal("JobFailed handler should NOT have been called")
	case <-cancelledDone:
		t.Fatal("JobCancelled handler should NOT have been called")
	case <-time.After(50 * time.Millisecond):
		// Expected - other handlers were not called
	}
}
