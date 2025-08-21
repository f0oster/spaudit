package jobs

import (
	"fmt"
	"time"
)

// JobLifecycle manages job state transitions and business rules
type JobLifecycle struct{}

// StartJob transitions job to running status with validation
func (jl *JobLifecycle) StartJob(job *Job) error {
	if job.Status != JobStatusPending {
		return fmt.Errorf("cannot start job in status: %s", job.Status)
	}

	job.Status = JobStatusRunning
	job.StartedAt = time.Now()
	job.InitializeState()
	return nil
}

// CompleteJob transitions job to completed status with state finalization
func (jl *JobLifecycle) CompleteJob(job *Job) error {
	if !job.IsActive() {
		return fmt.Errorf("cannot complete inactive job")
	}

	job.Status = JobStatusCompleted
	now := time.Now()
	job.CompletedAt = &now

	jl.finalizeJobState(job, "completed", "Audit completed successfully")
	return nil
}

// FailJob transitions job to failed status with error details
func (jl *JobLifecycle) FailJob(job *Job, errorMsg string) error {
	if !job.IsActive() {
		return fmt.Errorf("cannot fail inactive job")
	}

	job.Status = JobStatusFailed
	job.Error = errorMsg
	now := time.Now()
	job.CompletedAt = &now

	jl.finalizeJobState(job, "failed", fmt.Sprintf("Audit failed: %s", errorMsg))
	return nil
}

// CancelJob transitions job to cancelled status
func (jl *JobLifecycle) CancelJob(job *Job) error {
	if !job.IsActive() {
		return fmt.Errorf("cannot cancel inactive job")
	}

	job.Status = JobStatusCancelled
	now := time.Now()
	job.CompletedAt = &now

	jl.finalizeJobState(job, "cancelled", "Audit cancelled")
	return nil
}

// finalizeJobState handles consistent state finalization for completed jobs
func (jl *JobLifecycle) finalizeJobState(job *Job, stage, operation string) {
	// State is always initialized

	now := time.Now()

	// Complete current stage in timeline
	if len(job.State.Timeline) > 0 {
		lastStage := &job.State.Timeline[len(job.State.Timeline)-1]
		if lastStage.Completed == nil {
			lastStage.Completed = &now
			lastStage.Duration = now.Sub(lastStage.Started).String()
		}
	}

	// Update final state
	job.State.Stage = stage
	job.State.CurrentOperation = operation
	job.State.StageStartedAt = now

	if stage == "completed" {
		job.State.Progress.Percentage = 100
	}

	// Add final message to rolling buffer
	job.State.Messages = append(job.State.Messages, fmt.Sprintf("[%s] %s", stage, operation))
	if len(job.State.Messages) > 10 {
		job.State.Messages = job.State.Messages[1:]
	}
}
