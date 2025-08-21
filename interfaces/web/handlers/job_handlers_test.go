package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"spaudit/application"
	"spaudit/domain/jobs"
	"spaudit/interfaces/web/presenters"
)

// Mock implementations for testing
type MockJobService struct {
	mock.Mock
}

func (m *MockJobService) StartJob(jobType jobs.JobType, params application.JobParams) (*jobs.Job, error) {
	args := m.Called(jobType, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jobs.Job), args.Error(1)
}

func (m *MockJobService) CreateJob(jobType jobs.JobType, siteURL, description string) (*jobs.Job, error) {
	args := m.Called(jobType, siteURL, description)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jobs.Job), args.Error(1)
}

func (m *MockJobService) GetJob(jobID string) (*jobs.Job, bool) {
	args := m.Called(jobID)
	return args.Get(0).(*jobs.Job), args.Bool(1)
}

func (m *MockJobService) CancelJob(jobID string) (*jobs.Job, error) {
	args := m.Called(jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jobs.Job), args.Error(1)
}

func (m *MockJobService) ListAllJobs() []*jobs.Job {
	args := m.Called()
	return args.Get(0).([]*jobs.Job)
}

func (m *MockJobService) ListJobsByType(jobType jobs.JobType) []*jobs.Job {
	args := m.Called(jobType)
	return args.Get(0).([]*jobs.Job)
}

func (m *MockJobService) ListJobsByStatus(status jobs.JobStatus) []*jobs.Job {
	args := m.Called(status)
	return args.Get(0).([]*jobs.Job)
}

func (m *MockJobService) UpdateJobProgress(jobID string, stage, description string, percentage, itemsDone, itemsTotal int) error {
	args := m.Called(jobID, stage, description, percentage, itemsDone, itemsTotal)
	return args.Error(0)
}

func (m *MockJobService) SetUpdateNotifier(notifier application.UpdateNotifier) {
	m.Called(notifier)
}

func TestJobHandlers_StartAudit(t *testing.T) {
	// Setup
	mockJobService := new(MockJobService)
	jobPresenter := presenters.NewJobPresenter()
	handlers := NewJobHandlers(mockJobService, jobPresenter)

	// Test: Successful audit start
	t.Run("successful audit start", func(t *testing.T) {
		expectedJob := &jobs.Job{
			ID:      "test-job-123",
			Type:    jobs.JobTypeSiteAudit,
			Status:  jobs.JobStatusPending,
			Context: jobs.AuditJobContext{SiteURL: "https://example.sharepoint.com/sites/test"},
		}
		expectedJob.InitializeState()

		mockJobService.On("StartJob",
			jobs.JobTypeSiteAudit,
			mock.AnythingOfType("application.JobParams")).Return(expectedJob, nil)

		// Create form data
		formData := url.Values{}
		formData.Set("site_url", "https://example.sharepoint.com/sites/test")
		formData.Set("scan_individual_items", "on")

		req := httptest.NewRequest(http.MethodPost, "/audit", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()

		// Execute
		handlers.StartAudit(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/html", w.Header().Get("Content-Type"))
		assert.Contains(t, w.Body.String(), "✅")
		assert.Contains(t, w.Body.String(), "queued successfully")

		mockJobService.AssertExpectations(t)
	})

	// Test: Missing site_url
	t.Run("missing site_url", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/audit", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()

		handlers.StartAudit(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Test: Audit service error
	t.Run("audit service error", func(t *testing.T) {
		// Create fresh mock to avoid interference
		freshMockJobService := new(MockJobService)
		freshHandlers := NewJobHandlers(freshMockJobService, jobPresenter)

		freshMockJobService.On("StartJob",
			jobs.JobTypeSiteAudit,
			mock.AnythingOfType("application.JobParams")).Return((*jobs.Job)(nil), assert.AnError)

		formData := url.Values{}
		formData.Set("site_url", "https://error.sharepoint.com/sites/test")

		req := httptest.NewRequest(http.MethodPost, "/audit", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()

		freshHandlers.StartAudit(w, req)

		assert.Equal(t, http.StatusOK, w.Code) // HTML error response
		assert.Contains(t, w.Body.String(), "❌")
		assert.Contains(t, w.Body.String(), "Failed to queue audit")
		freshMockJobService.AssertExpectations(t)
	})

	mockJobService.AssertExpectations(t)
}

func TestJobHandlers_GetJobStatus(t *testing.T) {
	// Setup
	mockJobService := new(MockJobService)
	jobPresenter := presenters.NewJobPresenter()
	handlers := NewJobHandlers(mockJobService, jobPresenter)

	// Test: Job exists
	t.Run("job exists", func(t *testing.T) {
		existingJob := &jobs.Job{
			ID:      "test-job-123",
			Type:    jobs.JobTypeSiteAudit,
			Status:  jobs.JobStatusRunning,
			Context: jobs.AuditJobContext{SiteURL: "https://example.sharepoint.com/sites/test"},
		}
		existingJob.InitializeState()

		mockJobService.On("GetJob", "test-job-123").Return(existingJob, true)

		// Create request with jobID in URL
		req := httptest.NewRequest(http.MethodGet, "/jobs/test-job-123", nil)
		w := httptest.NewRecorder()

		// Setup chi context with URL parameter
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("jobID", "test-job-123")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Execute
		handlers.GetJobStatus(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response presenters.JobStatusView
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "test-job-123", response.ID)
		assert.Equal(t, "site_audit", response.Type)
	})

	// Test: Job not found
	t.Run("job not found", func(t *testing.T) {
		mockJobService.On("GetJob", "nonexistent").Return((*jobs.Job)(nil), false)

		req := httptest.NewRequest(http.MethodGet, "/jobs/nonexistent", nil)
		w := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("jobID", "nonexistent")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handlers.GetJobStatus(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response presenters.JobStatusView
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "not_found", response.Status)
	})

	// Test: Missing job ID
	t.Run("missing job ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
		w := httptest.NewRecorder()

		handlers.GetJobStatus(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	mockJobService.AssertExpectations(t)
}

func TestJobHandlers_CancelJob(t *testing.T) {
	// Setup
	mockJobService := new(MockJobService)
	jobPresenter := presenters.NewJobPresenter()
	handlers := NewJobHandlers(mockJobService, jobPresenter)

	// Test: Successful cancellation
	t.Run("successful cancellation", func(t *testing.T) {
		activeJob := &jobs.Job{
			ID:      "active-job-123",
			Type:    jobs.JobTypeSiteAudit,
			Status:  jobs.JobStatusRunning,
			Context: jobs.AuditJobContext{SiteURL: "https://example.sharepoint.com/sites/test"},
		}
		activeJob.InitializeState()

		mockJobService.On("CancelJob", "active-job-123").Return(activeJob, nil)

		req := httptest.NewRequest(http.MethodPost, "/jobs/active-job-123/cancel", nil)
		w := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("jobID", "active-job-123")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handlers.CancelJob(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/html", w.Header().Get("Content-Type"))
		assert.Contains(t, w.Body.String(), "✅ Job cancelled successfully")
	})

	// Test: Job not found
	t.Run("job not found", func(t *testing.T) {
		// Create fresh mock to avoid interference
		freshMockJobService := new(MockJobService)
		freshHandlers := NewJobHandlers(freshMockJobService, jobPresenter)

		freshMockJobService.On("CancelJob", "nonexistent").Return((*jobs.Job)(nil), fmt.Errorf("job not found"))

		req := httptest.NewRequest(http.MethodPost, "/jobs/nonexistent/cancel", nil)
		w := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("jobID", "nonexistent")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		freshHandlers.CancelJob(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "❌")
		assert.Contains(t, w.Body.String(), "not found")

		freshMockJobService.AssertExpectations(t)
	})

	// Test: Job not active
	t.Run("job not active", func(t *testing.T) {
		// Create fresh mock to avoid interference
		freshMockJobService := new(MockJobService)
		freshHandlers := NewJobHandlers(freshMockJobService, jobPresenter)

		freshMockJobService.On("CancelJob", "completed-job-123").Return((*jobs.Job)(nil), fmt.Errorf("job is no longer active"))

		req := httptest.NewRequest(http.MethodPost, "/jobs/completed-job-123/cancel", nil)
		w := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("jobID", "completed-job-123")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		freshHandlers.CancelJob(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "❌")
		assert.Contains(t, w.Body.String(), "no longer active")

		freshMockJobService.AssertExpectations(t)
	})

	mockJobService.AssertExpectations(t)
}

func TestJobHandlers_ListJobs(t *testing.T) {
	// Setup
	mockJobService := new(MockJobService)
	jobPresenter := presenters.NewJobPresenter()
	handlers := NewJobHandlers(mockJobService, jobPresenter)

	testJobs := []*jobs.Job{
		func() *jobs.Job {
			job := &jobs.Job{
				ID:      "job1",
				Type:    jobs.JobTypeSiteAudit,
				Status:  jobs.JobStatusRunning,
				Context: jobs.AuditJobContext{SiteURL: "https://example.sharepoint.com/sites/test1"},
			}
			job.InitializeState()
			return job
		}(),
		func() *jobs.Job {
			job := &jobs.Job{
				ID:      "job2",
				Type:    jobs.JobTypeSiteAudit,
				Status:  jobs.JobStatusCompleted,
				Context: jobs.AuditJobContext{SiteURL: "https://example.sharepoint.com/sites/test2"},
			}
			job.InitializeState()
			return job
		}(),
	}

	// Test: JSON response
	t.Run("JSON response", func(t *testing.T) {
		mockJobService.On("ListAllJobs").Return(testJobs)

		req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		handlers.ListJobs(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response presenters.JobListView
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response.Jobs, 2)
		assert.Equal(t, "job1", response.Jobs[0].ID)
		assert.Equal(t, "job2", response.Jobs[1].ID)
	})

	// Test: HTML response (HTMX)
	t.Run("HTML response", func(t *testing.T) {
		mockJobService.On("ListAllJobs").Return(testJobs)

		req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
		req.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()

		handlers.ListJobs(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/html", w.Header().Get("Content-Type"))
		assert.Contains(t, w.Body.String(), "job1")
		assert.Contains(t, w.Body.String(), "job2")
		assert.Contains(t, w.Body.String(), "Site Audit")
	})

	// Test: Empty job list
	t.Run("empty job list", func(t *testing.T) {
		// Create fresh mocks to avoid interference
		freshMockJobService := new(MockJobService)
		freshHandlers := NewJobHandlers(freshMockJobService, jobPresenter)

		freshMockJobService.On("ListAllJobs").Return([]*jobs.Job{})

		req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
		req.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()

		freshHandlers.ListJobs(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "No jobs yet")
		assert.Contains(t, w.Body.String(), "⏱️")

		freshMockJobService.AssertExpectations(t)
	})

	mockJobService.AssertExpectations(t)
}
