package events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"spaudit/domain/events"
	"spaudit/domain/jobs"
)

// MockSSEBroadcaster for testing NotificationEventHandlers
type MockSSEBroadcaster struct {
	mock.Mock
}

func (m *MockSSEBroadcaster) BroadcastJobUpdate(jobID string, data string) {
	m.Called(jobID, data)
}

func (m *MockSSEBroadcaster) BroadcastJobListUpdate() {
	m.Called()
}

func (m *MockSSEBroadcaster) BroadcastSitesUpdate() {
	m.Called()
}

func (m *MockSSEBroadcaster) BroadcastToast(message, toastType string) {
	m.Called(message, toastType)
}

func (m *MockSSEBroadcaster) BroadcastRichJobToast(job *jobs.Job) {
	m.Called(job)
}

// MockSiteService for testing NotificationEventHandlers
type MockSiteService struct {
	mock.Mock
}

func createTestJobForHandlers(jobID string, status jobs.JobStatus) *jobs.Job {
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

func TestNotificationEventHandlers_HandleJobCompleted_Success(t *testing.T) {
	// Arrange
	mockSSE := &MockSSEBroadcaster{}
	mockSiteService := &MockSiteService{}
	handlers := NewNotificationEventHandlers(mockSSE, mockSiteService)

	job := createTestJobForHandlers("completed-job-1", jobs.JobStatusCompleted)
	event := events.JobCompletedEvent{
		Job:       job,
		Timestamp: time.Now(),
	}

	// Set expectations
	mockSSE.On("BroadcastRichJobToast", job).Return()
	mockSSE.On("BroadcastJobListUpdate").Return()

	// Act
	handlers.handleJobCompleted(event)

	// Assert
	mockSSE.AssertExpectations(t)
	mockSSE.AssertCalled(t, "BroadcastRichJobToast", job)
	mockSSE.AssertCalled(t, "BroadcastJobListUpdate")
}

func TestNotificationEventHandlers_HandleJobFailed_Success(t *testing.T) {
	// Arrange
	mockSSE := &MockSSEBroadcaster{}
	mockSiteService := &MockSiteService{}
	handlers := NewNotificationEventHandlers(mockSSE, mockSiteService)

	job := createTestJobForHandlers("failed-job-1", jobs.JobStatusFailed)
	event := events.JobFailedEvent{
		Job:       job,
		Error:     "Database connection failed",
		Timestamp: time.Now(),
	}

	// Set expectations
	mockSSE.On("BroadcastRichJobToast", job).Return()
	mockSSE.On("BroadcastJobListUpdate").Return()

	// Act
	handlers.handleJobFailed(event)

	// Assert
	mockSSE.AssertExpectations(t)
	mockSSE.AssertCalled(t, "BroadcastRichJobToast", job)
	mockSSE.AssertCalled(t, "BroadcastJobListUpdate")
}

func TestNotificationEventHandlers_HandleJobCancelled_Success(t *testing.T) {
	// Arrange
	mockSSE := &MockSSEBroadcaster{}
	mockSiteService := &MockSiteService{}
	handlers := NewNotificationEventHandlers(mockSSE, mockSiteService)

	job := createTestJobForHandlers("cancelled-job-1", jobs.JobStatusCancelled)
	event := events.JobCancelledEvent{
		Job:       job,
		Timestamp: time.Now(),
	}

	// Set expectations
	mockSSE.On("BroadcastRichJobToast", job).Return()
	mockSSE.On("BroadcastJobListUpdate").Return()

	// Act
	handlers.handleJobCancelled(event)

	// Assert
	mockSSE.AssertExpectations(t)
	mockSSE.AssertCalled(t, "BroadcastRichJobToast", job)
	mockSSE.AssertCalled(t, "BroadcastJobListUpdate")
}

func TestNotificationEventHandlers_HandleSiteAuditCompleted_Success(t *testing.T) {
	// Arrange
	mockSSE := &MockSSEBroadcaster{}
	mockSiteService := &MockSiteService{}
	handlers := NewNotificationEventHandlers(mockSSE, mockSiteService)

	job := createTestJobForHandlers("site-audit-job", jobs.JobStatusCompleted)
	event := events.SiteAuditCompletedEvent{
		SiteURL:   "https://contoso.sharepoint.com",
		Job:       job,
		Timestamp: time.Now(),
	}

	// Set expectations
	mockSSE.On("BroadcastSitesUpdate").Return()

	// Act
	handlers.handleSiteAuditCompleted(event)

	// Assert
	mockSSE.AssertExpectations(t)
	mockSSE.AssertCalled(t, "BroadcastSitesUpdate")
}

func TestNotificationEventHandlers_RegisterHandlers_AllEventsRegistered(t *testing.T) {
	// Arrange
	mockSSE := &MockSSEBroadcaster{}
	mockSiteService := &MockSiteService{}
	handlers := NewNotificationEventHandlers(mockSSE, mockSiteService)
	eventBus := NewJobEventBus()

	// Act
	handlers.RegisterHandlers(eventBus)

	// Assert - Verify handlers are registered by checking they can be called
	// We can't directly test the registration, but we can test that the event bus
	// has handlers by publishing events and seeing if they're handled

	job := createTestJobForHandlers("register-test-job", jobs.JobStatusCompleted)
	failedJob := createTestJobForHandlers("register-test-job-failed", jobs.JobStatusFailed)
	cancelledJob := createTestJobForHandlers("register-test-job-cancelled", jobs.JobStatusCancelled)

	// Set expectations for all event types and all jobs
	mockSSE.On("BroadcastRichJobToast", job).Return()
	mockSSE.On("BroadcastRichJobToast", failedJob).Return()
	mockSSE.On("BroadcastRichJobToast", cancelledJob).Return()
	mockSSE.On("BroadcastJobListUpdate").Return()
	mockSSE.On("BroadcastSitesUpdate").Return()

	// Publish events of each type
	eventBus.PublishJobCompleted(events.JobCompletedEvent{
		Job:       job,
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

	eventBus.PublishSiteAuditCompleted(events.SiteAuditCompletedEvent{
		SiteURL:   "https://test.sharepoint.com",
		Job:       job,
		Timestamp: time.Now(),
	})

	// Wait for async handlers
	time.Sleep(20 * time.Millisecond)

	// Assert all handlers were called
	mockSSE.AssertCalled(t, "BroadcastRichJobToast", job)
	mockSSE.AssertCalled(t, "BroadcastRichJobToast", failedJob)
	mockSSE.AssertCalled(t, "BroadcastRichJobToast", cancelledJob)
	mockSSE.AssertCalled(t, "BroadcastSitesUpdate")

	// Verify call counts: 3 job events + 3 job list updates + 1 site update = 7 total calls
	assert.Equal(t, 7, len(mockSSE.Calls))
	mockSSE.AssertNumberOfCalls(t, "BroadcastRichJobToast", 3)
	mockSSE.AssertNumberOfCalls(t, "BroadcastJobListUpdate", 3)
	mockSSE.AssertNumberOfCalls(t, "BroadcastSitesUpdate", 1)
}

func TestNotificationEventHandlers_HandlerWithNilJob_DoesNotPanic(t *testing.T) {
	// Arrange
	mockSSE := &MockSSEBroadcaster{}
	mockSiteService := &MockSiteService{}
	handlers := NewNotificationEventHandlers(mockSSE, mockSiteService)

	// Set expectations (handler should still be called, even with nil job)
	mockSSE.On("BroadcastRichJobToast", (*jobs.Job)(nil)).Return()
	mockSSE.On("BroadcastJobListUpdate").Return()

	// Act & Assert - should not panic with nil job
	assert.NotPanics(t, func() {
		handlers.handleJobCompleted(events.JobCompletedEvent{
			Job:       nil,
			Timestamp: time.Now(),
		})
	})

	mockSSE.AssertExpectations(t)
}

func TestNotificationEventHandlers_MultipleHandlersForSameEvent_BothCalled(t *testing.T) {
	// This tests that the event bus can handle multiple different handler instances
	// for the same event type (which happens when multiple services subscribe)

	// Arrange
	mockSSE1 := &MockSSEBroadcaster{}
	mockSSE2 := &MockSSEBroadcaster{}
	mockSiteService := &MockSiteService{}

	handlers1 := NewNotificationEventHandlers(mockSSE1, mockSiteService)
	handlers2 := NewNotificationEventHandlers(mockSSE2, mockSiteService)

	eventBus := NewJobEventBus()

	// Register both sets of handlers
	handlers1.RegisterHandlers(eventBus)
	handlers2.RegisterHandlers(eventBus)

	job := createTestJobForHandlers("multi-handler-job", jobs.JobStatusCompleted)

	// Set expectations for both mock broadcasters
	mockSSE1.On("BroadcastRichJobToast", job).Return()
	mockSSE1.On("BroadcastJobListUpdate").Return()
	mockSSE2.On("BroadcastRichJobToast", job).Return()
	mockSSE2.On("BroadcastJobListUpdate").Return()

	// Act
	eventBus.PublishJobCompleted(events.JobCompletedEvent{
		Job:       job,
		Timestamp: time.Now(),
	})

	// Wait for async handlers
	time.Sleep(20 * time.Millisecond)

	// Assert - both sets of handlers should have been called
	mockSSE1.AssertExpectations(t)
	mockSSE2.AssertExpectations(t)
}

func TestNotificationEventHandlers_EventDataIntegrity_JobDataPreserved(t *testing.T) {
	// This test verifies that job data is passed through correctly from events to handlers

	// Arrange
	mockSSE := &MockSSEBroadcaster{}
	mockSiteService := &MockSiteService{}
	handlers := NewNotificationEventHandlers(mockSSE, mockSiteService)

	// Create a job with specific data
	job := &jobs.Job{
		ID:          "integrity-test-job",
		Type:        jobs.JobTypeSiteAudit,
		Status:      jobs.JobStatusCompleted,
		Result:      "Test job for data integrity verification",
		StartedAt:   time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC),
		CompletedAt: func() *time.Time { t := time.Date(2023, 12, 1, 10, 30, 0, 0, time.UTC); return &t }(),
		Context:     jobs.AuditJobContext{SiteURL: "https://integrity-test.sharepoint.com"},
	}
	job.InitializeState()

	event := events.JobCompletedEvent{
		Job:       job,
		Timestamp: time.Date(2023, 12, 1, 10, 30, 0, 0, time.UTC),
	}

	// Capture the job passed to BroadcastRichJobToast
	var capturedJob *jobs.Job
	mockSSE.On("BroadcastRichJobToast", mock.AnythingOfType("*jobs.Job")).Run(func(args mock.Arguments) {
		capturedJob = args.Get(0).(*jobs.Job)
	}).Return()
	mockSSE.On("BroadcastJobListUpdate").Return()

	// Act
	handlers.handleJobCompleted(event)

	// Assert - verify all job data is preserved
	mockSSE.AssertExpectations(t)
	assert.NotNil(t, capturedJob, "Job should be passed to handler")
	assert.Equal(t, "integrity-test-job", capturedJob.ID)
	assert.Equal(t, jobs.JobTypeSiteAudit, capturedJob.Type)
	assert.Equal(t, "https://integrity-test.sharepoint.com", capturedJob.GetSiteURL())
	assert.Equal(t, jobs.JobStatusCompleted, capturedJob.Status)
	assert.Equal(t, "Test job for data integrity verification", capturedJob.Result)
	assert.Equal(t, job.StartedAt, capturedJob.StartedAt)
	assert.Equal(t, job.CompletedAt, capturedJob.CompletedAt)
}
