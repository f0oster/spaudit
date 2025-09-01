package sharepoint

import (
	"path/filepath"
	"strings"
)

// ContentAnalysis represents analysis results for content items
type ContentAnalysis struct {
	TotalItems      int64
	FilesCount      int64
	FoldersCount    int64
	ListItemsCount  int64
	ItemsWithUnique int64

	// File type analysis
	FilesByExtension map[string]int64
	MostCommonTypes  []string
	DocumentsCount   int64 // Office docs, PDFs, etc.
	ImagesCount      int64 // Images and media
	ArchivesCount    int64 // ZIP, RAR, etc.
	OtherFilesCount  int64 // Everything else

	// Size and structure analysis
	EmptyFoldersCount  int64
	DeepFoldersCount   int64 // Folders with deep nesting
	OrphanedItemsCount int64 // Items without proper list association
}

// ContentRiskAssessment represents risk analysis for content
type ContentRiskAssessment struct {
	RiskScore       float64 // 0-100
	RiskLevel       string  // "Low", "Medium", "High"
	RiskFactors     []string
	Recommendations []string

	// Specific risk metrics
	SensitiveFilesCount  int64 // Files with potentially sensitive extensions
	LargeFilesCount      int64 // Files that might be unusually large
	ExecutableFilesCount int64 // Potentially dangerous file types
}

// ListContentSummary represents content summary for a specific list
type ListContentSummary struct {
	ListID            string
	ListTitle         string
	IsDocumentLibrary bool
	IsCustomList      bool

	Analysis       *ContentAnalysis
	RiskAssessment *ContentRiskAssessment
}

// ContentService analyzes content and scans files.
type ContentService struct {
	// Configuration for file type categorization, risk thresholds, etc.
	sensitiveExtensions  []string
	documentExtensions   []string
	imageExtensions      []string
	archiveExtensions    []string
	executableExtensions []string
}

// NewContentService creates a new content service
func NewContentService() *ContentService {
	return &ContentService{
		sensitiveExtensions:  []string{".key", ".pem", ".p12", ".pfx", ".cer", ".crt", ".config", ".env", ".ini"},
		documentExtensions:   []string{".docx", ".doc", ".xlsx", ".xls", ".pptx", ".ppt", ".pdf", ".txt", ".rtf", ".odt", ".ods", ".odp"},
		imageExtensions:      []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".svg", ".webp", ".ico"},
		archiveExtensions:    []string{".zip", ".rar", ".7z", ".tar", ".gz", ".bz2"},
		executableExtensions: []string{".exe", ".msi", ".bat", ".cmd", ".ps1", ".sh", ".scr", ".com", ".pif", ".jar"},
	}
}

// AnalyzeItems analyzes content metrics for a collection of items
func (s *ContentService) AnalyzeItems(items []*Item) *ContentAnalysis {
	analysis := &ContentAnalysis{
		FilesByExtension: make(map[string]int64),
	}

	for _, item := range items {
		analysis.TotalItems++

		if item.HasUnique {
			analysis.ItemsWithUnique++
		}

		// Classify item type
		if item.IsDocument() {
			analysis.FilesCount++
			s.analyzeFile(item, analysis)
		} else if item.IsDirectory() {
			analysis.FoldersCount++
			s.analyzeFolder(item, analysis)
		} else if item.IsListItem() {
			analysis.ListItemsCount++
		}
	}

	// Post-processing analysis
	analysis.MostCommonTypes = s.getMostCommonFileTypes(analysis.FilesByExtension)

	return analysis
}

// AnalyzeListContent analyzes content for a specific SharePoint list
func (s *ContentService) AnalyzeListContent(list *List, items []*Item) *ListContentSummary {
	analysis := s.AnalyzeItems(items)
	riskAssessment := s.AssessContentRisk(analysis)

	return &ListContentSummary{
		ListID:            list.ID,
		ListTitle:         list.Title,
		IsDocumentLibrary: list.IsDocumentLibrary(),
		IsCustomList:      list.IsCustomList(),
		Analysis:          analysis,
		RiskAssessment:    riskAssessment,
	}
}

// AssessContentRisk performs risk analysis on content analysis results
func (s *ContentService) AssessContentRisk(analysis *ContentAnalysis) *ContentRiskAssessment {
	riskScore := s.calculateContentRiskScore(analysis)
	riskFactors := s.identifyContentRiskFactors(analysis)
	recommendations := s.generateContentRecommendations(analysis)

	// Ensure slices are never nil
	if riskFactors == nil {
		riskFactors = []string{}
	}
	if recommendations == nil {
		recommendations = []string{}
	}

	return &ContentRiskAssessment{
		RiskScore:            riskScore,
		RiskLevel:            s.determineContentRiskLevel(riskScore),
		RiskFactors:          riskFactors,
		Recommendations:      recommendations,
		SensitiveFilesCount:  analysis.FilesByExtension[".key"] + analysis.FilesByExtension[".pem"] + analysis.FilesByExtension[".config"],
		ExecutableFilesCount: analysis.FilesByExtension[".exe"] + analysis.FilesByExtension[".msi"] + analysis.FilesByExtension[".bat"],
	}
}

// ClassifyFileExtension categorizes a file extension into a broad category
func (s *ContentService) ClassifyFileExtension(extension string) string {
	ext := strings.ToLower(extension)

	if s.containsExtension(s.documentExtensions, ext) {
		return "Document"
	}
	if s.containsExtension(s.imageExtensions, ext) {
		return "Image"
	}
	if s.containsExtension(s.archiveExtensions, ext) {
		return "Archive"
	}
	if s.containsExtension(s.executableExtensions, ext) {
		return "Executable"
	}
	if s.containsExtension(s.sensitiveExtensions, ext) {
		return "Sensitive"
	}

	return "Other"
}

// IsSensitiveFile determines if a file has a potentially sensitive extension
func (s *ContentService) IsSensitiveFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return s.containsExtension(s.sensitiveExtensions, ext)
}

// IsExecutableFile determines if a file has an executable extension
func (s *ContentService) IsExecutableFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return s.containsExtension(s.executableExtensions, ext)
}

// Private helper methods for content analysis

// analyzeFile performs specific analysis on file items
func (s *ContentService) analyzeFile(item *Item, analysis *ContentAnalysis) {
	// Extract file extension from name or URL
	var ext string
	if item.Name != "" {
		ext = strings.ToLower(filepath.Ext(item.Name))
	} else if item.URL != "" {
		ext = strings.ToLower(filepath.Ext(item.URL))
	}

	if ext != "" {
		analysis.FilesByExtension[ext]++

		// Categorize by type
		category := s.ClassifyFileExtension(ext)
		switch category {
		case "Document":
			analysis.DocumentsCount++
		case "Image":
			analysis.ImagesCount++
		case "Archive":
			analysis.ArchivesCount++
		default:
			analysis.OtherFilesCount++
		}
	} else {
		// Files with no extension are counted as "other"
		analysis.OtherFilesCount++
	}
}

// analyzeFolder performs specific analysis on folder items
func (s *ContentService) analyzeFolder(item *Item, analysis *ContentAnalysis) {
	// Folders are counted for basic analysis
}

// calculateContentRiskScore calculates risk score based on content analysis
func (s *ContentService) calculateContentRiskScore(analysis *ContentAnalysis) float64 {
	baseRisk := 0.0

	// Risk from total content volume (0-20 points)
	if analysis.TotalItems > 10000 {
		baseRisk += 20.0
	} else if analysis.TotalItems > 5000 {
		baseRisk += 15.0
	} else if analysis.TotalItems > 1000 {
		baseRisk += 10.0
	} else if analysis.TotalItems > 500 {
		baseRisk += 8.0
	} else if analysis.TotalItems > 100 {
		baseRisk += 5.0
	}

	// Risk from sensitive files (0-30 points)
	sensitiveCount := int64(0)
	for ext := range analysis.FilesByExtension {
		if s.containsExtension(s.sensitiveExtensions, ext) {
			sensitiveCount += analysis.FilesByExtension[ext]
		}
	}
	if sensitiveCount > 50 {
		baseRisk += 30.0
	} else if sensitiveCount > 20 {
		baseRisk += 20.0
	} else if sensitiveCount > 5 {
		baseRisk += 10.0
	} else if sensitiveCount > 0 {
		baseRisk += 5.0
	}

	// Risk from executable files (0-25 points)
	executableCount := int64(0)
	for ext := range analysis.FilesByExtension {
		if s.containsExtension(s.executableExtensions, ext) {
			executableCount += analysis.FilesByExtension[ext]
		}
	}
	if executableCount > 20 {
		baseRisk += 25.0
	} else if executableCount > 10 {
		baseRisk += 15.0
	} else if executableCount > 5 {
		baseRisk += 10.0
	} else if executableCount > 0 {
		baseRisk += 5.0
	}

	// Risk from items with unique permissions (0-15 points)
	if analysis.ItemsWithUnique > 0 {
		uniqueRatio := float64(analysis.ItemsWithUnique) / float64(analysis.TotalItems)
		baseRisk += uniqueRatio * 15.0
	}

	// Risk from file diversity (many different file types might indicate uncontrolled content)
	typeCount := len(analysis.FilesByExtension)
	if typeCount > 50 {
		baseRisk += 10.0
	} else if typeCount > 30 {
		baseRisk += 5.0
	}

	// Cap at 100
	if baseRisk > 100.0 {
		baseRisk = 100.0
	}

	return baseRisk
}

// determineContentRiskLevel determines risk level based on risk score
func (s *ContentService) determineContentRiskLevel(riskScore float64) string {
	if riskScore >= 70.0 {
		return "High"
	} else if riskScore >= 40.0 {
		return "Medium"
	}
	return "Low"
}

// identifyContentRiskFactors identifies specific risk factors in content
func (s *ContentService) identifyContentRiskFactors(analysis *ContentAnalysis) []string {
	var factors []string

	if analysis.TotalItems > 5000 {
		factors = append(factors, "Large volume of content items")
	}

	// Check for sensitive files
	sensitiveCount := int64(0)
	for ext := range analysis.FilesByExtension {
		if s.containsExtension(s.sensitiveExtensions, ext) {
			sensitiveCount += analysis.FilesByExtension[ext]
		}
	}
	if sensitiveCount > 0 {
		factors = append(factors, "Contains files with potentially sensitive extensions")
	}

	// Check for executable files
	executableCount := int64(0)
	for ext := range analysis.FilesByExtension {
		if s.containsExtension(s.executableExtensions, ext) {
			executableCount += analysis.FilesByExtension[ext]
		}
	}
	if executableCount > 0 {
		factors = append(factors, "Contains executable files")
	}

	if analysis.ItemsWithUnique > 0 {
		uniqueRatio := float64(analysis.ItemsWithUnique) / float64(analysis.TotalItems)
		if uniqueRatio > 0.1 {
			factors = append(factors, "High percentage of items with unique permissions")
		}
	}

	if len(analysis.FilesByExtension) > 30 {
		factors = append(factors, "Wide variety of file types (potential uncontrolled content)")
	}

	return factors
}

// generateContentRecommendations generates recommendations for content management
func (s *ContentService) generateContentRecommendations(analysis *ContentAnalysis) []string {
	var recommendations []string

	if analysis.TotalItems > 10000 {
		recommendations = append(recommendations, "Consider implementing content lifecycle management")
	}

	// Check for sensitive files
	sensitiveCount := int64(0)
	for ext := range analysis.FilesByExtension {
		if s.containsExtension(s.sensitiveExtensions, ext) {
			sensitiveCount += analysis.FilesByExtension[ext]
		}
	}
	if sensitiveCount > 0 {
		recommendations = append(recommendations, "Review and secure files with sensitive extensions")
	}

	// Check for executable files
	executableCount := int64(0)
	for ext := range analysis.FilesByExtension {
		if s.containsExtension(s.executableExtensions, ext) {
			executableCount += analysis.FilesByExtension[ext]
		}
	}
	if executableCount > 0 {
		recommendations = append(recommendations, "Review and restrict executable files")
	}

	if analysis.ItemsWithUnique > analysis.TotalItems/2 {
		recommendations = append(recommendations, "Review unique permissions - many items have custom access")
	}

	if len(analysis.FilesByExtension) > 50 {
		recommendations = append(recommendations, "Implement file type governance policies")
	}

	return recommendations
}

// getMostCommonFileTypes returns the most frequently used file extensions
func (s *ContentService) getMostCommonFileTypes(filesByExtension map[string]int64) []string {
	if len(filesByExtension) == 0 {
		return []string{}
	}

	// Simple implementation - return extensions with more than 5 files
	var common []string
	for ext, count := range filesByExtension {
		if count > 5 {
			common = append(common, ext)
		}
	}

	return common
}

// containsExtension checks if an extension is in the given slice
func (s *ContentService) containsExtension(extensions []string, ext string) bool {
	for _, e := range extensions {
		if e == ext {
			return true
		}
	}
	return false
}
