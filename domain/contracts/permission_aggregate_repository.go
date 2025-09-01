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

// PermissionAggregateRepository handles permission analysis operations that span assignments, items, and sharing.
// This aggregate repository encapsulates complex permission analytics and risk assessment business logic.
type PermissionAggregateRepository interface {
	// Get raw components for permission analysis - service will do the business logic (audit-scoped)
	GetPermissionAnalysisComponents(ctx context.Context, siteID int64, auditRunID int64, list *sharepoint.List) (*PermissionAnalysisComponents, error)
}
