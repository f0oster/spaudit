package presenters

import (
	"fmt"
	"time"

	"spaudit/domain/audit"
)

// Audit-related view data structures

// AuditStatusView represents the status of an audit for API responses
type AuditStatusView struct {
	Exists    bool   `json:"exists"`
	RequestID string `json:"request_id,omitempty"`
	SiteURL   string `json:"site_url,omitempty"`
	Status    string `json:"status,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
	Progress  string `json:"progress,omitempty"`
	JobID     string `json:"job_id,omitempty"`
	Message   string `json:"message,omitempty"`
}

// ActiveAuditsView represents a list of active audits
type ActiveAuditsView struct {
	ActiveAudits []*AuditStatusView `json:"active_audits"`
}

// AuditPresenter transforms audit domain data into UI-ready view models and HTML responses.
type AuditPresenter struct{}

// NewAuditPresenter creates a new audit presenter.
func NewAuditPresenter() *AuditPresenter {
	return &AuditPresenter{}
}

// FormatAuditStatus converts an ActiveAudit to a view model with status and progress information.
func (p *AuditPresenter) FormatAuditStatus(audit *audit.ActiveAudit) *AuditStatusView {
	if audit == nil {
		return nil
	}

	return &AuditStatusView{
		Exists:    true,
		RequestID: audit.Request.ID,
		SiteURL:   audit.Request.SiteURL,
		Status:    string(audit.Status),
		StartedAt: audit.StartedAt.Format(time.RFC3339),
		Progress:  audit.Progress.Message,
		JobID:     audit.JobID,
	}
}

// FormatAuditNotFound creates a view model for when no audit is found.
func (p *AuditPresenter) FormatAuditNotFound() *AuditStatusView {
	return &AuditStatusView{
		Exists:  false,
		Message: "No active audit for site",
	}
}

// FormatActiveAudits converts a list of ActiveAudits to view models for display.
func (p *AuditPresenter) FormatActiveAudits(audits []*audit.ActiveAudit) *ActiveAuditsView {
	auditViews := make([]*AuditStatusView, 0, len(audits))

	for _, audit := range audits {
		auditView := p.FormatAuditStatus(audit)
		if auditView != nil {
			auditViews = append(auditViews, auditView)
		}
	}

	return &ActiveAuditsView{
		ActiveAudits: auditViews,
	}
}

// FormatAuditQueuedResponse creates animated success HTML response for queued audit.
func (p *AuditPresenter) FormatAuditQueuedResponse(request *audit.AuditRequest) string {
	return fmt.Sprintf(`<div class="audit-success-message bg-gradient-to-r from-green-50 to-emerald-50 border-l-4 border-green-500 shadow-sm">
		<style>
			.audit-success-message {
				animation: fadeInThenOut 6s ease-in-out forwards;
				overflow: hidden;
				border-radius: 0.5rem;
			}
			@keyframes fadeInThenOut {
				0%% { opacity: 0; transform: translateY(-10px); max-height: 0; margin-bottom: 0; padding-top: 0; padding-bottom: 0; }
				10%% { opacity: 1; transform: translateY(0); max-height: 280px; margin-bottom: 16px; padding-top: 20px; padding-bottom: 20px; padding-left: 16px; padding-right: 16px; }
				80%% { opacity: 1; transform: translateY(0); max-height: 280px; margin-bottom: 16px; padding-top: 20px; padding-bottom: 20px; padding-left: 16px; padding-right: 16px; }
				100%% { opacity: 0; transform: translateY(-10px); max-height: 0; margin-bottom: 0; padding-top: 0; padding-bottom: 0; padding-left: 0; padding-right: 0; }
			}
		</style>
		<div class="flex items-start space-x-3">
			<div class="flex-shrink-0 mt-0.5">
				<div class="flex items-center justify-center w-8 h-8 bg-green-100 rounded-full">
					<svg class="w-4 h-4 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
					</svg>
				</div>
			</div>
			<div class="flex-1">
				<h3 class="text-sm font-semibold text-green-900 mb-1">
					Audit Started Successfully!
				</h3>
				<p class="text-sm text-green-800 mb-3">
					Your SharePoint audit has been queued and will begin processing shortly.
				</p>
				<div class="bg-green-900 bg-opacity-10 border border-green-200 rounded-md px-3 py-2 mb-3">
					<div class="text-xs text-green-900 space-y-1">
						<div><span class="font-medium">Job ID:</span> <code class="font-mono">%s</code></div>
						<div><span class="font-medium">Site:</span> <code class="font-mono break-all">%s</code></div>
					</div>
				</div>
				<div class="flex items-start space-x-2 text-xs text-green-700">
					<svg class="w-4 h-4 text-green-600 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
					</svg>
					<p><span class="font-medium">💡 Tip:</span> Watch the "Background Jobs" section below for real-time progress updates!</p>
				</div>
			</div>
		</div>
	</div>`, request.ID, request.SiteURL)
}

// FormatAuditErrorResponse creates animated error HTML response for audit failures.
func (p *AuditPresenter) FormatAuditErrorResponse(err error) string {
	return fmt.Sprintf(`<div class="audit-error-message bg-gradient-to-r from-red-50 to-red-100 border-l-4 border-red-500 shadow-sm">
		<style>
			.audit-error-message {
				animation: fadeInThenOut 8s ease-in-out forwards;
				overflow: hidden;
				border-radius: 0.5rem;
			}
			@keyframes fadeInThenOut {
				0%% { opacity: 0; transform: translateY(-10px); max-height: 0; margin-bottom: 0; padding-top: 0; padding-bottom: 0; }
				10%% { opacity: 1; transform: translateY(0); max-height: 300px; margin-bottom: 16px; padding-top: 20px; padding-bottom: 20px; padding-left: 16px; padding-right: 16px; }
				85%% { opacity: 1; transform: translateY(0); max-height: 300px; margin-bottom: 16px; padding-top: 20px; padding-bottom: 20px; padding-left: 16px; padding-right: 16px; }
				100%% { opacity: 0; transform: translateY(-10px); max-height: 0; margin-bottom: 0; padding-top: 0; padding-bottom: 0; padding-left: 0; padding-right: 0; }
			}
		</style>
		<div class="flex items-start space-x-3">
			<div class="flex-shrink-0 mt-0.5">
				<div class="flex items-center justify-center w-8 h-8 bg-red-100 rounded-full">
					<svg class="w-4 h-4 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
					</svg>
				</div>
			</div>
			<div class="flex-1">
				<h3 class="text-sm font-semibold text-red-900 mb-1">
					Failed to Start Audit
				</h3>
				<p class="text-sm text-red-800 mb-3">
					Unable to start your SharePoint audit due to the following error:
				</p>
				<div class="bg-red-900 bg-opacity-10 border border-red-200 rounded-md px-3 py-2 mb-3">
					<code class="text-xs text-red-900 font-mono break-words">%s</code>
				</div>
				<div class="flex items-start space-x-2 text-xs text-red-700">
					<svg class="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
					</svg>
					<p>Verify your SharePoint site URL is correct and accessible. Contact your administrator if the issue persists.</p>
				</div>
			</div>
		</div>
	</div>`, err.Error())
}

// FormatAuditConflictResponse creates animated warning HTML response for audit conflicts.
func (p *AuditPresenter) FormatAuditConflictResponse(err error) string {
	return fmt.Sprintf(`<div class="audit-conflict-message bg-gradient-to-r from-amber-50 to-orange-50 border-l-4 border-amber-500 shadow-sm">
		<style>
			.audit-conflict-message {
				animation: fadeInThenOut 7s ease-in-out forwards;
				overflow: hidden;
				border-radius: 0.5rem;
			}
			@keyframes fadeInThenOut {
				0%% { opacity: 0; transform: translateY(-10px); max-height: 0; margin-bottom: 0; padding-top: 0; padding-bottom: 0; }
				10%% { opacity: 1; transform: translateY(0); max-height: 250px; margin-bottom: 16px; padding-top: 20px; padding-bottom: 20px; padding-left: 16px; padding-right: 16px; }
				85%% { opacity: 1; transform: translateY(0); max-height: 250px; margin-bottom: 16px; padding-top: 20px; padding-bottom: 20px; padding-left: 16px; padding-right: 16px; }
				100%% { opacity: 0; transform: translateY(-10px); max-height: 0; margin-bottom: 0; padding-top: 0; padding-bottom: 0; padding-left: 0; padding-right: 0; }
			}
		</style>
		<div class="flex items-start space-x-3">
			<div class="flex-shrink-0 mt-0.5">
				<div class="flex items-center justify-center w-8 h-8 bg-amber-100 rounded-full">
					<svg class="w-4 h-4 text-amber-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.864-.833-2.634 0L4.18 16.5c-.77.833.192 2.5 1.732 2.5z"></path>
					</svg>
				</div>
			</div>
			<div class="flex-1">
				<h3 class="text-sm font-semibold text-amber-900 mb-1">
					Audit Already in Progress
				</h3>
				<p class="text-sm text-amber-800 mb-3">
					An audit is currently running or queued for this SharePoint site.
				</p>
				<div class="bg-amber-900 bg-opacity-10 border border-amber-200 rounded-md px-3 py-2 mb-3">
					<code class="text-xs text-amber-900 font-mono break-words">%s</code>
				</div>
				<div class="flex items-start space-x-2 text-xs text-amber-700">
					<svg class="w-4 h-4 text-amber-600 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
					</svg>
					<p>Please wait for the current audit to complete before starting a new one. Check the "Background Jobs" section below for real-time progress.</p>
				</div>
			</div>
		</div>
	</div>`, err.Error())
}
