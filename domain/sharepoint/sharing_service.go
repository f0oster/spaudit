package sharepoint

import (
	"regexp"
	"strings"
)

// SharingLinkInfo represents parsed information from a SharePoint sharing link
type SharingLinkInfo struct {
	ItemGUID   string
	SharingID  string
	LinkType   string
	IsValid    bool
	IsFlexible bool
}

// SharingRiskAssessment represents risk analysis results for sharing links
type SharingRiskAssessment struct {
	TotalLinks        int
	FlexibleLinks     int
	OrganizationLinks int
	ExternalLinks     int
	EditLinks         int
	ViewLinks         int
	RiskScore         float64
	RiskLevel         string // "Low", "Medium", "High"
	RiskFactors       []string
	Recommendations   []string
}

// SharingPatternAnalysis represents analysis of sharing patterns across items
type SharingPatternAnalysis struct {
	ItemsWithSharing    int
	TotalItems          int
	SharingRatio        float64
	MostCommonLinkTypes []string
	HighRiskItems       []string
}

// SharingService provides business logic for analyzing SharePoint sharing links
type SharingService struct {
	// Pure business logic - no external dependencies
	// Future: could add configuration for risk thresholds, link type rules, etc.
}

// NewSharingService creates a new sharing service
func NewSharingService() *SharingService {
	return &SharingService{}
}

// ParseFlexibleSharingLink extracts information from flexible sharing link format
// Pattern: SharingLinks.{GUID}.Flexible.{GUID}
// This is SharePoint's format for flexible sharing links (can be used for view or edit)
func (s *SharingService) ParseFlexibleSharingLink(loginName string) *SharingLinkInfo {
	// Pattern to match: SharingLinks.{GUID}.Flexible.{GUID}
	pattern := `SharingLinks\.([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\.Flexible\.([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})`

	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(loginName)

	if len(matches) >= 3 {
		return &SharingLinkInfo{
			ItemGUID:   matches[1],
			SharingID:  matches[2],
			LinkType:   "Flexible",
			IsValid:    true,
			IsFlexible: true,
		}
	}

	return &SharingLinkInfo{
		IsValid: false,
	}
}

// ParseSharingLink extracts information from any type of SharePoint sharing link
// Supports multiple formats:
//   - SharingLinks.{GUID}.Flexible.{GUID}
//   - SharingLinks.{GUID}.OrganizationEdit.{GUID}
//   - SharingLinks.{GUID}.OrganizationView.{GUID}
//   - And other SharePoint sharing link patterns
func (s *SharingService) ParseSharingLink(loginName string) *SharingLinkInfo {
	// Pattern to match: SharingLinks.{GUID}.{LinkType}.{GUID}
	pattern := `SharingLinks\.([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\.([^.]+)\.([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})`

	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(loginName)

	if len(matches) >= 4 {
		linkType := matches[2]
		return &SharingLinkInfo{
			ItemGUID:   matches[1],
			SharingID:  matches[3],
			LinkType:   linkType,
			IsValid:    true,
			IsFlexible: linkType == "Flexible",
		}
	}

	return &SharingLinkInfo{
		IsValid: false,
	}
}

// ValidateSharingLinkFormat validates if a login name follows SharePoint sharing link patterns
func (s *SharingService) ValidateSharingLinkFormat(loginName string) bool {
	info := s.ParseSharingLink(loginName)
	return info.IsValid
}

// AnalyzeSharingRisk performs risk analysis on a collection of sharing links
// This business logic can be reused for audit reporting and future remediation
func (s *SharingService) AnalyzeSharingRisk(links []*SharingLink) *SharingRiskAssessment {
	if len(links) == 0 {
		return &SharingRiskAssessment{
			RiskScore:       0,
			RiskLevel:       "Low",
			RiskFactors:     []string{},
			Recommendations: []string{},
		}
	}

	assessment := &SharingRiskAssessment{
		TotalLinks: len(links),
	}

	// Analyze link types and patterns
	linkTypeCounts := make(map[string]int)
	for _, link := range links {
		// Parse each sharing link to get detailed information
		info := s.ParseSharingLink("SharingLinks." + link.ItemGUID + ".Unknown." + link.ShareID)

		if info.IsValid {
			linkTypeCounts[info.LinkType]++

			switch {
			case info.IsFlexible:
				assessment.FlexibleLinks++
			case strings.Contains(strings.ToLower(info.LinkType), "organization"):
				assessment.OrganizationLinks++
			case strings.Contains(strings.ToLower(info.LinkType), "edit"):
				assessment.EditLinks++
			case strings.Contains(strings.ToLower(info.LinkType), "view"):
				assessment.ViewLinks++
			}
		}
	}

	// Calculate risk score based on SharePoint sharing best practices
	riskScore := s.calculateSharingRiskScore(assessment)
	assessment.RiskScore = riskScore

	// Determine risk level
	assessment.RiskLevel = s.determineSharingRiskLevel(riskScore)

	// Identify risk factors
	assessment.RiskFactors = s.identifySharingRiskFactors(assessment)

	// Generate recommendations
	assessment.Recommendations = s.generateSharingRecommendations(assessment)

	return assessment
}

// AnalyzeSharingPatterns analyzes sharing patterns across multiple items
func (s *SharingService) AnalyzeSharingPatterns(items []*Item, sharingData map[string][]*SharingLink) *SharingPatternAnalysis {
	if len(items) == 0 {
		return &SharingPatternAnalysis{}
	}

	analysis := &SharingPatternAnalysis{
		TotalItems: len(items),
	}

	linkTypeCounts := make(map[string]int)
	var highRiskItems []string

	for _, item := range items {
		if links, hasSharing := sharingData[item.GUID]; hasSharing && len(links) > 0 {
			analysis.ItemsWithSharing++

			// Analyze risk for this item
			riskAssessment := s.AnalyzeSharingRisk(links)
			if riskAssessment.RiskLevel == "High" {
				highRiskItems = append(highRiskItems, item.GUID)
			}

			// Count link types
			for _, link := range links {
				info := s.ParseSharingLink("SharingLinks." + link.ItemGUID + ".Unknown." + link.ShareID)
				if info.IsValid {
					linkTypeCounts[info.LinkType]++
				}
			}
		}
	}

	analysis.SharingRatio = float64(analysis.ItemsWithSharing) / float64(analysis.TotalItems)
	analysis.HighRiskItems = highRiskItems
	analysis.MostCommonLinkTypes = s.getMostCommonLinkTypes(linkTypeCounts)

	return analysis
}

// Private helper methods - pure business logic

// calculateSharingRiskScore applies business rules to calculate sharing risk score
func (s *SharingService) calculateSharingRiskScore(assessment *SharingRiskAssessment) float64 {
	baseRisk := 0.0

	// Risk from total number of sharing links (0-40 points)
	if assessment.TotalLinks > 100 {
		baseRisk += 40.0
	} else if assessment.TotalLinks > 50 {
		baseRisk += 30.0
	} else if assessment.TotalLinks > 20 {
		baseRisk += 20.0
	} else if assessment.TotalLinks > 10 {
		baseRisk += 10.0
	} else if assessment.TotalLinks > 0 {
		baseRisk += 5.0
	}

	// Risk from flexible links (higher risk - 0-30 points)
	flexibleRatio := float64(assessment.FlexibleLinks) / float64(assessment.TotalLinks)
	baseRisk += flexibleRatio * 30.0

	// Risk from edit links (medium risk - 0-15 points)
	if assessment.TotalLinks > 0 {
		editRatio := float64(assessment.EditLinks) / float64(assessment.TotalLinks)
		baseRisk += editRatio * 15.0
	}

	// Bonus risk for high concentration of risky link types
	if assessment.FlexibleLinks > 10 {
		baseRisk += 10.0
	}

	// Additional risk for mixed flexible+edit scenarios (compounding risk)
	if assessment.FlexibleLinks > 20 && assessment.EditLinks > 10 {
		baseRisk += 5.0
	}

	// Additional risk for extreme scenarios (500+ links)
	if assessment.TotalLinks > 500 {
		baseRisk += 20.0
	}

	// Cap at 100
	if baseRisk > 100.0 {
		baseRisk = 100.0
	}

	return baseRisk
}

// determineSharingRiskLevel determines risk level based on risk score
func (s *SharingService) determineSharingRiskLevel(riskScore float64) string {
	if riskScore >= 60.0 {
		return "High"
	} else if riskScore >= 30.0 {
		return "Medium"
	}
	return "Low"
}

// identifySharingRiskFactors identifies specific risk factors in sharing configuration
func (s *SharingService) identifySharingRiskFactors(assessment *SharingRiskAssessment) []string {
	var factors []string

	if assessment.TotalLinks > 20 {
		factors = append(factors, "High number of sharing links")
	}

	if assessment.FlexibleLinks > 5 {
		factors = append(factors, "Multiple flexible sharing links (can be used for view or edit)")
	}

	if assessment.EditLinks > 10 {
		factors = append(factors, "High number of edit-capable sharing links")
	}

	if assessment.FlexibleLinks > 0 && assessment.EditLinks > 0 {
		factors = append(factors, "Mix of flexible and edit links increases complexity")
	}

	if assessment.TotalLinks > 0 && assessment.OrganizationLinks == 0 {
		factors = append(factors, "No organization-scoped links (potential external sharing)")
	}

	return factors
}

// generateSharingRecommendations generates business recommendations for sharing management
func (s *SharingService) generateSharingRecommendations(assessment *SharingRiskAssessment) []string {
	var recommendations []string

	if assessment.RiskScore > 60.0 {
		recommendations = append(recommendations, "Review and audit all sharing links for necessity")
	}

	if assessment.FlexibleLinks > 5 {
		recommendations = append(recommendations, "Consider replacing flexible links with specific view/edit links")
	}

	if assessment.EditLinks > assessment.ViewLinks && assessment.ViewLinks > 0 {
		recommendations = append(recommendations, "Review edit permissions - consider if view-only access is sufficient")
	}

	if assessment.TotalLinks > 20 {
		recommendations = append(recommendations, "Implement sharing link governance and expiration policies")
	}

	if assessment.OrganizationLinks == 0 && assessment.TotalLinks > 0 {
		recommendations = append(recommendations, "Review external sharing permissions and consider organization-scoped links")
	}

	return recommendations
}

// getMostCommonLinkTypes returns the most frequently used link types
func (s *SharingService) getMostCommonLinkTypes(linkTypeCounts map[string]int) []string {
	if len(linkTypeCounts) == 0 {
		return []string{}
	}

	// Simple implementation - return types with more than 1 occurrence
	var common []string
	for linkType, count := range linkTypeCounts {
		if count > 1 {
			common = append(common, linkType)
		}
	}

	return common
}
