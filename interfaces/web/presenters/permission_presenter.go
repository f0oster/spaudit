package presenters

import (
	"fmt"
	"strings"

	"spaudit/application"
	"spaudit/domain/sharepoint"
)

// Permission-related view data structures

// Site represents basic site information for UI display.
type Site struct {
	SiteID  int64
	SiteURL string
	Title   string
}

// SiteWithMetadata represents site data enriched with computed metadata for dashboard display.
type SiteWithMetadata struct {
	SiteID          int64
	SiteURL         string
	Title           string
	Description     string
	TotalLists      int
	ListsWithUnique int
	LastAuditDate   string
	DaysAgo         int
}

// ListSummary represents list information optimized for table display and navigation.
type ListSummary struct {
	SiteID       int64
	SiteURL      string
	ListID       string
	WebID        string
	Title        string
	URL          string
	ItemCount    int64
	HasUnique    bool
	WebTitle     string
	LastModified string
	AuditRunID   int64
}

// ItemSummary represents SharePoint item information for permission analysis display.
type ItemSummary struct {
	SiteID    int64
	ItemGUID  string
	ListID    string
	ItemID    int64
	URL       string
	IsFile    bool
	IsFolder  bool
	HasUnique bool
	Name      string
}

// Assignment represents a permission assignment for UI display.
type Assignment struct {
	PrincipalTitle string
	LoginName      string
	PrincipalType  int32
	RoleName       string
	Inherited      bool
}

// ExpandableAssignment extends Assignment with root cause analysis for detailed permission investigation.
// RootCauseVM represents a single permission source for the UI
type RootCauseVM struct {
	Type         string // "SHARING_LINK", "SAME_WEB_INHERITANCE", "SYSTEM_GROUP", "UNKNOWN"
	Detail       string
	SourceObject string
	SourceRole   string
}

type ExpandableAssignment struct {
	Assignment
	// Root cause analysis (loaded on-demand)
	RootCauses    []RootCauseVM // All detected permission sources
	HasRootCauses bool          // Whether any root causes were found
	// Unique identifier for HTMX interactions
	UniqueID string
}

type AssignmentCollection struct {
	Assignments      []Assignment
	HasLimitedAccess bool
	HasSharingLinks  bool
	HasSiteGroups    bool
}

type ExpandableAssignmentCollection struct {
	Assignments      []ExpandableAssignment
	HasLimitedAccess bool
	HasSharingLinks  bool
	HasSiteGroups    bool
}

type SharingLink struct {
	SiteID             int64
	LinkID             string
	ItemGUID           string
	ItemName           string
	ItemURL            string
	IsFile             bool
	IsFolder           bool
	URL                string
	LinkKind           int64
	LinkKindName       string
	Scope              int64
	ScopeName          string
	IsActive           bool
	IsDefault          bool
	IsEditLink         bool
	IsReviewLink       bool
	CreatedAt          string
	LastModifiedAt     string
	TotalMembersCount  int64
	ActualMembersCount int64
	CreatedByTitle     string
	CreatedByLogin     string
	ModifiedByTitle    string
	ModifiedByLogin    string
}

type SharingLinkMember struct {
	SiteID        int64
	PrincipalID   int64
	Title         string
	LoginName     string
	Email         string
	PrincipalType int64
	IsGroup       bool
}

type ResolvedAssignment struct {
	Assignment
	RootCauses []RootCauseVM // All detected permission sources
}

type ResolvedAssignmentCollection struct {
	ResolvedAssignments []ResolvedAssignment
	SharingLinkCount    int
	InheritanceCount    int
	SystemGroupCount    int
	UnknownCount        int
}

// ListAnalytics represents comprehensive permission analytics for a SharePoint list.
// Used for dashboard display and risk assessment visualization.
type ListAnalytics struct {
	// Basic list info
	List ListSummary

	// Permission analytics
	TotalAssignments     int
	UniqueAssignments    int
	InheritedAssignments int
	ItemLevelAssignments int
	SharingLinkCount     int

	// Item analytics
	TotalItems      int64
	ItemsWithUnique int64
	FilesCount      int64
	FoldersCount    int64

	// Risk assessment
	PermissionRiskLevel string  // "Low", "Medium", "High"
	PermissionRiskScore float64 // 0-100

	// Risk breakdown (for transparency)
	RiskFromUniqueItems    float64 // Points from items with unique permissions
	RiskFromAssignments    float64 // Points from permission assignments
	RiskFromSharingLinks   float64 // Points from sharing links
	RiskFromElevatedAccess float64 // Points from Full Control/Contribute

	// Principal breakdown
	UserCount        int
	GroupCount       int
	SharingLinkUsers int

	// Sharing link type breakdown
	FlexibleLinksCount    int
	OrganizationViewCount int
	OrganizationEditCount int
	AnonymousViewCount    int
	AnonymousEditCount    int
	DirectLinksCount      int
	OtherLinksCount       int

	// Role distribution
	FullControlCount   int
	ContributeCount    int
	ReadCount          int
	LimitedAccessCount int
	OtherRolesCount    int
}

// PermissionPresenter transforms permission domain data into UI-ready view models.
type PermissionPresenter struct{}

// NewPermissionPresenter creates a new permission presenter.
func NewPermissionPresenter() *PermissionPresenter {
	return &PermissionPresenter{}
}

// Collection constructor methods with business logic

func (p *PermissionPresenter) NewAssignmentCollection(assignments []Assignment) AssignmentCollection {
	hasLimitedAccess := false
	hasSharingLinks := false
	hasSiteGroups := false

	for _, assignment := range assignments {
		if assignment.RoleName == "Limited Access" || assignment.RoleName == "Web-Only Limited Access" {
			hasLimitedAccess = true
		}
		if len(assignment.LoginName) > 12 && assignment.LoginName[:12] == "SharingLinks" {
			hasSharingLinks = true
		}
		// Detect site groups: groups with common site group names that have direct (non-inherited) permissions
		if !assignment.Inherited && (containsSiteGroupPattern(assignment.PrincipalTitle) || containsSiteGroupPattern(assignment.LoginName)) {
			hasSiteGroups = true
		}
	}

	return AssignmentCollection{
		Assignments:      assignments,
		HasLimitedAccess: hasLimitedAccess,
		HasSharingLinks:  hasSharingLinks,
		HasSiteGroups:    hasSiteGroups,
	}
}

func (p *PermissionPresenter) NewExpandableAssignmentCollection(assignments []ExpandableAssignment) ExpandableAssignmentCollection {
	hasLimitedAccess := false
	hasSharingLinks := false
	hasSiteGroups := false

	for _, assignment := range assignments {
		if assignment.RoleName == "Limited Access" || assignment.RoleName == "Web-Only Limited Access" {
			hasLimitedAccess = true
		}
		if len(assignment.LoginName) > 12 && assignment.LoginName[:12] == "SharingLinks" {
			hasSharingLinks = true
		}
		// Detect site groups: groups with common site group names that have direct (non-inherited) permissions
		if !assignment.Inherited && (containsSiteGroupPattern(assignment.PrincipalTitle) || containsSiteGroupPattern(assignment.LoginName)) {
			hasSiteGroups = true
		}
	}

	return ExpandableAssignmentCollection{
		Assignments:      assignments,
		HasLimitedAccess: hasLimitedAccess,
		HasSharingLinks:  hasSharingLinks,
		HasSiteGroups:    hasSiteGroups,
	}
}

// ToListAnalyticsViewModel converts permission analysis business data to view model.
func (p *PermissionPresenter) ToListAnalyticsViewModel(data *application.PermissionAnalysisData, list ListSummary) ListAnalytics {
	return ListAnalytics{
		List:                   list,
		TotalAssignments:       data.TotalAssignments,
		UniqueAssignments:      data.UniqueAssignments,
		InheritedAssignments:   data.InheritedAssignments,
		ItemLevelAssignments:   data.ItemLevelAssignments,
		UserCount:              data.UserCount,
		GroupCount:             data.GroupCount,
		SharingLinkCount:       data.SharingLinkCount,
		SharingLinkUsers:       data.SharingLinkUsers,
		FlexibleLinksCount:     data.FlexibleLinksCount,
		OrganizationViewCount:  data.OrganizationViewCount,
		OrganizationEditCount:  data.OrganizationEditCount,
		AnonymousViewCount:     data.AnonymousViewCount,
		AnonymousEditCount:     data.AnonymousEditCount,
		DirectLinksCount:       data.DirectLinksCount,
		OtherLinksCount:        data.OtherLinksCount,
		TotalItems:             data.TotalItems,
		ItemsWithUnique:        data.ItemsWithUnique,
		FilesCount:             data.FilesCount,
		FoldersCount:           data.FoldersCount,
		FullControlCount:       data.FullControlCount,
		ContributeCount:        data.ContributeCount,
		ReadCount:              data.ReadCount,
		LimitedAccessCount:     data.LimitedAccessCount,
		OtherRolesCount:        data.OtherRolesCount,
		PermissionRiskLevel:    data.PermissionRiskLevel,
		PermissionRiskScore:    data.PermissionRiskScore,
		RiskFromUniqueItems:    data.RiskFromUniqueItems,
		RiskFromAssignments:    data.RiskFromAssignments,
		RiskFromSharingLinks:   data.RiskFromSharingLinks,
		RiskFromElevatedAccess: data.RiskFromElevatedAccess,
	}
}

// Domain model to view model mapping functions.

func (p *PermissionPresenter) MapListToViewModel(list *sharepoint.List) ListSummary {
	var auditRunID int64
	if list.AuditRunID != nil {
		auditRunID = *list.AuditRunID
	}
	
	return ListSummary{
		SiteID:     list.SiteID,
		ListID:     list.ID,
		WebID:      list.WebID,
		Title:      list.Title,
		URL:        list.URL,
		ItemCount:  int64(list.ItemCount),
		HasUnique:  list.HasUnique,
		WebTitle:   list.Title,
		AuditRunID: auditRunID,
	}
}

func (p *PermissionPresenter) MapItemToViewModel(item *sharepoint.Item) ItemSummary {
	return ItemSummary{
		SiteID:    item.SiteID,
		ItemGUID:  item.GUID,
		ListID:    item.ListID,
		ItemID:    int64(item.ID),
		URL:       item.URL,
		IsFile:    item.IsFile,
		IsFolder:  item.IsFolder,
		HasUnique: item.HasUnique,
		Name:      item.Name,
	}
}

func (p *PermissionPresenter) MapAssignmentToViewModel(assignment *sharepoint.Assignment) Assignment {
	return Assignment{
		PrincipalTitle: assignment.Principal.GetDisplayName(),
		LoginName:      assignment.Principal.LoginName,
		PrincipalType:  int32(assignment.Principal.PrincipalType),
		RoleName:       assignment.RoleDefinition.Name,
		Inherited:      assignment.IsInherited(),
	}
}

func (p *PermissionPresenter) MapResolvedAssignmentToViewModel(resolved *sharepoint.ResolvedAssignment) ResolvedAssignment {
	// Convert domain RootCause to view model RootCauseVM
	rootCausesVM := make([]RootCauseVM, len(resolved.RootCauses))
	for i, rootCause := range resolved.RootCauses {
		rootCausesVM[i] = RootCauseVM{
			Type:         rootCause.Type,
			Detail:       rootCause.Detail,
			SourceObject: rootCause.SourceObject,
			SourceRole:   rootCause.SourceRole,
		}
	}

	return ResolvedAssignment{
		Assignment: p.MapAssignmentToViewModel(resolved.Assignment),
		RootCauses: rootCausesVM,
	}
}

func (p *PermissionPresenter) MapPrincipalToSharingLinkMemberViewModel(principal *sharepoint.Principal) SharingLinkMember {
	return SharingLinkMember{
		SiteID:        principal.SiteID,
		PrincipalID:   principal.ID,
		Title:         principal.Title,
		LoginName:     principal.LoginName,
		Email:         principal.Email,
		PrincipalType: principal.PrincipalType,
		IsGroup:       principal.IsGroup() || principal.IsSharePointGroup(),
	}
}

// ToAssignmentToggleHTML generates expandable HTML for assignment root cause details.
// TODO: Move to template-based rendering for HTML generation
func (p *PermissionPresenter) ToAssignmentToggleHTML(resolved *sharepoint.ResolvedAssignment, uniqueID string, isExpanded bool) string {
	if isExpanded {
		// Generate basic expanded content - detailed root cause analysis handled by templates
		if resolved != nil && len(resolved.RootCauses) > 0 {
			// Generate simplified HTML for multiple root causes
			var content strings.Builder
			content.WriteString(`<div class="space-y-3">`)

			for i, rootCause := range resolved.RootCauses {
				var icon, title, detail string

				switch rootCause.Type {
				case "SHARING_LINK":
					icon = `<span class="text-amber-600">🔗</span>`
					title = "Sharing Link Permission"
					detail = rootCause.Detail
				case "SAME_WEB_INHERITANCE":
					icon = `<span class="text-green-600">🏠</span>`
					title = "Web-Level Permission"
					detail = fmt.Sprintf("Has %s on %s", rootCause.SourceRole, rootCause.SourceObject)
				case "SYSTEM_GROUP":
					icon = `<span class="text-blue-600">⚙️</span>`
					title = "System Group Membership"
					detail = rootCause.Detail
				default:
					icon = `<span class="text-gray-600">❓</span>`
					title = "Unknown Source"
					detail = "Root cause could not be determined"
				}

				content.WriteString(`<div class="flex items-start gap-3">`)
				content.WriteString(icon)
				content.WriteString(`<div class="flex-1">`)

				if len(resolved.RootCauses) > 1 {
					content.WriteString(fmt.Sprintf(`<div class="text-xs text-slate-500 mb-1">Source %d</div>`, i+1))
				}

				content.WriteString(fmt.Sprintf(`<div class="font-medium text-slate-900">%s</div>`, title))
				content.WriteString(fmt.Sprintf(`<div class="text-sm text-slate-700">%s</div>`, detail))
				content.WriteString(`</div></div>`)
			}

			content.WriteString(`</div>`)

			return fmt.Sprintf(`<tr id="expand-row-%s" data-state="expanded" class="bg-slate-50">
				<td colspan="6" class="px-3 py-2 border-t">
					<input type="hidden" name="state" value="expanded">
					<div class="slide-down">%s</div>
				</td>
			</tr>`, uniqueID, content.String())
		}

		// Fallback for no root causes
		return fmt.Sprintf(`<tr id="expand-row-%s" data-state="expanded" class="bg-slate-50">
			<td colspan="6" class="px-3 py-2 border-t">
				<input type="hidden" name="state" value="expanded">
				<div class="text-sm text-slate-600">No root cause information available</div>
			</td>
		</tr>`, uniqueID)
	} else {
		// Generate collapsed (hidden) content HTML placeholder
		return fmt.Sprintf(`<tr id="expand-row-%s" data-state="hidden" style="display: none;" class="bg-slate-50">
			<td colspan="6" class="px-3 py-2 border-t">
				<input type="hidden" name="state" value="hidden">
			</td>
		</tr>`, uniqueID)
	}
}

// ToExpandableAssignmentCollection converts resolved assignments to expandable assignment collection.
func (p *PermissionPresenter) ToExpandableAssignmentCollection(resolvedAssignments []*sharepoint.ResolvedAssignment, listID string) ExpandableAssignmentCollection {
	vm := make([]ExpandableAssignment, len(resolvedAssignments))
	for i, resolved := range resolvedAssignments {
		baseAssignment := p.MapAssignmentToViewModel(resolved.Assignment)
		resolvedVM := p.MapResolvedAssignmentToViewModel(resolved)
		vm[i] = ExpandableAssignment{
			Assignment:    baseAssignment,
			RootCauses:    resolvedVM.RootCauses,
			HasRootCauses: len(resolvedVM.RootCauses) > 0,
			UniqueID:      fmt.Sprintf("assignment-%s-%d", listID, i),
		}
	}

	return p.NewExpandableAssignmentCollection(vm)
}

// MapSharingLinkWithItemDataToViewModel converts domain model to view model for UI display.
func (p *PermissionPresenter) MapSharingLinkWithItemDataToViewModel(linkData *sharepoint.SharingLinkWithItemData) SharingLink {
	link := linkData.SharingLink

	// Format created date
	var createdAt string
	if link.CreatedAt != nil {
		createdAt = link.CreatedAt.Format("2006-01-02 15:04")
	}

	// Get created by title
	var createdByTitle string
	if link.CreatedBy != nil {
		createdByTitle = link.CreatedBy.Title
	}

	return SharingLink{
		SiteID:             link.SiteID,
		LinkID:             link.ID,
		ItemGUID:           link.ItemGUID,
		ItemName:           linkData.ItemName,
		ItemURL:            "", // Not available in current domain model
		IsFile:             linkData.ItemIsFile,
		IsFolder:           linkData.ItemIsFolder,
		URL:                link.URL,
		LinkKindName:       link.GetLinkKindName(),
		ScopeName:          link.GetScopeName(),
		IsEditLink:         link.IsEditLink,
		IsReviewLink:       link.IsReviewLink,
		IsDefault:          link.IsDefault,
		IsActive:           link.IsActive,
		CreatedAt:          createdAt,
		CreatedByTitle:     createdByTitle,
		ActualMembersCount: int64(link.TotalMembersCount),
	}
}

// containsSiteGroupPattern checks if a principal name indicates a SharePoint site group
// that was likely re-added after breaking inheritance
func containsSiteGroupPattern(name string) bool {
	// Common patterns for site groups (case-insensitive)
	lowerName := strings.ToLower(name)

	// Standard SharePoint site groups
	siteGroupSuffixes := []string{
		" members", " owners", " visitors",
		" member", " owner", " visitor",
	}

	// Check for exact matches or patterns like "Site Name Members"
	for _, suffix := range siteGroupSuffixes {
		if strings.HasSuffix(lowerName, suffix) {
			return true
		}
	}

	// Check for common prefixes that indicate site groups (like GRP-, Team-, etc.)
	siteGroupPrefixes := []string{
		"grp-", "team-", "proj-", "project-",
	}

	for _, prefix := range siteGroupPrefixes {
		if strings.HasPrefix(lowerName, prefix) {
			// Further check if it has typical site group suffixes
			for _, suffix := range siteGroupSuffixes {
				if strings.HasSuffix(lowerName, suffix) {
					return true
				}
			}
		}
	}

	return false
}
