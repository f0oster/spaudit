package main

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
	"github.com/joho/godotenv"

	"spaudit/application"
	"spaudit/database"
	"spaudit/domain/contracts"
	jobsdom "spaudit/domain/jobs"
	"spaudit/gen/db"
	"spaudit/infrastructure/config"
	infrafactories "spaudit/infrastructure/factories"
	"spaudit/infrastructure/repositories"
	"spaudit/interfaces/web/handlers"
	"spaudit/interfaces/web/presenters"
	templates "spaudit/interfaces/web/templates"
	"spaudit/logging"
	"spaudit/platform/events"
	"spaudit/platform/executors"
	"spaudit/platform/factories"
)

func main() {
	// Create app-wide context for graceful shutdown
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	// Initialize configuration
	loadEnvironment()
	cfg := config.LoadAppConfigFromEnv()

	// Initialize logging
	logger := initializeLogging(cfg)

	// Initialize database
	db := initializeDatabase(cfg, logger)
	defer db.Close()

	// Build dependencies with app context
	deps := buildDependencies(appCtx, db, logger)

	// Setup routes and start server
	router := setupRoutes(deps, cfg)
	startServer(router, cfg.HTTPAddr, logger, deps, appCancel)
}

// ApplicationServices holds application services.
type ApplicationServices struct {
	JobService          application.JobService
	AuditService        application.AuditService
	SiteContentService  *application.SiteContentService
	PermissionService   *application.PermissionService
	SiteBrowsingService *application.SiteBrowsingService
	EventBus            *events.JobEventBus
	ServiceFactory      application.AuditRunScopedServiceFactory
}

// PresentationLayer groups all presentation components
type PresentationLayer struct {
	// Presenters
	AuditPresenter      *presenters.AuditPresenter
	JobPresenter        *presenters.JobPresenter
	ListPresenter       *presenters.ListPresenter
	PermissionPresenter *presenters.PermissionPresenter
	SitePresenter       *presenters.SitePresenter

	// Handlers
	ListHandlers  *handlers.ListHandlers
	AuditHandlers *handlers.AuditHandlers
	JobHandlers   *handlers.JobHandlers
	SSEManager    *handlers.SSEManager
}

// Dependencies holds all application dependencies organized by layer
type Dependencies struct {
	// Infrastructure
	DB      *database.Database
	Queries *db.Queries
	Logger  *logging.Logger

	// Repositories
	JobRepo contracts.JobRepository

	// Application Layer
	Services *ApplicationServices

	// Presentation Layer
	Presentation *PresentationLayer
}

func loadEnvironment() {
	if err := godotenv.Load(); err != nil {
		println("No .env file found, using environment variables")
	} else {
		println("Loaded configuration from .env file")
	}
}

func initializeLogging(cfg *config.AppConfig) *logging.Logger {
	logger := logging.NewLogger(cfg.Logging)
	logging.SetDefault(logger)

	logger.Info("Application starting",
		"version", "1.0.0",
		"log_level", cfg.Logging.Level,
		"log_format", cfg.Logging.Format,
		"db_path", cfg.Database.Path,
	)

	return logger
}

func initializeDatabase(cfg *config.AppConfig, logger *logging.Logger) *database.Database {
	db, err := database.New(*cfg.Database, logger)
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	return db
}

// RepositoryBundle holds all repository implementations
type RepositoryBundle struct {
	JobRepo     contracts.JobRepository
	AuditRepo   contracts.AuditRepository
	SiteRepo    contracts.SiteRepository
	ListRepo    contracts.ListRepository
	ItemRepo    contracts.ItemRepository
	SharingRepo contracts.SharingRepository

	// Aggregate repositories
	SiteContentAggregate contracts.SiteContentAggregateRepository
	PermissionAggregate  contracts.PermissionAggregateRepository
}

// buildRepositories creates all repository implementations with read/write database separation
func buildRepositories(database *database.Database) *RepositoryBundle {
	// Create base repository for shared functionality
	baseRepo := repositories.NewBaseRepository(database)

	// Create entity repositories (Tier 1)
	jobRepo := repositories.NewSqlcJobRepository(database)
	auditRepo := repositories.NewSqlcAuditRepository(database)
	siteRepo := repositories.NewSqlcSiteRepository(database)
	listRepo := repositories.NewSqlcListRepository(database)
	itemRepo := repositories.NewSqlcItemRepository(database)
	sharingRepo := repositories.NewSqlcSharingRepository(database)

	// Create aggregate repositories (Tier 2) - compose entity repositories
	siteContentAggregate := repositories.NewSiteContentAggregateRepository(
		baseRepo,
		siteRepo,
		listRepo,
		jobRepo,
		itemRepo,
		sharingRepo,
	)
	permissionAggregate := repositories.NewPermissionAggregateRepository(
		baseRepo,
		itemRepo,
		sharingRepo,
	)

	return &RepositoryBundle{
		JobRepo:     jobRepo,
		AuditRepo:   auditRepo,
		SiteRepo:    siteRepo,
		ListRepo:    listRepo,
		ItemRepo:    itemRepo,
		SharingRepo: sharingRepo,

		// Aggregate repositories
		SiteContentAggregate: siteContentAggregate,
		PermissionAggregate:  permissionAggregate,
	}
}

// buildApplicationServices creates application services with dependency injection.
func buildApplicationServices(appCtx context.Context, db *database.Database, repos *RepositoryBundle) *ApplicationServices {
	// Create event bus for job events
	eventBus := events.NewJobEventBus()

	// Create platform factories
	auditWorkflowFactory := factories.NewAuditWorkflowFactory(db)

	// Create platform executors
	siteAuditExecutor := executors.NewSiteAuditExecutor(auditWorkflowFactory)

	// Create job executor registry and register executors
	registry := application.NewJobExecutorRegistry()
	registry.RegisterExecutor(jobsdom.JobTypeSiteAudit, siteAuditExecutor)

	// Create job service
	// TODO: Pass appCtx to JobService for graceful job cancellation
	jobService := application.NewJobService(repos.JobRepo, repos.AuditRepo, registry, nil, eventBus)
	auditService := application.NewAuditService(jobService, db)

	// Services using aggregate repositories
	siteContentService := application.NewSiteContentService(
		repos.SiteContentAggregate,
	)
	permissionService := application.NewPermissionService(
		repos.PermissionAggregate,
	)
	siteBrowsingService := application.NewSiteBrowsingService(repos.SiteContentAggregate)

	// Create service factory for audit-run-scoped services
	repositoryFactory := infrafactories.NewScopedRepositoryFactory(db)
	serviceFactory := application.NewAuditRunScopedServiceFactory(repositoryFactory, repos.AuditRepo)

	return &ApplicationServices{
		JobService:          jobService,
		AuditService:        auditService,
		SiteContentService:  siteContentService,
		PermissionService:   permissionService,
		SiteBrowsingService: siteBrowsingService,
		EventBus:            eventBus,
		ServiceFactory:      serviceFactory,
	}
}

// buildPresentationLayer creates all presenters and handlers
func buildPresentationLayer(appCtx context.Context, services *ApplicationServices) *PresentationLayer {
	// Build presenters (view logic)
	auditPresenter := presenters.NewAuditPresenter()
	jobPresenter := presenters.NewJobPresenter()
	listPresenter := presenters.NewListPresenter()
	permissionPresenter := presenters.NewPermissionPresenter()
	sitePresenter := presenters.NewSitePresenter()

	// Build handlers - orchestrate services & presenters
	sseManager := handlers.NewSSEManager(appCtx)
	listHandlers := handlers.NewListHandlers(
		services.SiteContentService,
		services.PermissionService,
		services.SiteBrowsingService,
		services.JobService,
		services.AuditService,
		listPresenter,
		permissionPresenter,
		sitePresenter,
		services.ServiceFactory,
	)
	auditHandlers := handlers.NewAuditHandlers(services.AuditService, auditPresenter, sseManager)
	jobHandlers := handlers.NewJobHandlers(services.JobService, jobPresenter)

	// Wire up update notifications
	services.JobService.SetUpdateNotifier(sseManager)

	// Setup event system for job notifications
	setupEventHandlers(services, sseManager)

	return &PresentationLayer{
		AuditPresenter:      auditPresenter,
		JobPresenter:        jobPresenter,
		ListPresenter:       listPresenter,
		PermissionPresenter: permissionPresenter,
		SitePresenter:       sitePresenter,
		ListHandlers:        listHandlers,
		AuditHandlers:       auditHandlers,
		JobHandlers:         jobHandlers,
		SSEManager:          sseManager,
	}
}

// buildDependencies creates all application dependencies
func buildDependencies(appCtx context.Context, db *database.Database, logger *logging.Logger) *Dependencies {
	queries := db.Queries()

	// Build each layer
	repos := buildRepositories(db)
	services := buildApplicationServices(appCtx, db, repos)
	presentation := buildPresentationLayer(appCtx, services)

	return &Dependencies{
		DB:           db,
		Queries:      queries,
		JobRepo:      repos.JobRepo,
		Services:     services,
		Presentation: presentation,
		Logger:       logger,
	}
}

func setupRoutes(deps *Dependencies, cfg *config.AppConfig) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	setupHTTPLogging(r, deps, cfg)
	r.Use(middleware.Recoverer)

	// Static assets
	mountStaticAssets(r)

	// System endpoints
	setupSystemRoutes(r, deps)

	// Main application routes
	setupApplicationRoutes(r, deps)

	// Audit routes
	setupAuditRoutes(r, deps)

	return r
}

func setupHTTPLogging(r *chi.Mux, deps *Dependencies, cfg *config.AppConfig) {
	if cfg.HTTPLogPath == "" {
		// No HTTP logging configured, skip
		return
	}

	logFile, err := os.OpenFile(cfg.HTTPLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		deps.Logger.Error("Failed to open HTTP log file", "error", err, "path", cfg.HTTPLogPath)
		return
	}
	// Note: logFile is not closed here as it needs to stay open for the server lifetime

	httpLogger := httplog.NewLogger("spaudit", httplog.Options{
		Writer: logFile,
		JSON:   true,
	})
	r.Use(httplog.RequestLogger(httpLogger))

	deps.Logger.Info("HTTP request logging enabled", "path", cfg.HTTPLogPath)
}

func mountStaticAssets(r chi.Router) {
	sub, _ := fs.Sub(templates.FS, "assets")
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.FS(sub))))
}

func setupSystemRoutes(r *chi.Mux, deps *Dependencies) {
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		stats, err := deps.DB.Health()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"status":   "ok",
			"database": stats,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	r.Get("/events", deps.Presentation.SSEManager.HandleSSEConnection)
}

func setupApplicationRoutes(r *chi.Mux, deps *Dependencies) {
	// Main pages
	r.Get("/", deps.Presentation.ListHandlers.Home)

	// Site management (non-audit scoped)
	r.Get("/sites", deps.Presentation.ListHandlers.SitesTable)
	r.Get("/sites/search", deps.Presentation.ListHandlers.SearchSites)
	

	// API endpoints for audit runs
	r.Get("/api/sites/{siteID}/audit-runs", deps.Presentation.ListHandlers.GetAuditRunsForSite)
	
	// Audit-run-scoped routes
	r.Get("/sites/{siteID}/audit-runs/{auditRunID}/lists", deps.Presentation.ListHandlers.SiteListsPage)
	r.Get("/sites/{siteID}/audit-runs/{auditRunID}/lists/search", deps.Presentation.ListHandlers.SearchLists)

	// List details
	r.Get("/sites/{siteID}/audit-runs/{auditRunID}/lists/{listID}", deps.Presentation.ListHandlers.ListDetail)

	// List tabs (HTMX partials)
	r.Get("/sites/{siteID}/audit-runs/{auditRunID}/tabs/{listID}/overview", deps.Presentation.ListHandlers.OverviewTab)
	r.Get("/sites/{siteID}/audit-runs/{auditRunID}/tabs/{listID}/assignments", deps.Presentation.ListHandlers.AssignmentsTab)
	r.Get("/sites/{siteID}/audit-runs/{auditRunID}/tabs/{listID}/items", deps.Presentation.ListHandlers.ItemsTab)
	r.Get("/sites/{siteID}/audit-runs/{auditRunID}/tabs/{listID}/links", deps.Presentation.ListHandlers.LinksTab)

	// Object operations (HTMX partials)
	r.Get("/sites/{siteID}/audit-runs/{auditRunID}/object/{otype}/{okey}/assignments", deps.Presentation.ListHandlers.GetObjectAssignments)
	r.Post("/sites/{siteID}/audit-runs/{auditRunID}/assignments/{uniqueID}/toggle", deps.Presentation.ListHandlers.ToggleAssignment)
	r.Post("/sites/{siteID}/audit-runs/{auditRunID}/items/{itemGUID}/assignments/toggle", deps.Presentation.ListHandlers.ToggleItemAssignments)

	// Sharing link operations (HTMX partials)
	r.Get("/sites/{siteID}/audit-runs/{auditRunID}/sharing-links/{linkID}/members", deps.Presentation.ListHandlers.GetSharingLinkMembers)
	r.Post("/sites/{siteID}/audit-runs/{auditRunID}/sharing-links/{linkID}/members/toggle", deps.Presentation.ListHandlers.ToggleSharingLinkMembers)
	
	// Audit run switching
	r.Get("/sites/{siteID}/switch-audit-run", deps.Presentation.ListHandlers.SwitchAuditRun)
	r.Post("/sites/{siteID}/switch-audit-run", deps.Presentation.ListHandlers.SwitchAuditRun)
}

func setupAuditRoutes(r *chi.Mux, deps *Dependencies) {
	// Audit operations
	r.Post("/audit", deps.Presentation.AuditHandlers.RunAudit)
	r.Get("/audit/status", deps.Presentation.AuditHandlers.GetAuditStatus)
	r.Get("/audit/active", deps.Presentation.AuditHandlers.ListActiveAudits)

	// Job management
	r.Get("/jobs", deps.Presentation.JobHandlers.ListJobs)

	// Job cancellation
	r.Post("/jobs/{jobID}/cancel", deps.Presentation.JobHandlers.CancelJob)
}

func startServer(router *chi.Mux, addr string, logger *logging.Logger, deps *Dependencies, appCancel context.CancelFunc) {
	server := &http.Server{Addr: addr, Handler: router}

	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sig
		logger.Info("Shutdown signal received")

		// Cancel app-wide context first to signal all services to shutdown
		logger.Info("Cancelling app context...")
		appCancel()

		// Close SSE connections immediately
		logger.Info("Closing SSE connections...")
		deps.Presentation.SSEManager.CloseAll()

		shutdownCtx, cancel := context.WithTimeout(serverCtx, 30*time.Second)
		defer cancel()

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				logger.Error("Graceful shutdown timed out, forcing exit")
				os.Exit(1)
			}
		}()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("Server shutdown error", "error", err)
			os.Exit(1)
		}
		serverStopCtx()
	}()

	logger.Info("Server starting", "address", addr)
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}

	<-serverCtx.Done()
	logger.Info("Server stopped")
}

// setupEventHandlers wires up the event handlers for job notifications
func setupEventHandlers(services *ApplicationServices, sseManager *handlers.SSEManager) {
	// Create event handlers using the event bus from services
	notificationHandlers := events.NewNotificationEventHandlers(sseManager, services.SiteBrowsingService)

	// Register all event handlers with the existing event bus
	notificationHandlers.RegisterHandlers(services.EventBus)
}
