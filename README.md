# SharePoint Audit Tool

A SharePoint permissions auditing application for analyzing site permissions, sharing links, and security risks.

## ⚠️ Development Status

**This project is in very early stages of development**

- **No accuracy guarantees**: Audit results may contain errors or omissions
- **Breaking changes expected**: Database schema may change without notice
- **Data loss possible**: Database migrations are very unlikely preserve existing audit data
- **SharePoint Online only**: Only tested with SharePoint Online, and no plans to support on-prem SharePoint
- **Incomplete features**: Many components contain placeholder logic or are unfinished (content analysis, risk assessment, reporting)
- **Use at your own risk**: Not recommended for production security decisions

The project is suitable for testing only until it reaches a stable release.

## Quick Start

### Prerequisites
- **Go 1.25+ installed**
- **Mage build tool** - Install with `go install github.com/magefile/mage@latest`
- **SharePoint admin access**  
- **Entra ID app registration** with SharePoint permissions
- **Valid certificate (.pfx file)**

### Setup

#### 1. Create Entra ID App Registration
```bash
# Create app registration in Azure portal with:
# - Name: <your-app-name>
# - API Permissions: SharePoint -> Sites.FullControl.All
# - Authentication: Certificate (upload your .cer public certificate)
# - Copy Tenant ID and Application (Client) ID
```

#### 2. Clone and build
```bash
git clone https://github.com/f0oster/spaudit && cd spaudit
mage bootstrap
```

#### 3. Configure environment
```bash
cd cmd/server
cp ../../.env.example .env
# Edit .env with your required values:
# SP_TENANT_ID=your-tenant-id
# SP_CLIENT_ID=your-client-id  
# SP_CERT_PATH=./path/to/cert.pfx
```

#### 4. Build and start server
```bash
cd ..
mage build
cd cmd/server
./server.exe
```

#### 5. Open http://localhost:8080

## Screenshots

### Dashboard
![Dashboard](./docs/screenshots/dashboard.png?raw=true "Main dashboard showing audit overview and recent jobs")

### Site Lists Overview  
![Site Lists](./docs/screenshots/site_lists.png?raw=true "Site lists view showing permission summaries")

### Document Library Overview
![Library Overview](./docs/screenshots/site_documentlibrary_overview.png?raw=true "Document library overview with metadata and stats")

### Document Library Permissions
![Library Permissions](./docs/screenshots/site_documentlibrary_listperms.png?raw=true "Document library permission assignments")

### Custom Item Permissions
![Custom Item Permissions](./docs/screenshots/site_documentlibrary_customitemperms.png?raw=true "Individual items with unique permissions")

### Sharing Links Analysis
![Sharing Links](./docs/screenshots/site_documentlibrary_sharinglinks.png?raw=true "Sharing links governance and analysis")

## What It Does

This tool audits SharePoint sites to discover:
- **Permission assignments** - Who has access to what
- **Sharing links** - External sharing and link governance
- **Content analysis** - Files, folders, and sensitivity labels
- **Security risks** - Excessive permissions and exposure

## Basic Usage

### Running an Audit
1. Visit `http://localhost:8080`
2. Enter SharePoint site URL
3. Configure audit options:
   - **Individual Item Scanning**: Deep scan files and folders for unique permissions
   - **Sharing Link Analysis**: Analyze external sharing links and governance
   - **Skip Hidden Items**: Ignore system lists and hidden content
4. Click "Start Audit"

### Viewing Results
- **Dashboard**: Site overview and recent audits
- **Lists**: Browse site lists with permission summaries
- **Items**: View individual files/folders with detailed permissions
- **Sharing Links**: Review external sharing and access controls
- **Jobs**: Monitor audit progress and history

## Configuration

### Environment Variables
```bash
# SharePoint Authentication
SP_TENANT_ID=your-tenant-id
SP_CLIENT_ID=your-client-id
SP_CERT_PATH=./certificates/cert.pfx
SP_CERT_PASSWORD=password            # if certificate is password-protected

# Application
HTTP_ADDR=:8080                      # server address
DB_PATH=./spaudit.db                 # database location
LOG_LEVEL=info                       # debug, info, warn, error
```

### Audit Parameters
- **Batch Size**: Items processed per API call (default: 100)
- **Timeout**: Maximum audit duration in seconds (default: 1800)
- **Max Retries**: Retry attempts for failed operations (default: 3)

## Architecture Overview

The application follows clean architecture patterns with clear separation of concerns:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Web UI        │    │  Background      │    │  SharePoint     │
│ (HTMX/Tailwind) │◄──►│  Job System      │◄──►│  API Client     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Application    │    │   Domain         │    │  Infrastructure │
│  Services       │◄──►│   Models         │◄──►│  Repositories   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                     ┌──────────────────┐
                     │  SQLite Database │
                     └──────────────────┘
```

### Key Components
- **Domain Layer**: Core business entities (sites, lists, items, jobs)
- **Application Layer**: Business logic services
- **Infrastructure Layer**: Database access, SharePoint client, audit engines
- **Interface Layer**: Web handlers, templates, API endpoints
- **Platform Layer**: Background job processing and workflows

## How It Works

### Audit Process
1. **Site Discovery**: Connect to SharePoint and discover site structure
2. **List Processing**: Scan each list for metadata, permissions, and items
3. **Item Analysis**: Deep scan files/folders for unique permissions (if enabled)  
4. **Sharing Analysis**: Discover and analyze sharing links (if enabled)
5. **Results**: Store audit results with timestamps for historical comparison

### Job System
- **Background Processing**: Long-running audits don't block the web interface
- **Real-time Progress**: Live updates via Server-Sent Events
- **Cancellation**: Stop running audits with proper cleanup
- **Job History**: Track audit history and performance metrics

### Database Design
- **Audit Runs**: Each audit creates an immutable snapshot with unique `audit_run_id`
- **Historical Data**: Compare security posture changes over time
- **Performance Tracking**: Monitor audit execution and coverage metrics

## Development

### Building
```bash
# Bootstrap development environment (installs tools and runs generators)
mage bootstrap

# Build server
mage build

# Run tests
mage test

# Run all checks (formatting, linting, tests, build)
mage verify
```

#### Manual Commands
```bash
# Generate database queries and templates
mage gen

# Run specific tasks
mage lint        # Run linters
mage cover       # Generate coverage report
mage vuln        # Check for vulnerabilities
```

### Project Structure
```
spaudit/
├── cmd/server/           # Entry point
├── domain/               # Domain entities, contracts
├── application/          # Services/Application logic
├── infrastructure/       # Database, SharePoint client, repositories
├── interfaces/web/       # HTTP handlers, templates, presenters
├── platform/             # Background jobs, workflows
├── database/             # Schema migrations and queries
└── gen/db/               # Generated database code
```

## Dependencies

### Backend
- **Go 1.25**
- **SQLite** (modernc.org/sqlite) - Database
- **SQLC** - Type-safe SQL code generation
- **Chi** (github.com/go-chi/chi/v5) - HTTP router
- **Gosip** (github.com/koltyakov/gosip) - SharePoint client
- **Testify** (github.com/stretchr/testify) - Testing

### Frontend
- **Templ** (github.com/a-h/templ) - HTML templates
- **HTMX** - Frontend interactivity
- **Tailwind CSS** - Styling

## Acknowledgments

This project makes heavy use of the [Gosip](https://github.com/koltyakov/gosip) SharePoint API client library. Many thanks to [Andrew Koltyakov](https://github.com/koltyakov) and all the Gosip contributors for providing a robust, well-maintained Go library that makes SharePoint integration possible.

The Gosip library handles the SharePoint authentication and API interactions that form the foundation of this audit tool.

## License

MIT License - see [LICENSE](LICENSE) file for details.

---

For detailed architecture documentation, see [ARCHITECTURE.md](docs/ARCHITECTURE.md).  