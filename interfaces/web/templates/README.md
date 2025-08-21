# Frontend Architecture

This document outlines the frontend architecture for the SharePoint Audit application, built with [Templ](https://templ.guide/) and [HTMX](https://htmx.org/).

## Architecture Overview

The frontend follows a component-based architecture with server-side rendering and progressive enhancement via HTMX. The design emphasizes:

- **Type-safe templates** using Templ
- **Progressive enhancement** with HTMX
- **Component reusability** and modularity
- **Server-side rendering** with dynamic updates

## Directory Structure

```
web/
├── components/           # Reusable template components
│   ├── ui/              # Generic UI components
│   ├── sharepoint/      # Domain-specific components
│   └── core/            # Layout and HTMX utilities
├── pages/               # Full page templates
└── assets/              # Static assets (CSS, JS)
```

### Component Organization

#### `components/ui/` - Generic UI Components
Reusable interface elements that could work in any application:
- `alerts.templ` - Alert/notification components with variants
- `badges.templ` - Status badges and labels
- `expandable.templ` - Toggle buttons and expandable rows

#### `components/sharepoint/` - Domain Components
SharePoint-specific business logic components:
- `principal.templ` - User/group icons and representations
- `permission_help.templ` - Permission explanation components
- `assignment_help.templ` - Assignment context helpers

#### `components/core/` - Core Infrastructure
Application layout and HTMX utilities:
- `layout.templ` - Main page layout and navigation
- `tabs.templ` - Tab navigation components
- `htmx.templ` - HTMX configuration and helper components

#### `pages/` - Page Templates
Complete page templates that compose multiple components:
- `index.templ` - Dashboard page
- `list_*.templ` - List detail pages and tabs
- `assignments.templ` - Assignment views

## HTMX Patterns

### Global Configuration
The application uses centralized HTMX configuration in `core/htmx.templ`:

```go
templ HTMXConfig() {
    // Global error handling, timeouts, and navigation boost
    htmx.config.defaultSwapStyle = 'innerHTML';
    htmx.config.globalViewTransitions = true;
    // ... error handlers and loading states
}
```

### Component Helpers
Standardized HTMX components for consistency:

```go
// Forms with automatic loading states
@core.HTMXForm("/audit", "audit-status", "post", "audit-ind") {
    // form content
}

// Buttons with integrated indicators
@core.HTMXButton("/api/endpoint", "target", "post", "loading", "Click Me", "btn-class")
```

### Server-Side Search
Search functionality uses server-side filtering:

```html
<input hx-get="/lists/search" 
       hx-target="#lists-table tbody"
       hx-trigger="input changed delay:300ms" />
```

## Contributing Components

### Creating New Components

1. **Choose the right directory**:
   - `ui/` for generic, reusable components
   - `sharepoint/` for domain-specific components
   - `core/` for infrastructure components

2. **Follow naming conventions**:
   ```go
   // Component names should be descriptive and follow PascalCase
   templ StatusBadge(status string, variant string) { ... }
   templ PrincipalIcon(principalType int32) { ... }
   ```

3. **Use variant patterns for flexibility**:
   ```go
   templ Alert(variant string, title string, content templ.Component) {
       switch variant {
       case "info": // blue styling
       case "warning": // amber styling
       case "success": // green styling
       }
   }
   ```

### Component Guidelines

#### Make Components Composable
```go
// Good - accepts content as a component
templ Card(title string, content templ.Component) {
    <div class="card">
        <h3>{ title }</h3>
        @content
    </div>
}

// Usage
@Card("My Title") {
    <p>Card content here</p>
}
```

#### Use Consistent Class Patterns
```go
// Follow Tailwind utility patterns
templ Button(variant string, text string) {
    switch variant {
    case "primary":
        <button class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700">
    case "secondary": 
        <button class="px-4 py-2 bg-slate-200 text-slate-900 rounded-lg hover:bg-slate-300">
    }
}
```

#### Include HTMX Attributes
```go
// Components should accept HTMX configuration
templ SearchInput(endpoint string, target string) {
    <input type="search"
           hx-get={ endpoint }
           hx-target={ target }
           hx-trigger="input changed delay:300ms"
           class="border rounded-lg px-3 py-2" />
}
```

### Creating New Pages

1. **Use the standard layout**:
   ```go
   templ MyPage(data MyData) {
       @core.Layout("Page Title") {
           // page content
       }
   }
   ```

2. **Compose from existing components**:
   ```go
   templ ListDetailPage(list List) {
       @core.Layout(list.Title) {
           @ui.Alert("info", "Notice") {
               <p>This list has unique permissions.</p>
           }
           
           @sharepoint.PrincipalList(list.Principals)
       }
   }
   ```

3. **Implement tab patterns for complex pages**:
   ```go
   // Use the tab shell pattern
   @pages.ListShell(list, "overview", pages.ListOverviewTab(list))
   ```

## Best Practices

### HTMX Integration
- Use `hx-boost="true"` for progressive navigation enhancement
- Implement proper loading indicators with `hx-indicator`
- Handle errors with global event listeners
- Use `hx-swap="outerHTML"` for replacing entire components

### Performance
- Server-side rendering ensures fast initial loads
- HTMX provides smooth updates without full page refreshes
- Components are rendered on-demand via HTMX requests

### Accessibility
- Use semantic HTML elements
- Include proper ARIA labels where needed
- Ensure keyboard navigation works with HTMX interactions

### Testing Components
```bash
# Generate templates and build
templ generate
go build ./cmd/server

# Test in browser
./server
```

## File Generation

Templates are automatically generated using:
```bash
templ generate
```

This creates `*_templ.go` files that should not be edited manually.

## Styling

The application uses:
- **Tailwind CSS** for utility-first styling
- **Component classes** defined in `/assets/css/components.css`
- **Consistent color palette** (slate, blue, amber, green)

Components should use Tailwind utilities and avoid inline styles where possible.