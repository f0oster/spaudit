# SharePoint Audit Tool - Architecture Documentation

This document provides detailed technical architecture information for developers who need to understand the internal design and implementation patterns.

## Table of Contents
- [Architecture Principles](#architecture-principles)
- [Layer Details](#layer-details)
- [Data Flow Architecture](#data-flow-architecture)  
- [Job System Architecture](#job-system-architecture)
- [Component Architecture](#component-architecture)
- [Database Design](#database-design)
- [Development Patterns](#development-patterns)

## Architecture Principles

The project follows Clean Architecture patterns with strict dependency inversion:

### Core Principles
- **Dependency Direction**: All dependencies point inward toward the domain
- **Business Logic Isolation**: Pure business logic with no external dependencies
- **Interface Segregation**: Repository interfaces define clear contracts
- **Composition over Inheritance**: Services compose repositories rather than inherit

### Layer Responsibilities
```
┌─────────────────────────────────────────────────────────────┐
│                     External Systems                        │
│              (SharePoint API, SQLite, HTTP)                 │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                Interface Layer                              │
│      HTTP Handlers, Templates, Presenters                   │
│         (interfaces/web/)                                   │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                Platform Layer                               │
│      Workflows, Job Executors, Factories                    │
│            (platform/)                                      │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│               Application Layer                             │
│         Business Logic Services                             │
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

Business logic orchestration services that compose repositories.

```go
// application/site_content_service.go
type SiteContentService struct {
    siteRepo       contracts.SiteRepository
    listRepo       contracts.ListRepository
    itemRepo       contracts.ItemRepository
    jobRepo        contracts.JobRepository
}

func (s *SiteContentService) GetSiteWithLists(ctx context.Context, siteID int64) (*SiteWithListsData, error) {
    // 1. Compose data from multiple repositories
    site, err := s.siteRepo.GetByID(ctx, siteID)
    lists, err := s.listRepo.GetAllForSite(ctx, siteID)
    lastAuditDate, err := s.jobRepo.GetLastAuditDate(ctx, siteID)
    
    // 2. Apply business calculations
    return s.calculateBusinessMetrics(site, lists, lastAuditDate), nil
}
```

### Infrastructure Layer (`infrastructure/`)

External system implementations that fulfill domain contracts.

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

### Interface Layer (`interfaces/web/`)

HTTP communication with clean separation of concerns.

```go
// interfaces/web/handlers/site_handlers.go
type SiteHandlers struct {
    siteContentService *application.SiteContentService
    sitePresenter      *presenters.SitePresenter
}

func (h *SiteHandlers) GetSiteWithLists(w http.ResponseWriter, r *http.Request) {
    // 1. Extract HTTP parameters
    siteID := extractSiteID(r)

    // 2. Call application service (business logic)
    data, err := h.siteContentService.GetSiteWithLists(r.Context(), siteID)

    // 3. Transform to view model (presentation logic)
    viewModel := h.sitePresenter.ToSiteListsViewModel(data)

    // 4. Render response
    templates.SiteListsView(viewModel).Render(r.Context(), w)
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

The job system provides robust background processing for long-running SharePoint audits.

### Core Design Principles
- **Single Responsibility**: Each component has a focused role in the job lifecycle
- **Event-Driven**: Domain events decouple job execution from UI updates
- **Linear Execution**: Simplified flow eliminates complex callback chains
- **Context Cancellation**: Proper goroutine cleanup and responsive termination

### Job System Components

#### 1. Job Service (`application/job_service_impl.go`)

Central orchestrator with context cancellation support.

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

Type-specific execution engines implementing the `JobExecutor` interface.

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

Type-safe event publishing and subscription with panic recovery.

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

## Component Architecture

### Domain Services

Pure business logic with no external dependencies.

```go
// domain/sharepoint/content_service.go
type ContentService struct{}

func (s *ContentService) AnalyzeItems(items []*Item) *ContentAnalysis {
    analysis := &ContentAnalysis{
        TotalItems:      len(items),
        ItemsWithUnique: 0,
        FileTypes:       make(map[string]int),
    }
    
    for _, item := range items {
        if item.HasUnique {
            analysis.ItemsWithUnique++
        }
        analysis.FileTypes[item.FileExtension]++
    }
    
    return analysis
}
```

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

The application implements comprehensive audit run tracking for historical data preservation:

```sql
-- Each audit run gets unique tracking
CREATE TABLE audit_runs (
    audit_run_id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id TEXT NOT NULL REFERENCES jobs(id),
    started_at DATETIME NOT NULL,
    completed_at DATETIME,
    site_url TEXT NOT NULL
);

-- All entities tagged with audit_run_id for historical preservation
CREATE TABLE sites (
    site_id INTEGER NOT NULL,
    site_url TEXT NOT NULL,
    title TEXT,
    audit_run_id INTEGER REFERENCES audit_runs(audit_run_id),
    PRIMARY KEY (site_id, audit_run_id)
);
```

**Key Benefits:**
- **Historical Analysis**: Compare security posture changes over time
- **Audit Trail**: Complete record of what was discovered when
- **Performance Tracking**: Monitor audit execution efficiency
- **Rollback Support**: Query previous audit states

### SQLC Integration

Type-safe database queries with SQLC code generation:

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
    siteRepo := infrastructure.NewSqlcSiteRepository(db)
    listRepo := infrastructure.NewSqlcListRepository(db)
    
    // Application layer
    listService := application.NewListService(siteRepo, listRepo, jobRepo)
    
    // Interface layer
    listPresenter := presenters.NewListPresenter()
    listHandlers := handlers.NewListHandlers(listService, listPresenter)
    
    // Setup routes
    router := setupRoutes(listHandlers)
    log.Fatal(http.ListenAndServe(":8080", router))
}
```

## Key Architectural Decisions

### Why Clean Architecture?
- **Maintainability**: Clear layer boundaries with single responsibility
- **Testability**: Fast unit tests without external dependencies
- **Extensibility**: New interfaces (CLI, API) easily added
- **Team Productivity**: Developers can work on layers independently

### Why BaseRepository Pattern?
- Eliminates adapter layer complexity while maintaining clean SQL type conversion
- Shared utilities for common database operations
- Maintains type safety without boilerplate

### Why Event-Driven Job System?
- **Decoupling**: Job execution independent of UI concerns
- **Extensibility**: New notification types easily added via event handlers
- **Resilience**: Event handler failures don't affect job execution
- **Real-time**: UI updates happen immediately as events occur

### Why SQLC + SQLite?
- **Type Safety**: Generated queries prevent runtime SQL errors
- **Performance**: SQLite excellent for read-heavy audit data with WAL mode
- **Simplicity**: No external database server required
- **Flexibility**: Easy migration to PostgreSQL if needed

This architecture provides a solid foundation for the SharePoint audit system while maintaining clean separation of concerns, comprehensive error handling, and excellent extensibility for future capabilities.