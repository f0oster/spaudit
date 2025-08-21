package jobs

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// JobFactory creates new jobs with proper initialization
type JobFactory struct{}

// CreateJob creates a new job with initialized progress and state
func (jf *JobFactory) CreateJob(jobType JobType, siteURL, description string) *Job {
	jobID := jf.generateJobID(jobType, siteURL)

	job := &Job{
		ID:        jobID,
		Type:      jobType,
		Status:    JobStatusPending,
		StartedAt: time.Now(),
		Context:   AuditJobContext{SiteURL: siteURL},
	}

	// Initialize progress tracking and rich state
	job.UpdateProgress("initializing", "Preparing job...", 0, 0, 0)
	job.InitializeState()

	return job
}

// generateJobID creates a unique job identifier
func (jf *JobFactory) generateJobID(jobType JobType, siteURL string) string {
	// Generate random component for uniqueness
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-only if random fails
		return fmt.Sprintf("%s_%s", jobType, time.Now().Format("20060102_150405"))
	}

	return fmt.Sprintf("%s_%s_%s",
		jobType,
		time.Now().Format("20060102_150405"),
		hex.EncodeToString(bytes))
}
