package application

import (
	"context"

	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
)

// PermissionAnalysisData represents comprehensive permission analysis results.
type PermissionAnalysisData struct {
	TotalAssignments       int
	UniqueAssignments      int
	InheritedAssignments   int
	ItemLevelAssignments   int
	UserCount              int
	GroupCount             int
	SharingLinkCount       int
	SharingLinkUsers       int
	FlexibleLinksCount     int
	OrganizationViewCount  int
	OrganizationEditCount  int
	AnonymousViewCount     int
	AnonymousEditCount     int
	DirectLinksCount       int
	OtherLinksCount        int
	TotalItems             int64
	ItemsWithUnique        int64
	FilesCount             int64
	FoldersCount           int64
	FullControlCount       int
	ContributeCount        int
	ReadCount              int
	LimitedAccessCount     int
	OtherRolesCount        int
	PermissionRiskLevel    string
	PermissionRiskScore    float64
	RiskFromUniqueItems    float64
	RiskFromAssignments    float64
	RiskFromSharingLinks   float64
	RiskFromElevatedAccess float64
}

// PermissionService handles permission analysis and risk assessment business logic using aggregate repository.
type PermissionService struct {
	permissionAggregate contracts.PermissionAggregateRepository
	auditRunID          int64 // For audit-scoped operations
}

// NewPermissionService creates a new permission service with aggregate repository dependency injection.
func NewPermissionService(
	permissionAggregate contracts.PermissionAggregateRepository,
) *PermissionService {
	return newPermissionService(permissionAggregate, 0) // 0 means no specific audit run
}

// NewAuditScopedPermissionService creates a permission service scoped to a specific audit run.
func NewAuditScopedPermissionService(
	permissionAggregate contracts.PermissionAggregateRepository,
	auditRunID int64,
) *PermissionService {
	return newPermissionService(permissionAggregate, auditRunID)
}

// newPermissionService is the common constructor.
func newPermissionService(
	permissionAggregate contracts.PermissionAggregateRepository,
	auditRunID int64,
) *PermissionService {
	return &PermissionService{
		permissionAggregate: permissionAggregate,
		auditRunID:          auditRunID,
	}
}

// AnalyzeListPermissions creates comprehensive analytics for a list.
func (s *PermissionService) AnalyzeListPermissions(
	ctx context.Context,
	siteID int64,
	list *sharepoint.List,
) (*PermissionAnalysisData, error) {
	// Get raw components from aggregate repository (audit-scoped)
	components, err := s.permissionAggregate.GetPermissionAnalysisComponents(ctx, siteID, s.auditRunID, list)
	if err != nil {
		return nil, err
	}

	data := &PermissionAnalysisData{
		// List will be populated by presenter
	}

	// Calculate assignment analytics - we only audit unique permissions
	data.TotalAssignments = len(components.Assignments)
	data.UniqueAssignments = len(components.Assignments) // All assignments we audit are unique
	data.InheritedAssignments = 0                        // We don't audit inherited permissions
	data.UserCount, data.GroupCount, _ = s.calculatePrincipalTypes(components.Assignments)
	data.FullControlCount, data.ContributeCount, data.ReadCount,
		data.LimitedAccessCount, data.OtherRolesCount = s.calculateRoleDistribution(components.Assignments)

	// Handle sharing links
	if components.SharingLinks != nil {
		data.SharingLinkCount = len(components.SharingLinks)
		// Calculate total sharing link users by summing actual members count
		// and count different link types
		totalSharingLinkUsers := 0
		for _, link := range components.SharingLinks {
			totalSharingLinkUsers += int(link.TotalMembersCount)

			// Count by link kind name (using string comparison for reliability)
			switch link.GetLinkKindName() {
			case "Flexible":
				data.FlexibleLinksCount++
			case "Organization View":
				data.OrganizationViewCount++
			case "Organization Edit":
				data.OrganizationEditCount++
			case "Anonymous View":
				data.AnonymousViewCount++
			case "Anonymous Edit":
				data.AnonymousEditCount++
			case "Direct":
				data.DirectLinksCount++
			default:
				data.OtherLinksCount++
			}
		}
		data.SharingLinkUsers = totalSharingLinkUsers
	}

	// Always use SharePoint's reported item count for total
	data.TotalItems = int64(list.ItemCount)

	// Handle items analysis
	if components.Items != nil {
		// Use content service for comprehensive analysis
		contentService := sharepoint.NewContentService()
		contentAnalysis := contentService.AnalyzeItems(components.Items)
		data.ItemsWithUnique = contentAnalysis.ItemsWithUnique
		data.FilesCount = contentAnalysis.FilesCount
		data.FoldersCount = contentAnalysis.FoldersCount

		// Calculate item-level assignments count for items with unique permissions
		data.ItemLevelAssignments = 0
		// This would require additional repository calls in the aggregate approach
	}

	// Risk assessment using extracted business logic
	permissionsService := sharepoint.NewPermissionsService()
	riskData := &sharepoint.SharePointRiskData{
		TotalItems:         data.TotalItems,
		ItemsWithUnique:    data.ItemsWithUnique,
		TotalAssignments:   data.TotalAssignments,
		LimitedAccessCount: data.LimitedAccessCount,
		FullControlCount:   data.FullControlCount,
		ContributeCount:    data.ContributeCount,
		SharingLinkCount:   data.SharingLinkCount,
	}

	assessment := permissionsService.CalculateSharePointRiskAssessment(riskData)

	// Map to existing data structure (preserves existing interface)
	data.PermissionRiskScore = assessment.RiskScore
	data.PermissionRiskLevel = assessment.RiskLevel
	data.RiskFromUniqueItems = assessment.RiskFromUniqueItems
	data.RiskFromAssignments = assessment.RiskFromAssignments
	data.RiskFromSharingLinks = assessment.RiskFromSharingLinks
	data.RiskFromElevatedAccess = assessment.RiskFromElevatedAccess

	return data, nil
}

// calculatePrincipalTypes counts different types of principals.
func (s *PermissionService) calculatePrincipalTypes(assignments []*sharepoint.Assignment) (users, groups, sharingLinks int) {
	for _, assignment := range assignments {
		switch assignment.Principal.PrincipalType {
		case sharepoint.PrincipalTypeUser:
			users++
		case sharepoint.PrincipalTypeSecurity, sharepoint.PrincipalTypeSharePointGroup:
			groups++
		default:
			// Check if it's a sharing link based on login name pattern
			if len(assignment.Principal.LoginName) > 12 && assignment.Principal.LoginName[:12] == "SharingLinks" {
				sharingLinks++
			} else {
				groups++ // Default unknown types to groups
			}
		}
	}
	return
}

// calculateRoleDistribution counts assignments by role type.
func (s *PermissionService) calculateRoleDistribution(assignments []*sharepoint.Assignment) (fullControl, contribute, read, limitedAccess, other int) {
	for _, assignment := range assignments {
		switch assignment.RoleDefinition.Name {
		case "Full Control":
			fullControl++
		case "Contribute":
			contribute++
		case "Read":
			read++
		case "Limited Access", "Web-Only Limited Access":
			limitedAccess++
		default:
			other++
		}
	}
	return
}
