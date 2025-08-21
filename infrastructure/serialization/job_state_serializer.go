package serialization

import (
	"encoding/json"
	"fmt"

	"spaudit/domain/jobs"
)

// JobStateSerializer handles JSON serialization/deserialization of job state.
type JobStateSerializer struct{}

// NewJobStateSerializer creates a new job state serializer.
func NewJobStateSerializer() *JobStateSerializer {
	return &JobStateSerializer{}
}

// SerializeState converts JobState to JSON string.
func (s *JobStateSerializer) SerializeState(state jobs.JobState) (string, error) {
	data, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job state: %w", err)
	}
	return string(data), nil
}

// DeserializeState converts JSON string to JobState.
func (s *JobStateSerializer) DeserializeState(jsonStr string) (jobs.JobState, error) {
	if jsonStr == "" {
		// Return default initialized state
		return jobs.JobState{
			Stage:            "unknown",
			CurrentOperation: "No state available",
			Progress: jobs.JobProgress{
				Stage:       "unknown",
				Description: "No progress available",
				Percentage:  0,
			},
			Timeline: []jobs.JobStageInfo{},
			Stats:    jobs.JobStats{},
			Messages: []string{},
		}, nil
	}

	var state jobs.JobState
	if err := json.Unmarshal([]byte(jsonStr), &state); err != nil {
		return jobs.JobState{}, fmt.Errorf("failed to unmarshal job state: %w", err)
	}

	return state, nil
}

// SerializeContextData converts JobContextData to JSON string for storage.
func (s *JobStateSerializer) SerializeContextData(context jobs.JobContextData) (string, error) {
	if context == nil {
		return "", nil
	}

	data, err := json.Marshal(context)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job context: %w", err)
	}
	return string(data), nil
}

// DeserializeAuditContext converts JSON string to AuditJobContext.
func (s *JobStateSerializer) DeserializeAuditContext(jsonStr string) (jobs.AuditJobContext, error) {
	if jsonStr == "" {
		return jobs.AuditJobContext{}, nil
	}

	var context jobs.AuditJobContext
	if err := json.Unmarshal([]byte(jsonStr), &context); err != nil {
		return jobs.AuditJobContext{}, fmt.Errorf("failed to unmarshal audit context: %w", err)
	}

	return context, nil
}
