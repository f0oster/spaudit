package spauditor

import (
	"context"
	"fmt"

	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/infrastructure/spclient"
	"spaudit/logging"
)

// PermissionCollector handles collection and persistence of role assignments
type PermissionCollector struct {
	spClient spclient.SharePointClient
	repo     contracts.SharePointAuditRepository
	logger   *logging.Logger
}

// NewPermissionCollector creates a new permission collector
func NewPermissionCollector(spClient spclient.SharePointClient, repo contracts.SharePointAuditRepository) *PermissionCollector {
	return &PermissionCollector{
		spClient: spClient,
		repo:     repo,
		logger:   logging.Default().WithComponent("permission_collector"),
	}
}

// CollectRoleDefinitions retrieves and persists role definitions for the web
func (pc *PermissionCollector) CollectRoleDefinitions(ctx context.Context, auditRunID int64, siteID int64) error {
	roleDefs, err := pc.spClient.GetSiteRoleDefinitions(ctx)
	if err != nil {
		return fmt.Errorf("get role definitions: %w", err)
	}

	// Set site ID for all role definitions
	for _, roleDef := range roleDefs {
		roleDef.SiteID = siteID
	}

	if err := pc.repo.SaveRoleDefinitions(ctx, roleDefs); err != nil {
		return fmt.Errorf("save role definitions: %w", err)
	}

	return nil
}

// CollectWebRoleAssignments retrieves and persists role assignments for a web
func (pc *PermissionCollector) CollectWebRoleAssignments(ctx context.Context, auditRunID int64, siteID int64, webID string) error {
	target := spclient.PermissionTarget{
		ObjectType: sharepoint.ObjectTypeWeb,
		ObjectID:   webID,
	}

	return pc.collectRoleAssignments(ctx, auditRunID, siteID, target)
}

// CollectListRoleAssignments retrieves and persists role assignments for a list
func (pc *PermissionCollector) CollectListRoleAssignments(ctx context.Context, auditRunID int64, siteID int64, listID string) error {
	target := spclient.PermissionTarget{
		ObjectType: sharepoint.ObjectTypeList,
		ObjectID:   listID,
	}

	return pc.collectRoleAssignments(ctx, auditRunID, siteID, target)
}

// CollectItemRoleAssignments retrieves and persists role assignments for an item
func (pc *PermissionCollector) CollectItemRoleAssignments(ctx context.Context, auditRunID int64, siteID int64, listID, itemGUID string, itemID int) error {
	target := spclient.PermissionTarget{
		ObjectType: sharepoint.ObjectTypeItem,
		ObjectID:   listID,
		ListItemID: itemID,
	}

	return pc.collectRoleAssignmentsWithKey(ctx, auditRunID, siteID, target, itemGUID)
}

// collectRoleAssignments is the common implementation for collecting role assignments
func (pc *PermissionCollector) collectRoleAssignments(ctx context.Context, auditRunID int64, siteID int64, target spclient.PermissionTarget) error {
	return pc.collectRoleAssignmentsWithKey(ctx, auditRunID, siteID, target, target.ObjectID)
}

// collectRoleAssignmentsWithKey allows specifying a different object key (used for items)
func (pc *PermissionCollector) collectRoleAssignmentsWithKey(ctx context.Context, auditRunID int64, siteID int64, target spclient.PermissionTarget, objectKey string) error {
	// Note: We don't clear existing assignments since each audit run should preserve historical data
	// Each audit run gets a unique audit_run_id, so new assignments won't conflict with old ones

	// Get new assignments and principals
	assignments, principals, err := pc.spClient.GetObjectRoleAssignments(ctx, target)
	if err != nil {
		return fmt.Errorf("get role assignments: %w", err)
	}

	// Save principals (set site ID for each)
	for _, principal := range principals {
		principal.SiteID = siteID
		if err := pc.repo.SavePrincipal(ctx, principal); err != nil {
			// Check if context was canceled - if so, return the error to stop processing
			if ctx.Err() != nil {
				return fmt.Errorf("context canceled while saving principal %d: %w", principal.ID, ctx.Err())
			}
			pc.logger.Error("Failed to save principal during permission collection",
				"principal_id", principal.ID,
				"title", principal.Title,
				"login_name", principal.LoginName,
				"email", principal.Email,
				"type", principal.PrincipalType,
				"error", err.Error())
		}
	}

	// Update assignments with site ID and object keys for items (assignments come back with listID, but we want itemGUID)
	for _, assignment := range assignments {
		assignment.SiteID = siteID
		if target.ObjectType == sharepoint.ObjectTypeItem {
			assignment.ObjectKey = objectKey
		}
	}

	// Save assignments
	if err := pc.repo.SaveRoleAssignments(ctx, assignments); err != nil {
		return fmt.Errorf("save role assignments: %w", err)
	}

	return nil
}
