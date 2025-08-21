package application

import (
	"fmt"
	"sync"

	"spaudit/domain/jobs"
)

// JobExecutorRegistry manages a registry of job executors for different job types.
// This enables pluggable job execution while maintaining type safety and clean separation.
type JobExecutorRegistry struct {
	executors map[jobs.JobType]JobExecutor
	mutex     sync.RWMutex
}

// NewJobExecutorRegistry creates a new job executor registry.
func NewJobExecutorRegistry() *JobExecutorRegistry {
	return &JobExecutorRegistry{
		executors: make(map[jobs.JobType]JobExecutor),
	}
}

// RegisterExecutor registers an executor for a specific job type.
// This allows different parts of the system to provide execution logic for their job types.
func (r *JobExecutorRegistry) RegisterExecutor(jobType jobs.JobType, executor JobExecutor) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.executors[jobType] = executor
}

// GetExecutor retrieves the executor for a specific job type.
// Returns an error if no executor is registered for the given type.
func (r *JobExecutorRegistry) GetExecutor(jobType jobs.JobType) (JobExecutor, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	executor, exists := r.executors[jobType]
	if !exists {
		return nil, fmt.Errorf("no executor registered for job type: %s", jobType)
	}

	return executor, nil
}

// GetSupportedJobTypes returns all job types that have registered executors.
func (r *JobExecutorRegistry) GetSupportedJobTypes() []jobs.JobType {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	types := make([]jobs.JobType, 0, len(r.executors))
	for jobType := range r.executors {
		types = append(types, jobType)
	}

	return types
}

// IsSupported checks if a job type has a registered executor.
func (r *JobExecutorRegistry) IsSupported(jobType jobs.JobType) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.executors[jobType]
	return exists
}
