package contracts

import (
	"context"
	"spaudit/domain/sharepoint"
)

// PermissionAnalysisComponents represents the raw components needed for permission analysis.
type PermissionAnalysisComponents struct {
	Assignments  []*sharepoint.Assignment
	Items        []*sharepoint.Item
	SharingLinks []*sharepoint.SharingLink
	List         *sharepoint.List
}

// PermissionAggregateRepository handles permission analysis across assignments, items, and sharing.
type PermissionAggregateRepository interface {
	// Get raw components for permission analysis (audit-scoped)
	GetPermissionAnalysisComponents(ctx context.Context, siteID int64, auditRunID int64, list *sharepoint.List) (*PermissionAnalysisComponents, error)
}
