package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"spaudit/database"
	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/gen/db"
)

// SqlcAssignmentRepository implements contracts.AssignmentRepository using sqlc queries with read/write separation
type SqlcAssignmentRepository struct {
	*BaseRepository
}

// NewSqlcAssignmentRepository creates a new assignment repository with read/write database separation
func NewSqlcAssignmentRepository(database *database.Database) contracts.AssignmentRepository {
	return &SqlcAssignmentRepository{
		BaseRepository: NewBaseRepository(database),
	}
}

// GetAssignmentsForObject retrieves role assignments for an object
func (r *SqlcAssignmentRepository) GetAssignmentsForObject(ctx context.Context, siteID int64, objectType, objectKey string) ([]*sharepoint.Assignment, error) {
	rows, err := r.ReadQueries().GetAssignmentsForObject(ctx, db.GetAssignmentsForObjectParams{
		SiteID:     siteID,
		ObjectType: objectType,
		ObjectKey:  objectKey,
	})
	if err != nil {
		return nil, err
	}

	// Transform SQLC rows to domain assignments with complete construction
	assignments := make([]*sharepoint.Assignment, len(rows))
	for i, row := range rows {
		// Construct complete Principal with all required fields
		principal := &sharepoint.Principal{
			SiteID:        siteID, // ✅ Complete construction with required context
			ID:            row.PrincipalID,
			PrincipalType: row.PrincipalType,
			Title:         r.FromNullString(row.PrincipalTitle),
			LoginName:     r.FromNullString(row.LoginName),
			Email:         "", // Not available in this query
		}

		// Construct complete RoleDefinition with all required fields
		roleDefinition := &sharepoint.RoleDefinition{
			SiteID:      siteID, // ✅ Complete construction
			ID:          row.RoleDefID,
			Name:        row.RoleName,
			Description: r.FromNullString(row.Description),
		}

		// Construct complete RoleAssignment with all required fields
		roleAssignment := &sharepoint.RoleAssignment{
			SiteID:      siteID,     // ✅ Complete construction
			ObjectType:  objectType, // ✅ Provided by caller
			ObjectKey:   objectKey,  // ✅ Provided by caller
			PrincipalID: row.PrincipalID,
			RoleDefID:   row.RoleDefID,
			Inherited:   r.FromNullBool(row.Inherited),
		}

		assignments[i] = &sharepoint.Assignment{
			RoleAssignment: roleAssignment,
			Principal:      principal,
			RoleDefinition: roleDefinition,
		}
	}

	return assignments, nil
}

// GetResolvedAssignmentsForObject retrieves role assignments with root cause analysis
func (r *SqlcAssignmentRepository) GetResolvedAssignmentsForObject(ctx context.Context, siteID int64, objectType, objectKey string) ([]*sharepoint.ResolvedAssignment, error) {
	// First get the regular assignments
	assignments, err := r.GetAssignmentsForObject(ctx, siteID, objectType, objectKey)
	if err != nil {
		return nil, err
	}

	// Get the parent web ID for inheritance analysis
	webId, err := r.ReadQueries().GetWebIdForObject(ctx, db.GetWebIdForObjectParams{
		SiteID:     siteID,
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
		rootCauseAnalysis := r.analyzeRootCause(ctx, siteID, assignment, webIdStr)
		resolved = append(resolved, &sharepoint.ResolvedAssignment{
			Assignment: assignment,
			RootCauses: rootCauseAnalysis.RootCauses,
		})
	}

	return resolved, nil
}

type rootCauseAnalysis struct {
	RootCauses []sharepoint.RootCause // All detected sources
}

func (r *SqlcAssignmentRepository) analyzeRootCause(ctx context.Context, siteID int64, assignment *sharepoint.Assignment, webId string) rootCauseAnalysis {
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
	rootPerms, err := r.ReadQueries().GetRootPermissionsForPrincipalInWeb(ctx, db.GetRootPermissionsForPrincipalInWebParams{
		SiteID:      siteID,
		PrincipalID: assignment.Principal.ID,
		WebID:       webId,
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

	return rootCauseAnalysis{
		RootCauses: rootCauses,
	}
}

func (r *SqlcAssignmentRepository) analyzeSharingLinkRootCause(ctx context.Context, assignment *sharepoint.Assignment, loginName string) sharepoint.RootCause {
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
	sharedItem, err := r.ReadQueries().GetSharedItemForSharingLink(ctx, db.GetSharedItemForSharingLinkParams{
		SiteID: assignment.RoleAssignment.SiteID,
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
