package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"spaudit/application"
	"spaudit/interfaces/web/presenters"
	"spaudit/logging"
)

// AuditHandlers handles HTTP requests for audit operations.
type AuditHandlers struct {
	auditService   application.AuditService
	auditPresenter *presenters.AuditPresenter
	sseManager     *SSEManager
	logger         *logging.Logger
}

// NewAuditHandlers creates a new audit handlers instance.
func NewAuditHandlers(
	auditService application.AuditService,
	auditPresenter *presenters.AuditPresenter,
	sseManager *SSEManager,
) *AuditHandlers {
	return &AuditHandlers{
		auditService:   auditService,
		auditPresenter: auditPresenter,
		sseManager:     sseManager,
		logger:         logging.Default().WithComponent("audit_handler"),
	}
}

// GetAuditStatus retrieves audit status for a site.
// GET /audit/status?site_url={siteURL}
func (h *AuditHandlers) GetAuditStatus(w http.ResponseWriter, r *http.Request) {
	siteURL := r.URL.Query().Get("site_url")
	if siteURL == "" {
		http.Error(w, "missing site_url parameter", http.StatusBadRequest)
		return
	}

	// Use application service to get audit status
	audit, exists := h.auditService.GetAuditStatus(siteURL)

	w.Header().Set("Content-Type", "application/json")

	var auditView interface{}
	if !exists {
		auditView = h.auditPresenter.FormatAuditNotFound()
	} else {
		auditView = h.auditPresenter.FormatAuditStatus(audit)
	}

	if err := json.NewEncoder(w).Encode(auditView); err != nil {
		h.logger.Error("Failed to encode audit status response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ListActiveAudits returns all currently running audits.
// GET /audit/active
func (h *AuditHandlers) ListActiveAudits(w http.ResponseWriter, r *http.Request) {
	// Use application service to get active audits
	audits := h.auditService.GetActiveAudits()

	// Use presenter to format for display
	activeAuditsView := h.auditPresenter.FormatActiveAudits(audits)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(activeAuditsView); err != nil {
		h.logger.Error("Failed to encode active audits response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// RunAudit queues a new audit request.
// POST /audit
func (h *AuditHandlers) RunAudit(w http.ResponseWriter, r *http.Request) {
	siteURL := r.FormValue("site_url")

	if siteURL == "" {
		h.logger.Error("Missing site_url parameter in audit request")
		errorResponse := h.auditPresenter.FormatAuditErrorResponse(fmt.Errorf("site URL is required"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(errorResponse))
		return
	}

	// Parse form into structured data
	if err := r.ParseForm(); err != nil {
		h.logger.Error("Failed to parse form data", "error", err)
		errorResponse := h.auditPresenter.FormatAuditErrorResponse(fmt.Errorf("invalid form data: %v", err))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(errorResponse))
		return
	}

	// Use application service to build parameters from form data
	parameters := h.auditService.BuildAuditParametersFromFormData(r.Form)

	// Queue the audit through the application service
	request, err := h.auditService.QueueAudit(r.Context(), siteURL, parameters)
	if err != nil {
		h.logger.Error("Failed to queue audit", "site_url", siteURL, "error", err)

		// Return formatted HTML error message for HTMX (using 200 OK so HTMX always swaps)
		var errorResponse string
		if strings.Contains(err.Error(), "already running") || strings.Contains(err.Error(), "already queued") {
			errorResponse = h.auditPresenter.FormatAuditConflictResponse(err)
		} else {
			errorResponse = h.auditPresenter.FormatAuditErrorResponse(err)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(errorResponse))
		return
	}

	h.logger.Info("Audit queued successfully",
		"request_id", request.ID,
		"site_url", siteURL)

	// Broadcast job list update to all SSE clients
	h.sseManager.BroadcastJobListUpdate()

	// Use presenter to format success response
	response := h.auditPresenter.FormatAuditQueuedResponse(request)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(response))
}
