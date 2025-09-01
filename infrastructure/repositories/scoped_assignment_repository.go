package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/gen/db"
)

// ScopedAssignmentRepository wraps an AssignmentRepository with automatic site and audit run scoping
type ScopedAssignmentRepository struct {
	*BaseRepository
	queries     *db.Queries
	siteID      int64
	auditRunID  int64
}

// NewScopedAssignmentRepository creates a new scoped assignment repository
func NewScopedAssignmentRepository(baseRepo *BaseRepository, queries *db.Queries, siteID, auditRunID int64) contracts.AssignmentRepository {
	return &ScopedAssignmentRepository{
		BaseRepository: baseRepo,
		queries:        queries,
		siteID:         siteID,
		auditRunID:     auditRunID,
	}
}

// GetAssignmentsForObject retrieves role assignments for an object scoped to audit run
func (r *ScopedAssignmentRepository) GetAssignmentsForObject(ctx context.Context, siteID int64, objectType, objectKey string) ([]*sharepoint.Assignment, error) {
	// Verify the requested siteID matches our scoped siteID
	if siteID != r.siteID {
		return nil, contracts.ErrSiteScopeMismatch
	}

	// Get assignments from database scoped to our audit run
	rows, err := r.queries.GetAssignmentsForObjectByAuditRun(ctx, db.GetAssignmentsForObjectByAuditRunParams{
		SiteID:     r.siteID,
		ObjectType: objectType,
		ObjectKey:  objectKey,
		AuditRunID: r.auditRunID,
	})
	if err != nil {
		return nil, err
	}

	// Convert database rows to domain objects
	var assignments []*sharepoint.Assignment
	for _, row := range rows {
		
		// Construct complete Principal with all required fields
		principal := &sharepoint.Principal{
			SiteID:        r.siteID,
			ID:            row.PrincipalID,
			PrincipalType: row.PrincipalType,
			Title:         r.FromNullString(row.PrincipalTitle),
			LoginName:     r.FromNullString(row.LoginName),
			Email:         "",
		}

		// Construct complete RoleDefinition with all required fields
		roleDefinition := &sharepoint.RoleDefinition{
			SiteID:      r.siteID,
			ID:          row.RoleDefID,
			Name:        row.RoleName,
			Description: r.FromNullString(row.Description),
		}

		// Construct complete RoleAssignment with all required fields
		roleAssignment := &sharepoint.RoleAssignment{
			SiteID:      r.siteID,
			ObjectType:  objectType,
			ObjectKey:   objectKey,
			PrincipalID: row.PrincipalID,
			RoleDefID:   row.RoleDefID,
			Inherited:   r.FromNullBool(row.Inherited),
		}

		assignment := &sharepoint.Assignment{
			RoleAssignment: roleAssignment,
			Principal:      principal,
			RoleDefinition: roleDefinition,
		}
		
		assignments = append(assignments, assignment)
	}

	return assignments, nil
}

// GetResolvedAssignmentsForObject retrieves role assignments with root cause analysis scoped to audit run
func (r *ScopedAssignmentRepository) GetResolvedAssignmentsForObject(ctx context.Context, siteID int64, objectType, objectKey string) ([]*sharepoint.ResolvedAssignment, error) {
	// Verify the requested siteID matches our scoped siteID
	if siteID != r.siteID {
		return nil, contracts.ErrSiteScopeMismatch
	}

	// First get the regular assignments
	assignments, err := r.GetAssignmentsForObject(ctx, siteID, objectType, objectKey)
	if err != nil {
		return nil, err
	}

	// Get the parent web ID for inheritance analysis
	webId, err := r.queries.GetWebIdForObject(ctx, db.GetWebIdForObjectParams{
		SiteID:     r.siteID,
		ObjectType: objectType,
		ObjectKey:  objectKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get web ID: %w", err)
	}

	webIdStr, ok := webId.(string)
	if !ok {
		return nil, fmt.Errorf("web ID is not a string")
	}

	resolved := make([]*sharepoint.ResolvedAssignment, 0, len(assignments))

	for _, assignment := range assignments {
		rootCauseAnalysis := r.analyzeRootCause(ctx, assignment, webIdStr)
		resolved = append(resolved, &sharepoint.ResolvedAssignment{
			Assignment: assignment,
			RootCauses: rootCauseAnalysis.RootCauses,
		})
	}

	return resolved, nil
}

type scopedRootCauseAnalysis struct {
	RootCauses []sharepoint.RootCause // All detected sources
}

func (r *ScopedAssignmentRepository) analyzeRootCause(ctx context.Context, assignment *sharepoint.Assignment, webId string) scopedRootCauseAnalysis {
	loginName := assignment.Principal.LoginName
	var rootCauses []sharepoint.RootCause

	// Check if it's a sharing link
	if strings.HasPrefix(loginName, "SharingLinks.") {
		sharingLinkCause := r.analyzeSharingLinkRootCause(ctx, assignment, loginName)
		rootCauses = append(rootCauses, sharingLinkCause)
	}

	// Check if it's a system group
	if strings.Contains(loginName, "Limited Access System Group") {
		systemGroupCause := sharepoint.RootCause{
			Type:         sharepoint.RootCauseTypeSystemGroup,
			Detail:       "System-generated group for SharePoint navigation",
			SourceObject: "SharePoint System",
			SourceRole:   "System Role",
		}
		rootCauses = append(rootCauses, systemGroupCause)
	}

	// Check for same-web inheritance - get ALL permissions, not just first
	rootPerms, err := r.queries.GetRootPermissionsForPrincipalInWebByAuditRun(ctx, db.GetRootPermissionsForPrincipalInWebByAuditRunParams{
		SiteID:      r.siteID,
		PrincipalID: assignment.Principal.ID,
		WebID:       webId,
		AuditRunID:  r.auditRunID,
	})
	if err == nil && len(rootPerms) > 0 {
		// Process ALL root permissions, not just the first one
		for _, rootPerm := range rootPerms {
			objectName := "Unknown"
			if rootPerm.ObjectName != nil {
				if name, ok := rootPerm.ObjectName.(string); ok {
					objectName = name
				}
			}
			inheritanceCause := sharepoint.RootCause{
				Type:         sharepoint.RootCauseTypeInheritance,
				Detail:       fmt.Sprintf("Auto-granted for navigation to %s (%s)", objectName, rootPerm.RoleName),
				SourceObject: objectName,
				SourceRole:   rootPerm.RoleName,
			}
			rootCauses = append(rootCauses, inheritanceCause)
		}
	}

	// If no root causes found, add unknown
	if len(rootCauses) == 0 {
		unknownCause := sharepoint.RootCause{
			Type:         sharepoint.RootCauseTypeUnknown,
			Detail:       "Unable to determine root cause",
			SourceObject: "Unknown",
			SourceRole:   "",
		}
		rootCauses = append(rootCauses, unknownCause)
	}

	return scopedRootCauseAnalysis{
		RootCauses: rootCauses,
	}
}

func (r *ScopedAssignmentRepository) analyzeSharingLinkRootCause(ctx context.Context, assignment *sharepoint.Assignment, loginName string) sharepoint.RootCause {
	// Extract file/folder GUID from sharing link login name
	// Format: SharingLinks.{GUID}.{Type}.{ShareId}
	parts := strings.Split(loginName, ".")
	if len(parts) < 2 {
		return sharepoint.RootCause{
			Type:         sharepoint.RootCauseTypeSharingLink,
			Detail:       "SharePoint sharing link navigation permission",
			SourceObject: "Shared item",
			SourceRole:   "File Share",
		}
	}

	fileFolderGuid := parts[1]
	sharedItem, err := r.queries.GetSharedItemForSharingLink(ctx, db.GetSharedItemForSharingLinkParams{
		SiteID: r.siteID,
		FileFolderGuid: sql.NullString{
			String: fileFolderGuid,
			Valid:  true,
		},
	})
	if err != nil {
		return sharepoint.RootCause{
			Type:         sharepoint.RootCauseTypeSharingLink,
			Detail:       "SharePoint sharing link navigation permission",
			SourceObject: "Shared item",
			SourceRole:   "File Share",
		}
	}

	return sharepoint.RootCause{
		Type:         sharepoint.RootCauseTypeSharingLink,
		Detail:       fmt.Sprintf("Shared file: %s in %s", sharedItem.ItemName.String, sharedItem.ListTitle.String),
		SourceObject: sharedItem.ItemName.String,
		SourceRole:   "File Share",
	}
}

