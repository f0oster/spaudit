package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"spaudit/application"
	"spaudit/domain/jobs"
	"spaudit/interfaces/web/presenters"
	"spaudit/logging"
)

// JobHandlers handles job-related HTTP endpoints with registry-based execution.
// Provides thin orchestration layer for job management operations using pluggable executors.
type JobHandlers struct {
	jobService   application.JobService
	jobPresenter *presenters.JobPresenter
	logger       *logging.Logger
}

// NewJobHandlers creates a new job handlers instance with registry-based job service.
func NewJobHandlers(
	jobService application.JobService,
	jobPresenter *presenters.JobPresenter,
) *JobHandlers {
	return &JobHandlers{
		jobService:   jobService,
		jobPresenter: jobPresenter,
		logger:       logging.Default().WithComponent("job_handler"),
	}
}


// CancelJob cancels a running job - thin orchestration with business logic in service
func (h *JobHandlers) CancelJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	if jobID == "" {
		http.Error(w, "missing job ID", http.StatusBadRequest)
		return
	}

	// Delegate to service for all business logic
	_, err := h.jobService.CancelJob(jobID)
	if err != nil {
		h.logger.Error("Failed to cancel job", "job_id", jobID, "error", err)

		// Use presenter to format error response
		w.Header().Set("Content-Type", "text/html")
		if err.Error() == "job not found" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}

		errorMessage := h.jobPresenter.FormatCancelErrorMessage(err)
		w.Write([]byte(errorMessage))
		return
	}

	h.logger.Info("Job cancellation requested", "job_id", jobID)

	// Use presenter to format success message
	w.Header().Set("Content-Type", "text/html")
	successMessage := h.jobPresenter.FormatCancelSuccessMessage()
	w.Write([]byte(successMessage))
}

// ListJobs returns all jobs as HTML or JSON - delegates to service
func (h *JobHandlers) ListJobs(w http.ResponseWriter, r *http.Request) {
	// Get all jobs using service
	jobs := h.jobService.ListAllJobs()

	// Check if this is an HTMX request for HTML
	if r.Header.Get("HX-Request") == "true" || r.Header.Get("Accept") == "text/html" {
		h.handleJobListHTML(w, r, jobs)
		return
	}

	// Default to JSON response
	h.handleJobListJSON(w, r, jobs)
}

// handleJobListHTML handles HTML response for HTMX
func (h *JobHandlers) handleJobListHTML(w http.ResponseWriter, r *http.Request, jobs []*jobs.Job) {
	w.Header().Set("Content-Type", "text/html")

	// Check if this is a partial request (from SSE trigger)
	isPartial := r.Header.Get("HX-Request") == "true"

	// Use presenter to format HTML
	html := h.jobPresenter.FormatJobListHTML(jobs, isPartial)
	w.Write([]byte(html))
}

// handleJobListJSON handles JSON response for API
func (h *JobHandlers) handleJobListJSON(w http.ResponseWriter, r *http.Request, jobs []*jobs.Job) {
	w.Header().Set("Content-Type", "application/json")

	// Use presenter to format job list
	jobListView := h.jobPresenter.FormatJobList(jobs)
	if err := json.NewEncoder(w).Encode(jobListView); err != nil {
		h.logger.Error("Failed to encode job list response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
