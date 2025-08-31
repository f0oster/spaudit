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

## Quick Start

### Prerequisites
- Go 1.25+ installed
- SharePoint admin access  
- Valid certificate (.pfx file)

### Setup
```bash
# 1. Clone and build
git clone <repo> && cd spaudit
go mod tidy && go build ./cmd/server

# 2. Configure environment
cd cmd/server
cp .env.example .env
# Edit these required values:
# SP_TENANT_ID=your-tenant-id
# SP_CLIENT_ID=your-client-id  
# SP_CERT_PATH=./path/to/cert.pfx

# 3. Initialize database and start server
./server.exe -migrate
./server.exe

# 4. Open http://localhost:8080
```

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
│   (HTMX/HTMX)   │◄──►│  Job System      │◄──►│  API Client     │
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

## Common Tasks

### Troubleshooting Failed Audits
```bash
# Check recent job status
sqlite3 spaudit.db "SELECT id, status, error, created_at FROM jobs ORDER BY created_at DESC LIMIT 5;"

# Enable debug logging
export LOG_LEVEL=debug
./server.exe

# Check SharePoint connectivity
# Look for "spclient" entries in logs for API responses
```

### Performance Tuning
- **Large Sites**: Reduce batch size to 50-100 items
- **API Timeouts**: Increase timeout for sites with >10,000 items  
- **Memory Usage**: Avoid running multiple concurrent audits
- **Database**: Use SSD storage for better SQLite performance

### Certificate Issues
```bash
# Verify certificate path and permissions
ls -la ./certificates/cert.pfx

# Test certificate (if password-protected)
openssl pkcs12 -in cert.pfx -noout
```

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

## API Reference

### Key Endpoints
- `GET /` - Dashboard
- `POST /audit` - Start site audit
- `GET /jobs` - Job list and status
- `GET /sites/{id}/lists` - Site content browser
- `GET /events` - Server-Sent Events for real-time updates

### Real-time Updates
The UI automatically updates using Server-Sent Events:
- Job progress updates
- Completion notifications
- Error alerts

## Troubleshooting

### SharePoint Authentication
- **Certificate not found**: Verify `SP_CERT_PATH` points to valid .pfx file
- **Permission errors**: Ensure service principal has correct SharePoint permissions
- **Authentication failures**: Check `SP_TENANT_ID` and `SP_CLIENT_ID` match Azure app registration

### Database Issues
- **Migration failures**: Check database file write permissions
- **Lock timeouts**: Restart server, avoid concurrent access
- **Corruption**: Delete database file and re-run with `-migrate` flag

### Performance Issues  
- **Slow audits**: Adjust batch size for large sites
- **Memory usage**: Limit concurrent audits
- **API timeouts**: Increase timeout for sites with many items

### Development Issues
- **Template errors**: Run `templ generate` after .templ changes
- **Database errors**: Run `sqlc generate` after schema/query changes
- **Build failures**: Run `go mod tidy` to resolve dependencies

## Technology Stack

- **Go 1.25**: Backend language
- **SQLite**: Database with SQLC for type-safe queries  
- **HTMX + Tailwind**: Frontend framework
- **Templ**: Type-safe HTML templates
- **Server-Sent Events**: Real-time UI updates
- **SharePoint REST API**: Certificate-based authentication

## Acknowledgments

This project makes heavy use of the [Gosip](https://github.com/koltyakov/gosip) SharePoint API client library. Many thanks to [Andrew Koltyakov](https://github.com/koltyakov) and all the Gosip contributors for providing a robust, well-maintained Go library that makes SharePoint integration possible.

The Gosip library handles the SharePoint authentication and API interactions that form the foundation of this audit tool.

## License

MIT License - see [LICENSE](LICENSE) file for details.

---

For detailed architecture documentation, see [ARCHITECTURE.md](docs/ARCHITECTURE.md).  