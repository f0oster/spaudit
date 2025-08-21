package contracts

import (
	"context"

	"spaudit/domain/sharepoint"
)

// AssignmentRepository defines operations for Assignment entities.
type AssignmentRepository interface {
	// GetAssignmentsForObject retrieves role assignments for an object.
	GetAssignmentsForObject(ctx context.Context, siteID int64, objectType, objectKey string) ([]*sharepoint.Assignment, error)
	// GetResolvedAssignmentsForObject retrieves role assignments with root cause analysis.
	GetResolvedAssignmentsForObject(ctx context.Context, siteID int64, objectType, objectKey string) ([]*sharepoint.ResolvedAssignment, error)
}
