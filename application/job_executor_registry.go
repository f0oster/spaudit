package application

import (
	"fmt"
	"sync"

	"spaudit/domain/jobs"
)

// JobExecutorRegistry manages job executors for different job types.
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

// RegisterExecutor registers an executor for a job type.
func (r *JobExecutorRegistry) RegisterExecutor(jobType jobs.JobType, executor JobExecutor) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.executors[jobType] = executor
}

// GetExecutor retrieves the executor for a job type.
func (r *JobExecutorRegistry) GetExecutor(jobType jobs.JobType) (JobExecutor, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	executor, exists := r.executors[jobType]
	if !exists {
		return nil, fmt.Errorf("no executor registered for job type: %s", jobType)
	}

	return executor, nil
}

