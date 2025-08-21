package presenters

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"spaudit/domain/jobs"
)

// Helper to create test job data
func createTestJob(id string, jobType jobs.JobType, status jobs.JobStatus) *jobs.Job {
	job := &jobs.Job{
		ID:        id,
		Type:      jobType,
		Status:    status,
		StartedAt: time.Now(),
		Context:   jobs.AuditJobContext{SiteURL: "https://test.com"},
	}
	job.InitializeState()
	return job
}

func createTestJobWithProgress(id string, percentage int, stage, description string) *jobs.Job {
	job := createTestJob(id, jobs.JobTypeSiteAudit, jobs.JobStatusRunning)
	job.UpdateProgress(stage, description, percentage, percentage*2, 200)
	return job
}

func TestJobPresenter_FormatJobStatus_BasicFields(t *testing.T) {
	// Arrange
	presenter := NewJobPresenter()
	job := createTestJob("job-123", jobs.JobTypeSiteAudit, jobs.JobStatusRunning)

	// Act
	result := presenter.FormatJobStatus(job)

	// Assert - Test presentation outcomes
	require.NotNil(t, result)
	assert.Equal(t, "job-123", result.ID)
	assert.Equal(t, string(jobs.JobTypeSiteAudit), result.Type)
	assert.Equal(t, string(jobs.JobStatusRunning), result.Status)
	assert.Equal(t, "https://test.com", result.SiteURL)
	assert.True(t, result.IsActive)
	assert.False(t, result.IsComplete)
}

func TestJobPresenter_FormatJobStatus_ProgressFormatting(t *testing.T) {
	tests := []struct {
		name             string
		job              *jobs.Job
		expectPercentage int
		expectStage      string
	}{
		{
			name:             "job_with_progress",
			job:              createTestJobWithProgress("with-progress", 75, "processing", "Scanning items"),
			expectPercentage: 75,
			expectStage:      "processing",
		},
		{
			name:             "job_without_progress",
			job:              createTestJob("no-progress", jobs.JobTypeSiteAudit, jobs.JobStatusPending),
			expectPercentage: 0,
			expectStage:      "initializing",
		},
		{
			name: "completed_job",
			job: func() *jobs.Job {
				job := createTestJob("completed", jobs.JobTypeSiteAudit, jobs.JobStatusCompleted)
				completedAt := time.Now()
				job.CompletedAt = &completedAt
				return job
			}(),
			expectPercentage: 0,
			expectStage:      "initializing",
		},
	}

	presenter := NewJobPresenter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := presenter.FormatJobStatus(tt.job)

			require.NotNil(t, result)
			assert.Equal(t, tt.expectPercentage, result.Percentage)
			assert.Equal(t, tt.expectStage, result.Stage)
		})
	}
}

func TestJobPresenter_FormatJobStatus_WithState(t *testing.T) {
	// Arrange
	presenter := NewJobPresenter()
	job := createTestJob("job-with-state", jobs.JobTypeSiteAudit, jobs.JobStatusRunning)

	// Add rich state information
	job.State = jobs.JobState{
		Stage:            "processing_lists",
		CurrentOperation: "Processing SharePoint lists",
		CurrentItem:      "Documents Library",
		StageStartedAt:   time.Now().Add(-5 * time.Minute),
		Progress: jobs.JobProgress{
			Percentage: 60,
		},
		Context: jobs.JobContext{
			SiteTitle:        "Test Site",
			CurrentListTitle: "Documents",
			CurrentItemName:  "Important.docx",
		},
		Stats: jobs.JobStats{
			ListsFound:          10,
			ListsProcessed:      6,
			ItemsFound:          100,
			ItemsProcessed:      60,
			PermissionsAnalyzed: 25,
			SharingLinksFound:   5,
			ErrorsEncountered:   1,
		},
		Timeline: []jobs.JobStageInfo{
			{
				Stage:   "initializing",
				Started: time.Now().Add(-10 * time.Minute),
				Completed: func() *time.Time {
					t := time.Now().Add(-8 * time.Minute)
					return &t
				}(),
				Duration: "2m0s",
			},
			{
				Stage:   "processing_lists",
				Started: time.Now().Add(-8 * time.Minute),
			},
		},
		Messages: []string{
			"Started processing lists",
			"Found 10 lists to process",
			"Processing Documents library",
		},
	}

	// Act
	result := presenter.FormatJobStatus(job)

	// Assert - Test rich state presentation
	require.NotNil(t, result)

	// Basic fields
	assert.Equal(t, "job-with-state", result.ID)
	assert.Equal(t, 60, result.Percentage)
	assert.Equal(t, "processing_lists", result.Stage)
	assert.Equal(t, "Processing SharePoint lists", result.Description)

	// Rich fields from state
	assert.Equal(t, "Important.docx", result.CurrentItem)
	assert.Equal(t, "Documents", result.CurrentList)
	assert.Equal(t, "Test Site", result.SiteTitle)
	assert.Len(t, result.RecentMessages, 3)

	// Timeline
	require.Len(t, result.Timeline, 2)
	assert.Equal(t, "initializing", result.Timeline[0].Stage)
	assert.NotEmpty(t, result.Timeline[0].Completed)
	assert.Equal(t, "2m0s", result.Timeline[0].Duration)
	assert.Equal(t, "processing_lists", result.Timeline[1].Stage)
	assert.Empty(t, result.Timeline[1].Completed) // Still running

	// Stats
	assert.Equal(t, 10, result.Stats.ListsFound)
	assert.Equal(t, 6, result.Stats.ListsProcessed)
	assert.Equal(t, 100, result.Stats.ItemsFound)
	assert.Equal(t, 60, result.Stats.ItemsProcessed)
	assert.Equal(t, 25, result.Stats.PermissionsAnalyzed)
	assert.Equal(t, 5, result.Stats.SharingLinksFound)
	assert.Equal(t, 1, result.Stats.ErrorsEncountered)

	// Duration calculation for active job
	assert.NotEmpty(t, result.StageDuration)
}

func TestJobPresenter_FormatJobList_MultipleJobs(t *testing.T) {
	// Arrange
	presenter := NewJobPresenter()
	jobList := []*jobs.Job{
		createTestJob("job-1", jobs.JobTypeSiteAudit, jobs.JobStatusRunning),
		createTestJob("job-2", jobs.JobTypeSiteAudit, jobs.JobStatusCompleted),
		createTestJob("job-3", jobs.JobTypeSiteAudit, jobs.JobStatusFailed),
	}

	// Act
	result := presenter.FormatJobList(jobList)

	// Assert - Test list formatting outcomes
	require.NotNil(t, result)
	require.Len(t, result.Jobs, 3)

	// Should contain all job IDs
	assert.Equal(t, "job-1", result.Jobs[0].ID)
	assert.Equal(t, "job-2", result.Jobs[1].ID)
	assert.Equal(t, "job-3", result.Jobs[2].ID)

	// Should contain formatted statuses
	assert.Equal(t, string(jobs.JobStatusRunning), result.Jobs[0].Status)
	assert.Equal(t, string(jobs.JobStatusCompleted), result.Jobs[1].Status)
	assert.Equal(t, string(jobs.JobStatusFailed), result.Jobs[2].Status)

	// Should contain types
	assert.Equal(t, string(jobs.JobTypeSiteAudit), result.Jobs[0].Type)
	assert.Equal(t, string(jobs.JobTypeSiteAudit), result.Jobs[1].Type)
	assert.Equal(t, string(jobs.JobTypeSiteAudit), result.Jobs[2].Type)
}

func TestJobPresenter_FormatJobList_EmptyList(t *testing.T) {
	// Arrange
	presenter := NewJobPresenter()
	emptyJobs := []*jobs.Job{}

	// Act
	result := presenter.FormatJobList(emptyJobs)

	// Assert
	require.NotNil(t, result)
	assert.Empty(t, result.Jobs)
}

func TestJobPresenter_FormatJobListHTML_MultipleJobs(t *testing.T) {
	// Arrange
	presenter := NewJobPresenter()
	jobList := []*jobs.Job{
		createTestJob("job-1", jobs.JobTypeSiteAudit, jobs.JobStatusRunning),
		createTestJob("job-2", jobs.JobTypeSiteAudit, jobs.JobStatusCompleted),
		createTestJob("job-3", jobs.JobTypeSiteAudit, jobs.JobStatusFailed),
	}

	// Act
	html := presenter.FormatJobListHTML(jobList, true) // includeCancelButton = true

	// Assert - Test HTML generation outcomes
	assert.NotEmpty(t, html)

	// Should contain all job IDs
	assert.Contains(t, html, "job-1")
	assert.Contains(t, html, "job-2")
	assert.Contains(t, html, "job-3")

	// Should contain job types
	assert.Contains(t, html, "Site Audit")

	// Should contain statuses with appropriate styling
	assert.Contains(t, html, "🔄") // Running icon
	assert.Contains(t, html, "✅") // Completed icon
	assert.Contains(t, html, "❌") // Failed icon

	// Should contain cancel button for running jobs
	assert.Contains(t, html, "Cancel")               // Cancel button should be present
	assert.Contains(t, html, "jobs/job-1/cancel")    // Cancel URL for running job
	assert.NotContains(t, html, "jobs/job-2/cancel") // No cancel for completed job
}

func TestJobPresenter_FormatJobListHTML_EmptyList(t *testing.T) {
	// Arrange
	presenter := NewJobPresenter()
	emptyJobs := []*jobs.Job{}

	// Act
	html := presenter.FormatJobListHTML(emptyJobs, true)

	// Assert
	assert.NotEmpty(t, html)
	assert.Contains(t, html, "No jobs yet")    // Should show empty state message
	assert.Contains(t, html, "Start an audit") // Should show helpful message
}

func TestJobPresenter_FormatJobNotFound(t *testing.T) {
	// Arrange
	presenter := NewJobPresenter()

	// Act
	result := presenter.FormatJobNotFound()

	// Assert
	require.NotNil(t, result)
	assert.Empty(t, result.ID)
	assert.Equal(t, "not_found", result.Status)
	assert.Equal(t, "Job not found", result.Error)
}

// Test nil safety and error handling
func TestJobPresenter_NilSafety(t *testing.T) {
	presenter := NewJobPresenter()

	// Should not panic with nil job
	result := presenter.FormatJobStatus(nil)
	assert.Nil(t, result)

	// Should handle minimal state gracefully (new jobs have default initialized state)
	jobWithMinimalState := createTestJob("minimal-state", jobs.JobTypeSiteAudit, jobs.JobStatusRunning)

	result2 := presenter.FormatJobStatus(jobWithMinimalState)
	require.NotNil(t, result2)
	assert.Equal(t, 0, result2.Percentage)         // Default percentage from initialized state
	assert.Equal(t, "initializing", result2.Stage) // Default stage from initialized state
}

// Test that presenter doesn't modify input data
func TestJobPresenter_DoesNotModifyInput(t *testing.T) {
	presenter := NewJobPresenter()

	originalJob := createTestJobWithProgress("original", 50, "original", "Original description")
	originalPercentage := originalJob.State.Progress.Percentage
	originalDescription := originalJob.State.Progress.Description

	// Format the job
	presenter.FormatJobStatus(originalJob)

	// Verify original job unchanged
	assert.Equal(t, originalPercentage, originalJob.State.Progress.Percentage)
	assert.Equal(t, originalDescription, originalJob.State.Progress.Description)
}

func TestJobPresenter_MessageFormatting(t *testing.T) {
	// Test various message formatting functions
	presenter := NewJobPresenter()

	// Test cancel messages
	successMsg := presenter.FormatCancelSuccessMessage()
	assert.Contains(t, successMsg, "cancelled successfully")
	assert.Contains(t, successMsg, "text-green-600")

	errorMsg := presenter.FormatCancelErrorMessage(fmt.Errorf("test error"))
	assert.Contains(t, errorMsg, "test error")
	assert.Contains(t, errorMsg, "text-red-600")

	notActiveMsg := presenter.FormatJobNotActiveMessage()
	assert.Contains(t, notActiveMsg, "no longer active")
	assert.Contains(t, notActiveMsg, "text-orange-600")

	// Test audit queue messages
	queuedMsg := presenter.FormatAuditQueuedSuccessMessage()
	assert.Contains(t, queuedMsg, "queued successfully")
	assert.Contains(t, queuedMsg, "text-green-600")

	queueErrorMsg := presenter.FormatAuditQueuedErrorMessage(fmt.Errorf("queue error"))
	assert.Contains(t, queueErrorMsg, "queue error")
	assert.Contains(t, queueErrorMsg, "text-red-600")

	alreadyRunningMsg := presenter.FormatAuditAlreadyRunningMessage()
	assert.Contains(t, alreadyRunningMsg, "already running")
	assert.Contains(t, alreadyRunningMsg, "text-orange-600")
}

// Test edge cases and robustness improvements

func TestJobPresenter_EdgeCases_HTMLGeneration(t *testing.T) {
	presenter := NewJobPresenter()

	// Test HTML generation with edge case data
	jobs := []*jobs.Job{
		// Job with minimal data
		createTestJob("minimal", jobs.JobTypeSiteAudit, jobs.JobStatusPending),
		// Job with complete rich state
		func() *jobs.Job {
			job := createTestJob("complete", jobs.JobTypeSiteAudit, jobs.JobStatusRunning)
			job.State = jobs.JobState{
				Stage:            "finalizing",
				CurrentOperation: "Completing audit",
				Progress:         jobs.JobProgress{Percentage: 100},
				Context: jobs.JobContext{
					SiteTitle:        "Rich Site",
					CurrentListTitle: "Active List",
					CurrentItemName:  "Current Item.xlsx",
				},
				Stats: jobs.JobStats{
					ListsFound:        10,
					ItemsFound:        100,
					ErrorsEncountered: 3,
				},
			}
			return job
		}(),
	}

	// Test partial update HTML
	htmlPartial := presenter.FormatJobListHTML(jobs, true)
	assert.NotEmpty(t, htmlPartial)
	assert.NotContains(t, htmlPartial, "hx-ext=\"sse\"") // Partial shouldn't have SSE wrapper

	// Test full HTML with SSE wrapper
	htmlFull := presenter.FormatJobListHTML(jobs, false)
	assert.NotEmpty(t, htmlFull)
	assert.Contains(t, htmlFull, "hx-ext=\"sse\"") // Full should have SSE wrapper
	assert.Contains(t, htmlFull, "minimal")        // Should contain job content

	// Verify HTML contains expected job information
	assert.Contains(t, htmlPartial, "minimal")    // Job ID
	assert.Contains(t, htmlPartial, "complete")   // Job ID
	assert.Contains(t, htmlPartial, "Site Audit") // Job type display
}
