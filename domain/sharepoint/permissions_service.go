package sharepoint

import (
	"math"
)

// PermissionAnalysis represents the results of analyzing permissions for an object
type PermissionAnalysis struct {
	ObjectType      string
	ObjectKey       string
	SiteID          int64
	HasUniquePerms  bool
	AssignmentCount int
	PrincipalCount  int
	RiskScore       float64
	RiskFactors     []string
	Recommendations []string
}

// InheritanceAnalysis represents analysis of permission inheritance patterns
type InheritanceAnalysis struct {
	TotalObjects           int
	ObjectsWithUniquePerms int
	InheritanceBreakRatio  float64
	ComplexityScore        float64
}

// SharePointRiskData represents data needed for SharePoint-specific risk calculation
type SharePointRiskData struct {
	TotalItems         int64
	ItemsWithUnique    int64
	TotalAssignments   int
	LimitedAccessCount int
	FullControlCount   int
	ContributeCount    int
	SharingLinkCount   int
}

// SharePointRiskAssessment represents detailed risk assessment for SharePoint objects
type SharePointRiskAssessment struct {
	RiskScore              float64
	RiskLevel              string // "Low", "Medium", "High"
	RiskFromUniqueItems    float64
	RiskFromAssignments    float64
	RiskFromSharingLinks   float64
	RiskFromElevatedAccess float64
}

// PermissionsService provides business logic for analyzing SharePoint permissions
type PermissionsService struct {
	// Pure business logic - no external dependencies
	// Future: could add configuration for risk thresholds, etc.
}

// NewPermissionsService creates a new permission service
func NewPermissionsService() *PermissionsService {
	return &PermissionsService{}
}

// AnalyzeAssignments analyzes a collection of role assignments for business insights
func (s *PermissionsService) AnalyzeAssignments(assignments []*RoleAssignment) *PermissionAnalysis {
	if len(assignments) == 0 {
		return &PermissionAnalysis{
			HasUniquePerms:  false,
			AssignmentCount: 0,
			PrincipalCount:  0,
			RiskScore:       0,
			RiskFactors:     []string{},
			Recommendations: []string{},
		}
	}

	// Basic analysis - foundation for business logic
	analysis := &PermissionAnalysis{
		ObjectType:      assignments[0].ObjectType,
		ObjectKey:       assignments[0].ObjectKey,
		SiteID:          assignments[0].SiteID,
		HasUniquePerms:  true, // If assignments exist, object has unique permissions
		AssignmentCount: len(assignments),
		PrincipalCount:  s.countUniquePrincipals(assignments),
		RiskFactors:     []string{},
		Recommendations: []string{},
	}

	// Calculate risk score based on business rules
	analysis.RiskScore = s.calculateRiskScore(analysis)

	// Identify risk factors
	analysis.RiskFactors = s.identifyRiskFactors(analysis)

	// Generate recommendations
	analysis.Recommendations = s.generateRecommendations(analysis)

	return analysis
}

// AnalyzeInheritancePatterns analyzes inheritance patterns across multiple objects
func (s *PermissionsService) AnalyzeInheritancePatterns(allAnalyses []*PermissionAnalysis) *InheritanceAnalysis {
	if len(allAnalyses) == 0 {
		return &InheritanceAnalysis{}
	}

	uniqueCount := 0
	for _, analysis := range allAnalyses {
		if analysis.HasUniquePerms {
			uniqueCount++
		}
	}

	totalObjects := len(allAnalyses)
	inheritanceBreakRatio := float64(uniqueCount) / float64(totalObjects)

	return &InheritanceAnalysis{
		TotalObjects:           totalObjects,
		ObjectsWithUniquePerms: uniqueCount,
		InheritanceBreakRatio:  inheritanceBreakRatio,
		ComplexityScore:        s.calculateComplexityScore(inheritanceBreakRatio, allAnalyses),
	}
}

// IdentifyExcessivePermissions identifies objects with potentially excessive permissions
// This business logic can be reused for audit reporting and future remediation
func (s *PermissionsService) IdentifyExcessivePermissions(analyses []*PermissionAnalysis, riskThreshold float64) []*PermissionAnalysis {
	var excessive []*PermissionAnalysis

	for _, analysis := range analyses {
		if analysis.RiskScore >= riskThreshold {
			excessive = append(excessive, analysis)
		}
	}

	return excessive
}

// CalculateSharePointRiskAssessment performs the same risk calculation as the existing permission service
// This is extracted business logic that can be reused by audit, remediation, compliance features
func (s *PermissionsService) CalculateSharePointRiskAssessment(riskData *SharePointRiskData) *SharePointRiskAssessment {
	assessment := &SharePointRiskAssessment{}

	// Primary risk: Percentage of items with unique permissions (0-50 points)
	// This is the most important indicator for SharePoint security
	uniqueItemsRisk := 0.0
	if riskData.TotalItems > 0 {
		itemUniquePercentage := float64(riskData.ItemsWithUnique) / float64(riskData.TotalItems)
		uniqueItemsRisk = itemUniquePercentage * 50.0
	}
	assessment.RiskFromUniqueItems = uniqueItemsRisk

	// High-risk assignments: Exclude limited access since it's low risk (0-25 points, logarithmic scale)
	// Limited Access is automatically granted by SharePoint for navigation - low security risk
	assignmentRisk := 0.0
	highRiskAssignments := riskData.TotalAssignments - riskData.LimitedAccessCount
	if highRiskAssignments > 0 {
		// Logarithmic scale: 10 assignments = ~8 points, 100 = ~17 points, 1000 = ~25 points
		complexityScore := math.Log10(float64(highRiskAssignments)) * 8.0
		assignmentRisk = math.Min(complexityScore, 25.0)
	}
	assessment.RiskFromAssignments = assignmentRisk

	// Sharing links risk (0-15 points)
	// External sharing is a significant security concern in SharePoint
	sharingRisk := math.Min(float64(riskData.SharingLinkCount)*1.5, 15.0)
	assessment.RiskFromSharingLinks = sharingRisk

	// Elevated permissions risk (0-10 points)
	// Full Control and Contribute are high-privilege access levels
	elevatedRisk := math.Min(float64(riskData.FullControlCount+riskData.ContributeCount)*1.5, 10.0)
	assessment.RiskFromElevatedAccess = elevatedRisk

	// Calculate total risk score
	riskScore := uniqueItemsRisk + assignmentRisk + sharingRisk + elevatedRisk

	// Special case: If only limited access/read permissions, no items with unique perms, and no sharing links - very low risk
	// This represents a well-governed SharePoint site with proper inheritance
	if riskData.ItemsWithUnique == 0 && riskData.SharingLinkCount == 0 &&
		(riskData.FullControlCount+riskData.ContributeCount) == 0 {
		riskScore = math.Min(riskScore*0.5, 15.0) // Cap at 15 points for low-risk scenarios

		// Update breakdown to reflect the reduction
		assessment.RiskFromUniqueItems *= 0.5
		assessment.RiskFromAssignments *= 0.5
		assessment.RiskFromSharingLinks *= 0.5
		assessment.RiskFromElevatedAccess *= 0.5
	}

	// Determine risk level based on SharePoint-specific thresholds
	// These thresholds are based on SharePoint governance best practices
	riskLevel := "Low"
	if riskScore >= 50.0 {
		riskLevel = "High"
	} else if riskScore >= 20.0 {
		riskLevel = "Medium"
	}

	assessment.RiskScore = math.Min(riskScore, 100.0)
	assessment.RiskLevel = riskLevel

	return assessment
}

// Private helper methods - pure business logic

// countUniquePrincipals counts unique principals across assignments
func (s *PermissionsService) countUniquePrincipals(assignments []*RoleAssignment) int {
	principals := make(map[int64]bool)
	for _, assignment := range assignments {
		principals[assignment.PrincipalID] = true
	}
	return len(principals)
}

// calculateRiskScore applies business rules to calculate risk score
func (s *PermissionsService) calculateRiskScore(analysis *PermissionAnalysis) float64 {
	// Business logic: Risk scoring algorithm
	baseRisk := 0.0

	// Risk factor: High number of assignments
	if analysis.AssignmentCount > 10 {
		baseRisk += 0.3
	} else if analysis.AssignmentCount > 5 {
		baseRisk += 0.1
	}

	// Risk factor: High number of principals
	if analysis.PrincipalCount > 20 {
		baseRisk += 0.4
	} else if analysis.PrincipalCount > 10 {
		baseRisk += 0.2
	}

	// Cap at 1.0
	if baseRisk > 1.0 {
		baseRisk = 1.0
	}

	return baseRisk
}

// identifyRiskFactors identifies specific risk factors
func (s *PermissionsService) identifyRiskFactors(analysis *PermissionAnalysis) []string {
	var factors []string

	if analysis.AssignmentCount > 10 {
		factors = append(factors, "High number of permission assignments")
	}

	if analysis.PrincipalCount > 20 {
		factors = append(factors, "High number of users/groups with access")
	}

	if analysis.HasUniquePerms {
		factors = append(factors, "Breaks permission inheritance")
	}

	return factors
}

// generateRecommendations generates business recommendations
func (s *PermissionsService) generateRecommendations(analysis *PermissionAnalysis) []string {
	var recommendations []string

	if analysis.RiskScore > 0.7 {
		recommendations = append(recommendations, "Review and simplify permission structure")
	}

	if analysis.PrincipalCount > 20 {
		recommendations = append(recommendations, "Consider using SharePoint groups to reduce direct user assignments")
	}

	if analysis.AssignmentCount > 15 {
		recommendations = append(recommendations, "Audit individual permissions for redundancy")
	}

	return recommendations
}

// calculateComplexityScore calculates overall complexity across multiple objects
func (s *PermissionsService) calculateComplexityScore(inheritanceBreakRatio float64, analyses []*PermissionAnalysis) float64 {
	// Business logic: Complexity algorithm
	complexityScore := inheritanceBreakRatio * 0.7 // Inheritance breaks contribute to complexity

	// Add complexity based on average risk scores
	totalRisk := 0.0
	for _, analysis := range analyses {
		totalRisk += analysis.RiskScore
	}

	if len(analyses) > 0 {
		avgRisk := totalRisk / float64(len(analyses))
		complexityScore += avgRisk * 0.3
	}

	if complexityScore > 1.0 {
		complexityScore = 1.0
	}

	return complexityScore
}
