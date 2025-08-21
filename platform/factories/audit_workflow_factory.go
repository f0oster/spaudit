package factories

import (
	"context"
	"fmt"

	"spaudit/application"
	"spaudit/database"
	"spaudit/domain/audit"
	"spaudit/domain/jobs"
	"spaudit/domain/sharepoint"
	"spaudit/infrastructure/repositories"
	"spaudit/infrastructure/spclient"
	"spaudit/logging"
	"spaudit/platform/workflows"
	"spaudit/spauth"

	"github.com/koltyakov/gosip/api"
)

// AuditWorkflowFactory creates fully configured audit workflows
type AuditWorkflowFactory struct {
	db     *database.Database
	logger *logging.Logger
}

// NewAuditWorkflowFactory creates a new audit workflow factory
func NewAuditWorkflowFactory(db *database.Database) *AuditWorkflowFactory {
	return &AuditWorkflowFactory{
		db:     db,
		logger: logging.Default().WithComponent("audit_workflow_factory"),
	}
}

// CreateAuditWorkflow creates a fully configured audit workflow for the specified site
func (f *AuditWorkflowFactory) CreateAuditWorkflow(siteURL string, parameters *audit.AuditParameters) (application.AuditWorkflow, error) {
	f.logger.Info("Creating audit workflow", "siteURL", siteURL)

	// Use default parameters if none provided
	if parameters == nil {
		parameters = audit.DefaultParameters()
	}

	// Create SharePoint client for this specific site
	spClient, err := f.createSharePointClient(siteURL, parameters)
	if err != nil {
		return nil, fmt.Errorf("create SharePoint client: %w", err)
	}

	// Initialize base repositories
	f.logger.Info("Creating base repositories")
	baseRepo := repositories.NewBaseRepository(f.db)
	baseAuditRepo := repositories.NewSqlcAuditRepository(f.db)

	// Get the site_id for this siteURL (site should already exist from job creation)
	ctx := context.Background()
	site, err := baseAuditRepo.GetSiteByURL(ctx, siteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get site for URL %s: %w", siteURL, err)
	}
	if site == nil {
		return nil, fmt.Errorf("site not found for URL %s - site should have been created during job setup", siteURL)
	}

	f.logger.Info("Found site for workflow", "site_url", siteURL, "site_id", site.ID)

	// Create site-scoped SharePoint audit repository with the real site_id
	sharepointAuditRepo := repositories.NewSharePointAuditRepository(baseRepo, site.ID, baseAuditRepo)
	f.logger.Info("Created repositories")

	// Create other repositories
	f.logger.Info("Creating other repositories")
	sharingRepo := repositories.NewSqlcSharingRepository(f.db)
	itemRepo := repositories.NewSqlcItemRepository(f.db)
	f.logger.Info("Created other repositories")

	// Create audit workflow with repositories
	f.logger.Info("Creating audit workflow")
	auditWorkflow := workflows.NewAuditWorkflow(
		sharepointAuditRepo,
		sharingRepo,
		itemRepo,
		spClient,
		f.db,
	)
	f.logger.Info("Audit workflow created successfully")

	return &WorkflowAdapter{workflow: auditWorkflow}, nil
}

// createSharePointClient creates a properly configured SharePoint client for the specific site
func (f *AuditWorkflowFactory) createSharePointClient(siteURL string, parameters *audit.AuditParameters) (spclient.SharePointClient, error) {
	f.logger.Info("Setting up SharePoint authentication", "siteURL", siteURL)

	// Setup SharePoint authentication
	cfg, err := spauth.FromEnv()
	if err != nil {
		return nil, fmt.Errorf("auth config error: %w", err)
	}
	cfg.SiteURL = siteURL

	client, err := spauth.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("auth client error: %w", err)
	}

	// Create SharePoint client adapter with parameters
	sp := api.NewSP(client)
	spClient := spclient.NewSharePointClient(sp, client, parameters)

	f.logger.Info("SharePoint client created successfully", "siteURL", siteURL)
	return spClient, nil
}

// WorkflowAdapter adapts the concrete workflow to the application interface
type WorkflowAdapter struct {
	workflow *workflows.AuditWorkflow
}

// ExecuteSiteAudit implements the application.AuditWorkflow interface
func (w *WorkflowAdapter) ExecuteSiteAudit(ctx context.Context, job *jobs.Job, siteURL string) (application.AuditWorkflowResult, error) {
	result, err := w.workflow.ExecuteSiteAudit(ctx, job, siteURL)
	if err != nil {
		return nil, err
	}
	return &WorkflowResultAdapter{result: result}, nil
}

// SetProgressReporter implements the application.AuditWorkflow interface
func (w *WorkflowAdapter) SetProgressReporter(reporter application.ProgressReporter) {
	w.workflow.SetProgressReporter(reporter)
}

// WorkflowResultAdapter adapts the concrete workflow result to the application interface
type WorkflowResultAdapter struct {
	result *workflows.AuditWorkflowResult
}

// GetDuration implements the application.AuditWorkflowResult interface
func (r *WorkflowResultAdapter) GetDuration() string {
	return r.result.Duration.String()
}

// GetTotalLists implements the application.AuditWorkflowResult interface
func (r *WorkflowResultAdapter) GetTotalLists() int {
	return r.result.TotalLists
}

// GetTotalItems implements the application.AuditWorkflowResult interface
func (r *WorkflowResultAdapter) GetTotalItems() int64 {
	return r.result.TotalItems
}

// GetItemsWithUnique implements the application.AuditWorkflowResult interface
func (r *WorkflowResultAdapter) GetItemsWithUnique() int64 {
	return r.result.ItemsWithUnique
}

// GetContentRisk implements the application.AuditWorkflowResult interface
func (r *WorkflowResultAdapter) GetContentRisk() application.RiskAssessment {
	return &ContentRiskAdapter{risk: r.result.ContentRisk}
}

// GetSharingRisk implements the application.AuditWorkflowResult interface
func (r *WorkflowResultAdapter) GetSharingRisk() application.SharingRiskAssessment {
	return &SharingRiskAdapter{risk: r.result.SharingRisk}
}

// GetPermissionAnalysis implements the application.AuditWorkflowResult interface
func (r *WorkflowResultAdapter) GetPermissionAnalysis() application.PermissionRiskAssessment {
	return &PermissionRiskAdapter{risk: r.result.PermissionAnalysis}
}

// ContentRiskAdapter adapts content risk to the application interface
type ContentRiskAdapter struct {
	risk *sharepoint.ContentRiskAssessment
}

func (c *ContentRiskAdapter) GetRiskLevel() string {
	if c.risk == nil {
		return "unknown"
	}
	return c.risk.RiskLevel
}

func (c *ContentRiskAdapter) GetRiskScore() float64 {
	if c.risk == nil {
		return 0.0
	}
	return c.risk.RiskScore
}

// SharingRiskAdapter adapts sharing risk to the application interface
type SharingRiskAdapter struct {
	risk *sharepoint.SharingRiskAssessment
}

func (s *SharingRiskAdapter) GetRiskLevel() string {
	if s.risk == nil {
		return "unknown"
	}
	return s.risk.RiskLevel
}

func (s *SharingRiskAdapter) GetTotalLinks() int {
	if s.risk == nil {
		return 0
	}
	return s.risk.TotalLinks
}

// PermissionRiskAdapter adapts permission risk to the application interface
type PermissionRiskAdapter struct {
	risk *sharepoint.SharePointRiskAssessment
}

func (p *PermissionRiskAdapter) GetRiskLevel() string {
	if p.risk == nil {
		return "unknown"
	}
	return p.risk.RiskLevel
}

func (p *PermissionRiskAdapter) GetRiskScore() float64 {
	if p.risk == nil {
		return 0.0
	}
	return p.risk.RiskScore
}

// Ensure AuditWorkflowFactory implements the application interface
var _ application.WorkflowFactory = (*AuditWorkflowFactory)(nil)
