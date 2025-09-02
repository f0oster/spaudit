# SharePoint Audit Tool - Architecture Documentation

This document describes the target technical architecture and implementation patterns. Components described may represent the intended design rather than current implementation state.

## Table of Contents
- [Architecture Principles](#architecture-principles)
- [Layer Details](#layer-details)
- [Data Flow Architecture](#data-flow-architecture)  
- [Job System Architecture](#job-system-architecture)
- [Component Architecture](#component-architecture)
- [Database Design](#database-design)
- [Development Patterns](#development-patterns)

## Architecture Principles

The project uses layered architecture with dependency inversion:

### Core Principles
- **Dependency Direction**: All dependencies point inward toward the domain
- **Domain Isolation**: Domain code imports no external packages
- **Interface Segregation**: Repository interfaces define clear contracts
- **Composition over Inheritance**: Services compose repositories rather than inherit

### Layer Responsibilities
```
┌─────────────────────────────────────────────────────────────┐
│                   External Users                            │
│                  (HTTP Requests)                            │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────┐ ┌─────────────────────────────────────┐
│            Interface Layer          │ │            Platform Layer           │
│    HTTP Handlers, Templates,        │ │      Workflows, Job Executors,      │
│         Presenters                  │ │            Factories                │
│        (interfaces/web/)            │ │           (platform/)               │
└─────────────────┬───────────────────┘ └─────────────────┬───────────────────┘
                  │                                       │
                  └───────────────┬───────────────────────┘
                                  ▼
                  ┌─────────────────────────────────────────────────────────────┐
                  │               Application Layer                             │
                  │                    Services                                 │
                  │            (application/)                                   │
                  └─────────────────────────┬───────────────────────────────────┘
                                            │
                  ┌─────────────────────────▼───────────────────────────────────┐
                  │               Infrastructure Layer                          │
                  │    Repositories, SharePoint Client, Database                │
                  │          (infrastructure/)                                  │
                  └─────────────────────────┬───────────────────────────────────┘
                                            │
                  ┌─────────────────────────▼───────────────────────────────────┐
                  │                 Domain Layer                                │
                  │    Entities, Value Objects, Domain Services                 │
                  │    Repository Contracts, Domain Events                      │
                  │              (domain/)                                      │
                  └─────────────────────────────────────────────────────────────┘
```

**Dependency Direction**: Both Interface and Platform layers depend directly on the Application layer. Infrastructure layer depends on Domain contracts. All dependencies point toward the Domain layer.

## Layer Details

### Domain Layer (`domain/`)

Pure business entities with no external dependencies.

#### Key Components
```go
// domain/sharepoint/site.go
type Site struct {
    ID    int64
    URL   string
    Title string
}

// domain/jobs/job.go  
type Job struct {
    ID          string
    Type        JobType
    Status      JobStatus
    AuditRunID  *int64           // Historical tracking
    StartedAt   time.Time
    CompletedAt *time.Time
    State       JobState         // Rich progress state
    Result      string
    Error       string
    Context     JobContextData   // Job-specific context
}

// domain/events/job_events.go
type JobCompletedEvent struct {
    Job       *jobs.Job
    Timestamp time.Time
}
```

#### Repository Contracts
```go
// domain/contracts/site_repository.go
type SiteRepository interface {
    GetByID(ctx context.Context, siteID int64) (*sharepoint.Site, error)
    GetByURL(ctx context.Context, siteURL string) (*sharepoint.Site, error)
    Save(ctx context.Context, site *sharepoint.Site) error
}
```

### Application Layer (`application/`)

Services that compose repositories, domain logic, and infrastructure components.

```go
// application/site_content_service.go
type SiteContentService struct {
    contentAggregate contracts.SiteContentAggregateRepository
    auditRunID       int64 // For audit-scoped operations
}

func (s *SiteContentService) GetSiteWithLists(ctx context.Context, siteID int64) (*SiteWithListsData, error) {
    // 1. Get site with metadata from aggregate repository
    siteWithMeta, err := s.contentAggregate.GetSiteWithMetadata(ctx, siteID)
    
    // 2. Get lists for the site (automatically audit-run filtered)
    lists, err := s.contentAggregate.GetListsForSite(ctx, siteID)
    
    // 3. Calculate aggregated metrics
    return s.calculateMetrics(siteWithMeta, lists), nil
}
```

### Infrastructure Layer (`infrastructure/`)

Implementations that fulfill domain contracts.

#### Individual Repository Pattern
```go
// infrastructure/repositories/sqlc_site_repository.go
type SqlcSiteRepository struct {
    BaseRepository  // Embedded for shared SQL type conversion
    queries *db.Queries
}

func (r *SqlcSiteRepository) GetByID(ctx context.Context, siteID int64) (*sharepoint.Site, error) {
    // 1. Database query via SQLC
    siteRow, err := r.queries.GetSiteByID(ctx, siteID)
    
    // 2. Database model to domain model conversion
    return &sharepoint.Site{
        ID:    siteRow.SiteID,
        URL:   siteRow.SiteUrl,
        Title: r.FromNullString(siteRow.Title), // BaseRepository utility
    }, nil
}
```

#### Aggregate Repository Pattern
Combines multiple repository operations for complex use cases:

```go
// infrastructure/repositories/site_content_aggregate_repository.go
type SiteContentAggregateRepository struct {
    siteRepo         contracts.SiteRepository
    listRepo         contracts.ListRepository
    permissionRepo   contracts.PermissionRepository
    auditRepo        contracts.AuditRepository
    auditRunID       int64 // Scoping to specific audit run
}

func (r *SiteContentAggregateRepository) GetSiteWithMetadata(ctx context.Context, siteID int64) (*SiteWithMetadata, error) {
    // 1. Get site entity
    site, err := r.siteRepo.GetByID(ctx, siteID)
    
    // 2. Get audit metadata for this audit run
    auditRun, err := r.auditRepo.GetAuditRun(ctx, r.auditRunID)
    
    // 3. Calculate business metrics across repositories
    return &SiteWithMetadata{
        Site:            site,
        LastAuditDate:   auditRun.StartedAt,
        LastAuditDaysAgo: int(time.Since(auditRun.StartedAt).Hours() / 24),
        AuditRunID:      r.auditRunID,
    }, nil
}

func (r *SiteContentAggregateRepository) GetListsForSite(ctx context.Context, siteID int64) ([]*ListSummary, error) {
    // Automatically filtered to audit run scope
    return r.listRepo.GetListsSummarizedBySiteAndAuditRun(ctx, siteID, r.auditRunID)
}
```

### Interface Layer (`interfaces/web/`)

HTTP handling with separated concerns.

```go
// interfaces/web/handlers/list_handlers.go
type ListHandlers struct {
    serviceFactory application.AuditRunScopedServiceFactory
    listPresenter  *presenters.ListPresenter
}

func (h *ListHandlers) SiteListsPage(w http.ResponseWriter, r *http.Request) {
    // 1. Extract HTTP parameters including audit run ID
    siteID := extractSiteID(r)
    auditRunID := extractAuditRunID(r) // "latest" or specific ID

    // 2. Create audit-run-scoped services
    scopedServices, err := h.serviceFactory.CreateForAuditRun(r.Context(), siteID, auditRunID)

    // 3. Get data (automatically filtered to audit run)
    data, err := scopedServices.SiteContentService.GetSiteWithLists(r.Context(), siteID)

    // 4. Transform to view model
    viewModel := h.listPresenter.ToSiteListsViewModel(data)

    // 5. Render response
    templates.SiteListsPage(*viewModel).Render(r.Context(), w)
}
```

## Data Flow Architecture

### Request Flow (Web UI)

```
HTTP Request 
    ↓
interfaces/web/handlers/ (HTTP concerns & parameter extraction)
    ↓
application/*_service.go (Business logic orchestration)
    ↓
domain/contracts/*_repository.go (Interface contracts)
    ↓
infrastructure/repositories/sqlc_*_repository.go (Database implementation)
    ↓
gen/db/ (SQLC generated queries)
    ↓
SQLite Database
```

### Background Job Flow

```
Job Creation (Web UI)
    ↓
application/audit_service.go (Audit configuration & job creation)
    ↓
application/job_service_impl.go (Job lifecycle management & async execution)
    ↓
platform/executors/site_audit_executor.go (Job execution implementation)
    ↓
platform/workflows/audit_workflow.go (Multi-step audit orchestration)
    ↓
infrastructure/spauditor/sharepoint_data_collector.go (SharePoint data collection)
    ↓
infrastructure/spclient/ (SharePoint API client)
    ↓
infrastructure/repositories/ (Data persistence with audit run tracking)
```

## Job System Architecture

The job system handles background processing for SharePoint audits.

### Design Principles
- **Single Responsibility**: Each component handles one job lifecycle concern
- **Event-Driven**: Domain events separate job execution from UI updates
- **Linear Execution**: Sequential flow without complex callbacks
- **Context Cancellation**: Goroutine cleanup and termination support

### Job System Components

#### 1. Job Service (`application/job_service_impl.go`)

Manages job lifecycle with context cancellation.

```go
type JobServiceImpl struct {
    jobRepo      contracts.JobRepository
    auditRepo    contracts.AuditRepository
    registry     *JobExecutorRegistry
    notifier     UpdateNotifier
    eventBus     EventPublisher
    logger       *logging.Logger
    
    // Context cancellation for running jobs
    runningJobs map[string]context.CancelFunc
    jobsMutex   sync.RWMutex
}
```

**Key Responsibilities:**
- Job lifecycle management (create → start → execute → complete/fail)
- Context-based cancellation with goroutine cleanup
- Progress tracking and state management
- Event publishing for UI updates

#### 2. Job Executors (`platform/executors/`)

Job executors implementing the `JobExecutor` interface.

```go
type JobExecutor interface {
    Execute(ctx context.Context, job *jobs.Job, progressCallback ProgressCallback) error
}

type SiteAuditExecutor struct {
    workflowFactory *factories.AuditWorkflowFactory
    logger          *logging.Logger
}
```

#### 3. Event Bus (`platform/events/job_event_bus.go`)

Event publishing and subscription with panic recovery.

```go
type JobEventBus struct {
    mu                           sync.RWMutex
    jobCompletedHandlers         []func(events.JobCompletedEvent)
    jobFailedHandlers           []func(events.JobFailedEvent)
    jobCancelledHandlers        []func(events.JobCancelledEvent)
}

func (bus *JobEventBus) PublishJobCompleted(event events.JobCompletedEvent) {
    for _, handler := range bus.jobCompletedHandlers {
        go func(h func(events.JobCompletedEvent)) {
            defer func() {
                if r := recover(); r != nil {
                    bus.logger.Error("Event handler panicked", "panic", r)
                }
            }()
            h(event)
        }(handler)
    }
}
```

### Job Lifecycle States

```go
const (
    JobStatusPending   JobStatus = "pending"    // Queued, awaiting execution
    JobStatusRunning   JobStatus = "running"    // Currently executing
    JobStatusCompleted JobStatus = "completed"  // Successfully finished
    JobStatusFailed    JobStatus = "failed"     // Execution failed
    JobStatusCancelled JobStatus = "cancelled"  // User cancelled
)
```

#### State Transitions
```
[Created] → PENDING → RUNNING → COMPLETED
                        ↓           ↑
                    CANCELLED   FAILED
```

### Progress Tracking

Jobs maintain rich state information for real-time UI updates:

```go
type JobState struct {
    Stage            string         `json:"stage"`              
    StageStartedAt   time.Time      `json:"stage_started_at"`   
    CurrentOperation string         `json:"current_operation"`   
    CurrentItem      string         `json:"current_item"`       
    Progress         JobProgress    `json:"progress"`           
    Context          JobContext     `json:"context"`            
    Timeline         []JobStageInfo `json:"timeline"`           
    Stats            JobStats       `json:"stats"`              
    Messages         []string       `json:"messages"`           
}

// Enhanced progress features:
// - Per-list progress with substates (metadata, permissions, items)
// - Item count-based progress using SharePoint ItemCount
// - List filtering visibility (hidden vs processed counts)
// - Per-link sharing audit progress
// - Context cancellation support
```

## Audit Run Scoping Architecture

The application implements audit run scoping for historical data access and isolation:

### AuditRunScopedServiceFactory

Creates services automatically filtered to specific audit runs.

```go
// application/audit_run_scoped_service_factory.go
type AuditRunScopedServiceFactory interface {
    CreateForAuditRun(ctx context.Context, siteID int64, auditRunIDStr string) (*AuditRunScopedServices, error)
}

type AuditRunScopedServices struct {
    SiteContentService  *SiteContentService
    PermissionService   *PermissionService
    SiteBrowsingService *SiteBrowsingService
    AuditRunID          int64
}
```

**Key Features:**
- **"Latest" Resolution**: Converts "latest" string to actual audit run ID
- **Automatic Filtering**: All repository operations scoped to specific audit run
- **Historical Access**: Compare data across different audit runs
- **Data Isolation**: Prevents cross-audit data contamination

### Scoped vs Unscoped Services

```go
// Unscoped - operates on all data
siteService := application.NewSiteContentService(aggregateRepo)

// Scoped - operates only on specific audit run data
scopedServices, err := serviceFactory.CreateForAuditRun(ctx, siteID, "latest")
siteData, err := scopedServices.SiteContentService.GetSiteWithLists(ctx, siteID)
```

### Handler Integration

```go
// interfaces/web/handlers/list_handlers.go
func (h *ListHandlers) SiteListsPage(w http.ResponseWriter, r *http.Request) {
    siteID := extractSiteID(r)
    auditRunID := extractAuditRunID(r) // "latest" or specific ID
    
    // Create audit-run-scoped services
    scopedServices, err := h.serviceFactory.CreateForAuditRun(ctx, siteID, auditRunID)
    
    // All operations automatically scoped to audit run
    data, err := scopedServices.SiteContentService.GetSiteWithLists(ctx, siteID)
}
```

## Component Architecture

### Domain Services

Pure business logic with no external dependencies.

#### Content Analysis Service
Categorizes SharePoint content by file type and calculates risk metrics:

```go
// domain/sharepoint/content_service.go
type ContentService struct {
    sensitiveExtensions  []string // .key, .pem, .config, etc.
    documentExtensions   []string // .docx, .pdf, .txt, etc.
    executableExtensions []string // .exe, .msi, .bat, etc.
}

func (s *ContentService) AnalyzeItems(items []*Item) *ContentAnalysis {
    analysis := &ContentAnalysis{
        TotalItems:       int64(len(items)),
        FilesByExtension: make(map[string]int64),
    }
    
    for _, item := range items {
        if item.HasUnique {
            analysis.ItemsWithUnique++
        }
        
        if item.IsDocument() {
            analysis.FilesCount++
            s.analyzeFile(item, analysis)
        } else if item.IsDirectory() {
            analysis.FoldersCount++
        }
    }
    
    // Business calculations
    analysis.MostCommonTypes = s.getMostCommonFileTypes(analysis.FilesByExtension)
    
    return analysis
}

func (s *ContentService) AssessContentRisk(analysis *ContentAnalysis) *ContentRiskAssessment {
    riskScore := s.calculateContentRiskScore(analysis)
    riskFactors := s.identifyContentRiskFactors(analysis)
    
    return &ContentRiskAssessment{
        RiskScore:            riskScore,
        RiskLevel:            s.determineContentRiskLevel(riskScore),
        RiskFactors:          riskFactors,
        Recommendations:      s.generateContentRecommendations(analysis),
        SensitiveFilesCount:  s.countSensitiveFiles(analysis.FilesByExtension),
        ExecutableFilesCount: s.countExecutableFiles(analysis.FilesByExtension),
    }
}
```

#### Implementation Details
- **File Type Categorization**: Maps extensions to categories (document, image, archive, executable, sensitive)
- **Risk Scoring**: Calculates 0-100 score using item counts and file type distribution
- **Recommendation Generation**: Returns string arrays based on threshold checks
- **Extension Matching**: Uses string slices to match file extensions against predefined lists

### Repository Pattern with Base Repository

```go
// infrastructure/repositories/base_repository.go
type BaseRepository struct{}

func (b *BaseRepository) FromNullString(ns sql.NullString) string {
    if !ns.Valid {
        return ""
    }
    return ns.String
}

func (b *BaseRepository) ToNullString(s string) sql.NullString {
    return sql.NullString{String: s, Valid: s != ""}
}
```

### Presenter Pattern

```go
// interfaces/web/presenters/site_presenter.go
type SitePresenter struct{}

func (p *SitePresenter) ToSiteListsViewModel(data *application.SiteWithListsData) *SiteListsViewModel {
    return &SiteListsViewModel{
        Site: SiteWithMetadata{
            SiteID:        data.Site.ID,
            Title:         data.Site.Title,
            DaysAgo:       data.LastAuditDaysAgo,        // UI calculation
            LastAuditDate: p.formatDate(data.LastAuditDate), // UI formatting
        },
        Lists:           p.formatListSummaries(data.Lists),
        TotalLists:      data.TotalLists,               // Business metric
        ListsWithUnique: data.ListsWithUnique,          // Business metric
    }
}
```

## Database Design

### Audit Run Tracking & Historical Data

The application implements audit run tracking for historical data:

```sql
-- Each audit run gets unique tracking
CREATE TABLE audit_runs (
    audit_run_id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id TEXT NOT NULL REFERENCES jobs(id),
    started_at DATETIME NOT NULL,
    completed_at DATETIME,
    site_url TEXT NOT NULL
);

-- Sites table (simplified, not audit-run scoped)
CREATE TABLE sites (
    site_id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_url TEXT NOT NULL UNIQUE,
    title TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Lists are audit-run scoped for historical tracking
CREATE TABLE lists (
    list_id TEXT NOT NULL,
    site_id INTEGER NOT NULL REFERENCES sites(site_id),
    title TEXT NOT NULL,
    item_count INTEGER NOT NULL DEFAULT 0,
    is_document_library BOOLEAN NOT NULL DEFAULT FALSE,
    audit_run_id INTEGER NOT NULL REFERENCES audit_runs(audit_run_id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (list_id, site_id, audit_run_id)
);

-- Items are audit-run scoped
CREATE TABLE items (
    item_id TEXT NOT NULL,
    list_id TEXT NOT NULL,
    site_id INTEGER NOT NULL REFERENCES sites(site_id),
    name TEXT,
    url TEXT,
    has_unique_permissions BOOLEAN NOT NULL DEFAULT FALSE,
    audit_run_id INTEGER NOT NULL REFERENCES audit_runs(audit_run_id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (item_id, list_id, site_id, audit_run_id)
);
```

**Schema Characteristics:**
- **Composite Keys**: Lists and items use (id, site_id, audit_run_id) primary keys
- **Temporal Data**: All audit-scoped tables store created_at timestamps
- **Foreign Key Constraints**: audit_run_id references audit_runs table
- **Query Patterns**: Latest data accessed via ORDER BY audit_run_id DESC LIMIT 1

### SQLC Integration

Database queries using SQLC code generation:

```sql
-- database/queries/sites.sql
-- name: GetSiteByURL :one
SELECT site_id, site_url, title 
FROM sites 
WHERE site_url = sqlc.arg(site_url) 
ORDER BY audit_run_id DESC 
LIMIT 1;

-- name: UpsertSite :one
INSERT INTO sites (site_id, site_url, title, audit_run_id) 
VALUES (sqlc.arg(site_id), sqlc.arg(site_url), sqlc.arg(title), sqlc.arg(audit_run_id))
ON CONFLICT(site_id, audit_run_id) DO UPDATE SET
  title=excluded.title
RETURNING site_id;
```

Generated type-safe Go code:
```go
// gen/db/sites.sql.go (generated by SQLC)
func (q *Queries) GetSiteByURL(ctx context.Context, siteUrl string) (Site, error) {
    row := q.db.QueryRowContext(ctx, getSiteByURL, siteUrl)
    var i Site
    err := row.Scan(&i.SiteID, &i.SiteUrl, &i.Title)
    return i, err
}
```

## Development Patterns

### Adding New Features

Follow the **6-Phase Development Pattern**:

#### Phase 1: Domain Entities
```go
// domain/sharepoint/content_type.go
type ContentType struct {
    ID          string
    SiteID      int64
    Name        string
    Description string
}

func (ct *ContentType) IsSystemContentType() bool {
    return strings.HasPrefix(ct.ID, "0x01")
}
```

#### Phase 2: Repository Contract
```go
// domain/contracts/content_type_repository.go
type ContentTypeRepository interface {
    SaveContentType(ctx context.Context, auditRunID int64, contentType *sharepoint.ContentType) error
    GetContentTypesForSite(ctx context.Context, siteID int64) ([]*sharepoint.ContentType, error)
}
```

#### Phase 3: Database Schema & Queries
```sql
-- database/migrations/001_schema.sql
CREATE TABLE content_types (
  site_id         INTEGER NOT NULL REFERENCES sites(site_id),
  content_type_id TEXT NOT NULL,
  name            TEXT,
  audit_run_id    INTEGER REFERENCES audit_runs(audit_run_id),
  PRIMARY KEY (site_id, content_type_id, audit_run_id)
);

-- database/queries/content_types.sql
-- name: UpsertContentType :exec
INSERT INTO content_types (site_id, content_type_id, name, audit_run_id)
VALUES (sqlc.arg(site_id), sqlc.arg(content_type_id), sqlc.arg(name), sqlc.arg(audit_run_id))
ON CONFLICT(site_id, content_type_id, audit_run_id) DO UPDATE SET name=excluded.name;
```

#### Phase 4: Repository Implementation
```bash
# Generate SQLC code
sqlc generate

# Implement repository
# infrastructure/repositories/sqlc_content_type_repository.go
```

#### Phase 5: Application Service
```go
// application/content_type_service.go
type ContentTypeService struct {
    siteRepo        contracts.SiteRepository
    contentTypeRepo contracts.ContentTypeRepository
}

func (s *ContentTypeService) GetContentTypesForSite(ctx context.Context, siteID int64) (*ContentTypesData, error) {
    // Business logic composition
}
```

#### Phase 6: Handler, Presenter & Templates
```go
// interfaces/web/handlers/content_type_handlers.go
// interfaces/web/presenters/content_type_presenter.go
// interfaces/web/templates/components/content_types.templ
```

### Testing Strategy by Layer

**Domain Layer**: No tests needed (pure data structures)

**Application Services**: Unit tests with mocked repositories
```go
func TestSiteContentService_GetSiteWithLists_Success(t *testing.T) {
    // Arrange: Mock dependencies
    mockSiteRepo := &mocks.MockSiteRepository{}
    mockListRepo := &mocks.MockListRepository{}
    
    service := NewSiteContentService(mockSiteRepo, mockListRepo, ...)
    
    // Act: Test business logic
    result, err := service.GetSiteWithLists(ctx, siteID)
    
    // Assert: Verify business calculations
    assert.Equal(t, 1, result.ListsWithUnique) // Business rule verification
}
```

**Infrastructure Layer**: Integration tests with real database

**Interface Layer**: HTTP tests with mocked services

### Dependency Injection

All dependencies are wired in `cmd/server/main.go`:

```go
func main() {
    // Infrastructure layer
    db := database.New(dbPath)
    spClient := infrastructure.NewSharePointClientFromEnv()
    
    // Individual repositories
    siteRepo := infrastructure.NewSqlcSiteRepository(db)
    listRepo := infrastructure.NewSqlcListRepository(db)
    itemRepo := infrastructure.NewSqlcItemRepository(db)
    auditRepo := infrastructure.NewSqlcAuditRepository(db)
    jobRepo := infrastructure.NewSqlcJobRepository(db)
    
    // Aggregate repositories (combining multiple repos)
    siteContentAggregate := infrastructure.NewSiteContentAggregateRepository(
        siteRepo, listRepo, itemRepo, auditRepo)
    
    // Application layer services
    jobService := application.NewJobServiceImpl(jobRepo, auditRepo, executorRegistry, eventBus)
    auditService := application.NewAuditService(auditRepo, siteRepo, jobService)
    
    // Audit-run-scoped service factory
    serviceFactory := application.NewAuditRunScopedServiceFactory(
        siteContentAggregate, auditRepo)
    
    // Platform layer
    auditWorkflowFactory := platform.NewAuditWorkflowFactory(spClient, siteRepo, listRepo)
    siteAuditExecutor := platform.NewSiteAuditExecutor(auditWorkflowFactory)
    executorRegistry := platform.NewJobExecutorRegistry()
    executorRegistry.Register("site_audit", siteAuditExecutor)
    
    // Interface layer
    listPresenter := presenters.NewListPresenter()
    listHandlers := handlers.NewListHandlers(serviceFactory, listPresenter)
    
    // Setup routes and start server
    router := setupRoutes(listHandlers, auditHandlers, jobHandlers)
    log.Fatal(http.ListenAndServe(":8080", router))
}
```

## Architectural Decisions

### Layered Architecture
- Clear layer boundaries with single responsibility
- Fast unit tests without external dependencies
- New interfaces easily added
- Developers can work on layers independently

### BaseRepository Pattern
- Shared SQL type conversion utilities
- Common database operations
- Type safety without boilerplate

### Event-Driven Job System
- Job execution separate from UI concerns
- New notification types added via event handlers
- Event handler failures don't affect job execution
- UI updates happen as events occur

### SQLC + SQLite
- Generated queries prevent SQL errors
- SQLite handles read-heavy audit data with WAL mode
- No external database server needed
- Migration to PostgreSQL supported

This architecture supports the SharePoint audit system with separated concerns, error handling, and extensibility.