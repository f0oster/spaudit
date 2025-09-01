// Package handlers contains HTTP request handlers for the web interface.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"spaudit/domain/contracts"

	"spaudit/application"
	"spaudit/interfaces/web/presenters"
	"spaudit/interfaces/web/templates/pages"
)

// ListHandlers handles HTTP requests for SharePoint list operations.
type ListHandlers struct {
	// Application services (web UI orchestration)
	siteContentService  *application.SiteContentService
	permissionService   *application.PermissionService
	siteBrowsingService *application.SiteBrowsingService
	jobService          application.JobService
	auditService        application.AuditService

	// Presenters (view logic)
	listPresenter       *presenters.ListPresenter
	permissionPresenter *presenters.PermissionPresenter
	sitePresenter       *presenters.SitePresenter
	
	// Service factory for creating audit-run-scoped services
	serviceFactory      application.AuditRunScopedServiceFactory
}

// NewListHandlers creates a new list handlers instance.
func NewListHandlers(
	siteContentService *application.SiteContentService,
	permissionService *application.PermissionService,
	siteBrowsingService *application.SiteBrowsingService,
	jobService application.JobService,
	auditService application.AuditService,
	listPresenter *presenters.ListPresenter,
	permissionPresenter *presenters.PermissionPresenter,
	sitePresenter *presenters.SitePresenter,
	serviceFactory application.AuditRunScopedServiceFactory,
) *ListHandlers {
	return &ListHandlers{
		siteContentService:  siteContentService,
		permissionService:   permissionService,
		siteBrowsingService: siteBrowsingService,
		jobService:          jobService,
		auditService:        auditService,
		listPresenter:       listPresenter,
		permissionPresenter: permissionPresenter,
		sitePresenter:       sitePresenter,
		serviceFactory:      serviceFactory,
	}
}

// Home renders the main dashboard/site selection page.
// GET /
func (h *ListHandlers) Home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get business data from services
	allJobs := h.jobService.ListAllJobs()
	
	// Get sites with their latest audit run metadata instead of aggregated data
	sitesData, err := h.getSitesWithLatestAuditRunMetadata(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform to view model using presenter
	siteSelectionVM := h.sitePresenter.ToSiteSelectionViewModel(sitesData, len(allJobs) > 0)

	// Render response
	RenderResponse(ctx, w, r, pages.SiteSelectionPage(*siteSelectionVM))
}

// SiteListsPage renders the lists page for a site and audit run.
// GET /sites/{siteID}/audit-runs/{auditRunID}/lists
func (h *ListHandlers) SiteListsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract and validate parameters
	siteID, err := h.extractSiteID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	auditRunIDStr, err := h.extractAuditRunID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create audit-run-scoped services
	scopedServices, err := h.serviceFactory.CreateForAuditRun(ctx, siteID, auditRunIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create audit-run-scoped services: %v", err), http.StatusInternalServerError)
		return
	}

	// Get business data from audit-run-scoped service
	data, err := scopedServices.SiteContentService.GetSiteWithLists(ctx, siteID)
	if err != nil {
		http.Error(w, "Site not found", http.StatusNotFound)
		return
	}

	// Add audit run information to context for template
	data.AuditRunID = scopedServices.AuditRunID

	// Convert to view model using presenter
	viewModel := h.listPresenter.ToSiteListsViewModel(data)

	// Fetch audit runs for selector using audit service
	auditRunsData, err := h.auditService.GetAuditRunsForSite(ctx, siteID, 50)
	if err != nil {
		// Log error but don't fail the request - selector just won't be populated
		// TODO: Add proper logging
	} else {
		// Convert to view model format
		auditRuns := make([]presenters.AuditRunOption, len(auditRunsData))
		for i, auditRun := range auditRunsData {
			auditRuns[i] = presenters.AuditRunOption{
				ID:        auditRun.ID,
				StartedAt: auditRun.StartedAt,
				Status:    auditRun.GetStatus(),
			}
		}
		viewModel.AuditRuns = auditRuns
	}

	// Render response
	RenderResponse(ctx, w, r, pages.SiteListsPage(*viewModel))
}

// ListDetail renders the detailed view for a specific list.
// GET /sites/{siteID}/audit-runs/{auditRunID}/lists/{listID}
func (h *ListHandlers) ListDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	siteID, listID, err := h.extractSiteAndListID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	auditRunIDStr, err := h.extractAuditRunID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create audit-run-scoped services
	scopedServices, err := h.serviceFactory.CreateForAuditRun(ctx, siteID, auditRunIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create audit-run-scoped services: %v", err), http.StatusInternalServerError)
		return
	}

	// Get list data from audit-run-scoped service
	listData, err := scopedServices.SiteContentService.GetListByID(ctx, siteID, listID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Analyze permissions using audit-run-scoped service
	analyticsData, err := scopedServices.PermissionService.AnalyzeListPermissions(ctx, siteID, listData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform to view models using presenters
	vmList := h.permissionPresenter.MapListToViewModel(listData)
	analytics := h.permissionPresenter.ToListAnalyticsViewModel(analyticsData, vmList)

	// Render response (default tab: overview)
	RenderResponse(ctx, w, r, pages.ListShell(vmList, "overview", pages.ListOverviewTab(analytics)))
}

// OverviewTab renders the overview tab content for a list (HTMX partial).
func (h *ListHandlers) OverviewTab(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	siteID, listID, err := h.extractSiteAndListID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	auditRunIDStr, err := h.extractAuditRunID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create audit-run-scoped services
	scopedServices, err := h.serviceFactory.CreateForAuditRun(ctx, siteID, auditRunIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create audit-run-scoped services: %v", err), http.StatusInternalServerError)
		return
	}

	// Get list data from audit-run-scoped service
	listData, err := scopedServices.SiteContentService.GetListByID(ctx, siteID, listID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Analyze permissions using audit-run-scoped service
	analyticsData, err := scopedServices.PermissionService.AnalyzeListPermissions(ctx, siteID, listData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform to view models using presenters
	vmList := h.permissionPresenter.MapListToViewModel(listData)
	analytics := h.permissionPresenter.ToListAnalyticsViewModel(analyticsData, vmList)

	// Check if this is an HTMX partial request or direct navigation
	if IsHTMXPartialRequest(r) {
		RenderResponse(ctx, w, r, pages.TabsAndContent(siteID, scopedServices.AuditRunID, listID, "overview", pages.ListOverviewTab(analytics)))
	} else {
		// Direct navigation - render full page
		RenderResponse(ctx, w, r, pages.ListShell(vmList, "overview", pages.ListOverviewTab(analytics)))
	}
}

// AssignmentsTab shows the assignments tab for a list
func (h *ListHandlers) AssignmentsTab(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	siteID, listID, err := h.extractSiteAndListID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	auditRunIDStr, err := h.extractAuditRunID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create audit-run-scoped services
	scopedServices, err := h.serviceFactory.CreateForAuditRun(ctx, siteID, auditRunIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create audit-run-scoped services: %v", err), http.StatusInternalServerError)
		return
	}

	// Get business data from audit-run-scoped service (assignments with root cause analysis)
	assignmentsData, err := scopedServices.SiteContentService.GetListAssignmentsWithRootCause(ctx, siteID, listID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform to view model using presenter
	assignmentCollection := h.permissionPresenter.ToExpandableAssignmentCollection(assignmentsData, listID)

	if IsHTMXPartialRequest(r) {
		RenderResponse(ctx, w, r, pages.TabsAndContent(siteID, scopedServices.AuditRunID, listID, "assignments", pages.ListAssignmentsTab(siteID, scopedServices.AuditRunID, assignmentCollection)))
	} else {
		// Direct navigation - need list data for full page
		listData, err := scopedServices.SiteContentService.GetListByID(ctx, siteID, listID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		vmList := h.permissionPresenter.MapListToViewModel(listData)
		RenderResponse(ctx, w, r, pages.ListShell(vmList, "assignments", pages.ListAssignmentsTab(siteID, scopedServices.AuditRunID, assignmentCollection)))
	}
}

// ItemsTab shows the items tab for a list
func (h *ListHandlers) ItemsTab(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	siteID, listID, err := h.extractSiteAndListID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	auditRunIDStr, err := h.extractAuditRunID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create audit-run-scoped services
	scopedServices, err := h.serviceFactory.CreateForAuditRun(ctx, siteID, auditRunIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create audit-run-scoped services: %v", err), http.StatusInternalServerError)
		return
	}

	// TODO: Implement proper pagination for items tab
	// - Extract page/limit from query parameters (e.g., ?page=1&limit=50)
	// - Add pagination metadata to response (total count, current page, has next/prev)
	// - Update UI to show pagination controls and handle HTMX partial updates
	// - Consider default limits: 50 for UI responsiveness, max 500 for performance
	// - Add loading states for large datasets

	// TEMPORARY: Using high static limit - replace with pagination
	itemsData, err := scopedServices.SiteContentService.GetListItems(ctx, siteID, listID, 0, 1000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform to view models using presenter
	items := make([]presenters.ItemSummary, len(itemsData))
	for i, item := range itemsData {
		items[i] = h.permissionPresenter.MapItemToViewModel(item)
	}

	if IsHTMXPartialRequest(r) {
		// Get list data for the tab component
		listData, err := scopedServices.SiteContentService.GetListByID(ctx, siteID, listID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		vmList := h.permissionPresenter.MapListToViewModel(listData)
		RenderResponse(ctx, w, r, pages.TabsAndContent(siteID, scopedServices.AuditRunID, listID, "items", pages.ListItemsTab(vmList, scopedServices.AuditRunID, items)))
	} else {
		// Direct navigation - need list data for full page
		listData, err := scopedServices.SiteContentService.GetListByID(ctx, siteID, listID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		vmList := h.permissionPresenter.MapListToViewModel(listData)
		RenderResponse(ctx, w, r, pages.ListShell(vmList, "items", pages.ListItemsTab(vmList, scopedServices.AuditRunID, items)))
	}
}

// LinksTab shows the sharing links tab for a list
func (h *ListHandlers) LinksTab(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	siteID, listID, err := h.extractSiteAndListID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	auditRunIDStr, err := h.extractAuditRunID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create audit-run-scoped services
	scopedServices, err := h.serviceFactory.CreateForAuditRun(ctx, siteID, auditRunIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create audit-run-scoped services: %v", err), http.StatusInternalServerError)
		return
	}

	// Get data with item details from audit-run-scoped service
	linkData, err := scopedServices.SiteContentService.GetListSharingLinksWithItemData(ctx, siteID, listID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform to view models using presenter
	linkVMs := make([]presenters.SharingLink, len(linkData))
	for i, linkWithItem := range linkData {
		linkVMs[i] = h.permissionPresenter.MapSharingLinkWithItemDataToViewModel(linkWithItem)
	}

	if IsHTMXPartialRequest(r) {
		RenderResponse(ctx, w, r, pages.TabsAndContent(siteID, scopedServices.AuditRunID, listID, "links", pages.ListLinksTab(linkVMs, scopedServices.AuditRunID)))
	} else {
		// Direct navigation - need list data for full page
		listData, err := scopedServices.SiteContentService.GetListByID(ctx, siteID, listID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		vmList := h.permissionPresenter.MapListToViewModel(listData)
		RenderResponse(ctx, w, r, pages.ListShell(vmList, "links", pages.ListLinksTab(linkVMs, scopedServices.AuditRunID)))
	}
}

// ToggleAssignment handles HTMX assignment toggle requests
func (h *ListHandlers) ToggleAssignment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	uniqueID := chi.URLParam(r, "uniqueID")
	siteID, err := h.extractSiteID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	auditRunIDStr, err := h.extractAuditRunID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create audit-run-scoped services
	scopedServices, err := h.serviceFactory.CreateForAuditRun(ctx, siteID, auditRunIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create audit-run-scoped services: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse unique ID to get list ID and index
	listID, index, err := h.parseAssignmentUniqueID(uniqueID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Determine current state and desired action
	currentState := r.FormValue("state")
	isCurrentlyHidden := currentState == "hidden" || currentState == ""

	if isCurrentlyHidden {
		// Expand - get business data and generate expanded HTML
		assignmentsData, err := scopedServices.SiteContentService.GetListAssignmentsWithRootCause(ctx, siteID, listID)
		if err != nil || index >= len(assignmentsData) {
			http.Error(w, "Assignment not found", http.StatusNotFound)
			return
		}

		// Use presenter to generate HTML
		htmlContent := h.permissionPresenter.ToAssignmentToggleHTML(assignmentsData[index], uniqueID, true)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))
	} else {
		// Collapse - generate collapsed HTML using presenter
		htmlContent := h.permissionPresenter.ToAssignmentToggleHTML(nil, uniqueID, false)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))
	}
}

// SearchLists handles HTMX search requests for filtering lists
func (h *ListHandlers) SearchLists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	siteID, err := h.extractSiteID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	auditRunIDStr, err := h.extractAuditRunID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create audit-run-scoped services
	scopedServices, err := h.serviceFactory.CreateForAuditRun(ctx, siteID, auditRunIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create audit-run-scoped services: %v", err), http.StatusInternalServerError)
		return
	}

	searchQuery := h.extractSearchQuery(r)

	// Get business data from audit-run-scoped service
	listsData, err := scopedServices.SiteContentService.GetListsForSite(ctx, siteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to view models and apply search filter using presenter
	listVMs := h.listPresenter.ToListSummaries(listsData)
	filteredLists := h.listPresenter.FilterListsForSearch(listVMs, searchQuery)

	// Return just the table body rows
	RenderResponse(ctx, w, r, pages.ListTableRows(filteredLists, siteID, scopedServices.AuditRunID))
}

// SearchSites handles HTMX search requests for filtering sites
func (h *ListHandlers) SearchSites(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	searchQuery := h.extractSearchQuery(r)

	// Get business data from service
	sitesData, err := h.siteBrowsingService.SearchSites(ctx, searchQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform to view models using presenter
	siteVMs := h.sitePresenter.ToSitesWithMetadata(sitesData)

	// Return just the table body rows
	RenderResponse(ctx, w, r, pages.SiteTableRows(siteVMs))
}

// SitesTable handles full sites table requests with search preservation
func (h *ListHandlers) SitesTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	searchQuery := h.extractSearchQuery(r)

	// Get sites with their latest audit run metadata instead of aggregated data
	var sitesData []*contracts.SiteWithMetadata
	var err error
	
	if searchQuery == "" {
		// No search query - get all sites with latest audit run metadata
		sitesData, err = h.getSitesWithLatestAuditRunMetadata(ctx)
	} else {
		// Search query provided - get all sites first, then filter
		allSitesData, err := h.getSitesWithLatestAuditRunMetadata(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		// Filter sites based on search query (simple contains search)
		for _, siteData := range allSitesData {
			if strings.Contains(strings.ToLower(siteData.Site.Title), strings.ToLower(searchQuery)) ||
				strings.Contains(strings.ToLower(siteData.Site.URL), strings.ToLower(searchQuery)) {
				sitesData = append(sitesData, siteData)
			}
		}
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform to view models using presenter
	siteSelectionVM := h.sitePresenter.ToSiteSelectionViewModel(sitesData, false)
	RenderResponse(ctx, w, r, pages.SitesTableInner(*siteSelectionVM))
}

// Helper methods for parameter extraction and validation

func (h *ListHandlers) extractSiteID(r *http.Request) (int64, error) {
	siteIDParam := chi.URLParam(r, "siteID")
	if siteIDParam == "" {
		return 0, fmt.Errorf("siteID parameter is required")
	}

	siteID, err := strconv.ParseInt(siteIDParam, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid siteID parameter: %w", err)
	}

	return siteID, nil
}

// GetAuditRunsForSite returns audit runs for a site as JSON
func (h *ListHandlers) GetAuditRunsForSite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract site ID
	siteIDStr := chi.URLParam(r, "siteID")
	siteID, err := strconv.ParseInt(siteIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid site ID", http.StatusBadRequest)
		return
	}

	// Get audit runs for this site using audit service
	auditRunsData, err := h.auditService.GetAuditRunsForSite(ctx, siteID, 50)
	if err != nil {
		http.Error(w, "Failed to get audit runs", http.StatusInternalServerError)
		return
	}

	// Convert to JSON response format
	type AuditRunResponse struct {
		ID        int64  `json:"id"`
		StartedAt string `json:"started_at"`
		Status    string `json:"status"`
	}

	auditRuns := make([]AuditRunResponse, len(auditRunsData))
	for i, auditRun := range auditRunsData {
		auditRuns[i] = AuditRunResponse{
			ID:        auditRun.ID,
			StartedAt: auditRun.StartedAt.Format("2006-01-02 15:04:05"),
			Status:    auditRun.GetStatus(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(auditRuns); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// extractAuditRunID extracts audit run ID from URL parameters
// Returns "latest" as special case or parses the numeric ID
func (h *ListHandlers) extractAuditRunID(r *http.Request) (string, error) {
	auditRunIDParam := chi.URLParam(r, "auditRunID")
	if auditRunIDParam == "" {
		// Default to "latest" if not specified
		return "latest", nil
	}
	
	// Allow "latest" as special case
	if auditRunIDParam == "latest" {
		return "latest", nil
	}
	
	// Validate that it's a valid number if not "latest"
	if _, err := strconv.ParseInt(auditRunIDParam, 10, 64); err != nil {
		return "", fmt.Errorf("invalid auditRunID parameter: %w", err)
	}
	
	return auditRunIDParam, nil
}

func (h *ListHandlers) extractSiteAndListID(r *http.Request) (int64, string, error) {
	siteID, err := h.extractSiteID(r)
	if err != nil {
		return 0, "", err
	}

	listID := chi.URLParam(r, "listID")
	if listID == "" {
		return 0, "", fmt.Errorf("listID parameter is required")
	}

	return siteID, listID, nil
}

func (h *ListHandlers) extractSearchQuery(r *http.Request) string {
	// Try both query parameter and form value for flexibility
	searchQuery := strings.TrimSpace(r.FormValue("search"))
	if searchQuery == "" {
		searchQuery = strings.TrimSpace(r.URL.Query().Get("search"))
	}
	return searchQuery
}

func (h *ListHandlers) parseAssignmentUniqueID(uniqueID string) (string, int, error) {
	// Parse the unique ID format: assignment-{listID}-{index}
	if !strings.HasPrefix(uniqueID, "assignment-") {
		return "", 0, fmt.Errorf("invalid unique ID format")
	}

	parts := strings.Split(uniqueID[len("assignment-"):], "-")
	if len(parts) < 2 {
		return "", 0, fmt.Errorf("invalid unique ID format")
	}

	// The index is the last part, everything else is listID
	indexStr := parts[len(parts)-1]
	listID := strings.Join(parts[:len(parts)-1], "-")

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid unique ID format: %w", err)
	}

	return listID, index, nil
}

// Helper methods for combining business logic calls


// GetObjectAssignments handles GET requests for object assignments (HTMX partial)
func (h *ListHandlers) GetObjectAssignments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	siteID, err := h.extractSiteID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	auditRunIDStr, err := h.extractAuditRunID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create audit-run-scoped services
	scopedServices, err := h.serviceFactory.CreateForAuditRun(ctx, siteID, auditRunIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create audit-run-scoped services: %v", err), http.StatusInternalServerError)
		return
	}

	objectType := chi.URLParam(r, "otype")
	objectKey := chi.URLParam(r, "okey")

	// Get business data from audit-run-scoped service
	assignments, err := scopedServices.SiteContentService.GetAssignmentsForObject(ctx, siteID, objectType, objectKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform to view models using presenter
	vm := make([]presenters.Assignment, len(assignments))
	for i, assignment := range assignments {
		vm[i] = h.permissionPresenter.MapAssignmentToViewModel(assignment)
	}

	assignmentCollection := h.permissionPresenter.NewAssignmentCollection(vm)

	// Render response
	RenderResponse(ctx, w, r, pages.AssignmentsList(assignmentCollection))
}

// GetSharingLinkMembers handles GET requests for sharing link members (HTMX partial)
func (h *ListHandlers) GetSharingLinkMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	siteID, err := h.extractSiteID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	auditRunIDStr, err := h.extractAuditRunID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create audit-run-scoped services
	scopedServices, err := h.serviceFactory.CreateForAuditRun(ctx, siteID, auditRunIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create audit-run-scoped services: %v", err), http.StatusInternalServerError)
		return
	}

	linkID := chi.URLParam(r, "linkID")
	if linkID == "" {
		http.Error(w, "invalid link ID", http.StatusBadRequest)
		return
	}

	// Get business data from audit-run-scoped service
	principals, err := scopedServices.SiteContentService.GetSharingLinkMembers(ctx, siteID, linkID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform to view models using presenter
	vm := make([]presenters.SharingLinkMember, len(principals))
	for i, principal := range principals {
		vm[i] = h.permissionPresenter.MapPrincipalToSharingLinkMemberViewModel(principal)
	}

	// Render response
	RenderResponse(ctx, w, r, pages.SharingLinkMembersList(vm))
}

// ToggleSharingLinkMembers handles POST requests for sharing link member visibility toggle
func (h *ListHandlers) ToggleSharingLinkMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	siteID, err := h.extractSiteID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	auditRunIDStr, err := h.extractAuditRunID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create audit-run-scoped services
	scopedServices, err := h.serviceFactory.CreateForAuditRun(ctx, siteID, auditRunIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create audit-run-scoped services: %v", err), http.StatusInternalServerError)
		return
	}

	linkID := chi.URLParam(r, "linkID")
	if linkID == "" {
		http.Error(w, "invalid link ID", http.StatusBadRequest)
		return
	}

	// Get current state and determine action
	currentState := r.FormValue("state")
	isCurrentlyHidden := currentState == "hidden" || currentState == ""

	// Get business data from audit-run-scoped service (always needed for member count)
	principals, err := scopedServices.SiteContentService.GetSharingLinkMembers(ctx, siteID, linkID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	if isCurrentlyHidden {
		// Show members - return expandable row with proper template rendering
		vm := make([]presenters.SharingLinkMember, len(principals))
		for i, principal := range principals {
			vm[i] = h.permissionPresenter.MapPrincipalToSharingLinkMemberViewModel(principal)
		}

		// Return visible expandable row with content
		w.Write([]byte(`<tr id="members-row-` + linkID + `" data-state="visible" class="bg-slate-50" style="display: table-row;">
			<td colspan="8" class="px-3 py-2 border-t">
				<input type="hidden" name="state" value="visible">`))

		RenderResponse(ctx, w, r, pages.SharingLinkMembersList(vm))
		w.Write([]byte(`</td></tr>`))

		// Update button text with OOB swap
		memberCount := len(principals)
		hideText := fmt.Sprintf("Hide %d members", memberCount)
		siteIDStr := strconv.FormatInt(siteID, 10)
		auditRunIDStrFormatted := strconv.FormatInt(scopedServices.AuditRunID, 10)
		endpoint := fmt.Sprintf("/sites/%s/audit-runs/%s/sharing-links/%s/members/toggle", siteIDStr, auditRunIDStrFormatted, linkID)
		w.Write([]byte(`<button id="btn-members-row-` + linkID + `" hx-swap-oob="true" class="text-blue-600 hover:text-blue-700 text-xs font-medium hover:underline" hx-post="` + endpoint + `" hx-target="#members-row-` + linkID + `" hx-swap="outerHTML" hx-include="#members-row-` + linkID + `">` + hideText + `</button>`))
	} else {
		// Hide members - return hidden empty row
		w.Write([]byte(`<tr id="members-row-` + linkID + `" data-state="hidden" style="display: none;" class="bg-slate-50">
			<td colspan="8" class="px-3 py-2 border-t">
				<input type="hidden" name="state" value="hidden">
			</td>
		</tr>`))

		// Update button text with OOB swap
		memberCount := len(principals)
		viewText := fmt.Sprintf("%d members", memberCount)
		siteIDStr := strconv.FormatInt(siteID, 10)
		auditRunIDStrFormatted := strconv.FormatInt(scopedServices.AuditRunID, 10)
		endpoint := fmt.Sprintf("/sites/%s/audit-runs/%s/sharing-links/%s/members/toggle", siteIDStr, auditRunIDStrFormatted, linkID)
		w.Write([]byte(`<button id="btn-members-row-` + linkID + `" hx-swap-oob="true" class="text-blue-600 hover:text-blue-700 text-xs font-medium hover:underline" hx-post="` + endpoint + `" hx-target="#members-row-` + linkID + `" hx-swap="outerHTML" hx-include="#members-row-` + linkID + `">` + viewText + `</button>`))
	}
}

// ToggleItemAssignments handles POST requests for item assignment visibility toggle
func (h *ListHandlers) ToggleItemAssignments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	siteID, err := h.extractSiteID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	auditRunIDStr, err := h.extractAuditRunID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create audit-run-scoped services
	scopedServices, err := h.serviceFactory.CreateForAuditRun(ctx, siteID, auditRunIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create audit-run-scoped services: %v", err), http.StatusInternalServerError)
		return
	}

	itemGUID := chi.URLParam(r, "itemGUID")

	// Get current state and determine action
	currentState := r.FormValue("state")
	isCurrentlyHidden := currentState == "hidden" || currentState == ""

	w.Header().Set("Content-Type", "text/html")

	if isCurrentlyHidden {
		// Show assignments - load and return expandable row with proper template rendering
		assignments, err := scopedServices.SiteContentService.GetAssignmentsForObject(ctx, siteID, "item", itemGUID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		vm := make([]presenters.Assignment, len(assignments))
		for i, assignment := range assignments {
			vm[i] = h.permissionPresenter.MapAssignmentToViewModel(assignment)
		}

		assignmentCollection := h.permissionPresenter.NewAssignmentCollection(vm)

		// Return visible expandable row with content
		w.Write([]byte(`<tr id="assign-row-` + itemGUID + `" data-state="visible" class="bg-slate-50" style="display: table-row;">
			<td colspan="5" class="px-3 py-2 border-t">
				<input type="hidden" name="state" value="visible">`))

		RenderResponse(ctx, w, r, pages.AssignmentsList(assignmentCollection))
		w.Write([]byte(`</td></tr>`))

		// Update button text with OOB swap
		siteIDStr := strconv.FormatInt(siteID, 10)
		endpoint := fmt.Sprintf("/sites/%s/audit-runs/%s/items/%s/assignments/toggle", siteIDStr, auditRunIDStr, itemGUID)
		w.Write([]byte(`<button id="btn-assign-row-` + itemGUID + `" hx-swap-oob="true" class="text-blue-600 hover:text-blue-700 text-xs font-medium hover:underline" hx-post="` + endpoint + `" hx-target="#assign-row-` + itemGUID + `" hx-swap="outerHTML" hx-include="#assign-row-` + itemGUID + `">Hide assignments</button>`))
	} else {
		// Hide assignments - return hidden empty row
		w.Write([]byte(`<tr id="assign-row-` + itemGUID + `" data-state="hidden" style="display: none;" class="bg-slate-50">
			<td colspan="5" class="px-3 py-2 border-t">
				<input type="hidden" name="state" value="hidden">
			</td>
		</tr>`))

		// Update button text with OOB swap
		siteIDStr := strconv.FormatInt(siteID, 10)
		endpoint := fmt.Sprintf("/sites/%s/audit-runs/%s/items/%s/assignments/toggle", siteIDStr, auditRunIDStr, itemGUID)
		w.Write([]byte(`<button id="btn-assign-row-` + itemGUID + `" hx-swap-oob="true" class="text-blue-600 hover:text-blue-700 text-xs font-medium hover:underline" hx-post="` + endpoint + `" hx-target="#assign-row-` + itemGUID + `" hx-swap="outerHTML" hx-include="#assign-row-` + itemGUID + `">Assignments</button>`))
	}
}


// getSitesWithLatestAuditRunMetadata gets all sites with their latest audit run metadata
// instead of aggregated metadata across all audit runs
func (h *ListHandlers) getSitesWithLatestAuditRunMetadata(ctx context.Context) ([]*contracts.SiteWithMetadata, error) {
	// First, get all sites (without metadata aggregation)
	allSitesData, err := h.siteBrowsingService.GetAllSitesWithMetadata(ctx)
	if err != nil {
		return nil, err
	}

	// For each site, get the latest audit run metadata using fully scoped services
	var latestSitesData []*contracts.SiteWithMetadata
	for _, siteData := range allSitesData {
		// Use the scoped service factory to get latest audit run data
		scopedServices, err := h.serviceFactory.CreateForAuditRun(ctx, siteData.Site.ID, "latest")
		if err != nil {
			// If no audit runs exist for this site, skip it or use the original data
			latestSitesData = append(latestSitesData, siteData)
			continue
		}

		// Get site metadata for the latest audit run using fully scoped services
		latestSiteData, err := scopedServices.SiteBrowsingService.GetSiteWithMetadata(ctx, siteData.Site.ID)
		if err != nil {
			// If error getting latest data, fall back to original
			latestSitesData = append(latestSitesData, siteData)
			continue
		}

		latestSitesData = append(latestSitesData, latestSiteData)
	}

	return latestSitesData, nil
}

// SwitchAuditRun handles audit run switching from the selector
func (h *ListHandlers) SwitchAuditRun(w http.ResponseWriter, r *http.Request) {
	siteID := chi.URLParam(r, "siteID")
	
	// Get selected run ID from form value (POST) or query parameter (GET)
	selectedRunID := r.FormValue("audit_run_id")
	if selectedRunID == "" {
		selectedRunID = r.URL.Query().Get("audit-run-selector")
	}
	if selectedRunID == "" {
		selectedRunID = "latest"
	}
	
	// Redirect to the same page but with the new audit run ID
	// For now, redirect to lists page - could be made more sophisticated
	redirectURL := fmt.Sprintf("/sites/%s/audit-runs/%s/lists", siteID, selectedRunID)
	
	w.Header().Set("HX-Redirect", redirectURL)
	w.WriteHeader(http.StatusOK)
}
