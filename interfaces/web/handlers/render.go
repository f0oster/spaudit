package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
)

// RenderResponse handles rendering Templ components with proper HTTP headers.
func RenderResponse(ctx context.Context, w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(ctx, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// IsHTMXRequest checks if the request originates from HTMX.
func IsHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// IsHTMXPartialRequest checks if this is a targeted HTMX partial content request.
func IsHTMXPartialRequest(r *http.Request) bool {
	return IsHTMXRequest(r) && r.Header.Get("HX-Target") != ""
}

// GetHTMXTarget returns the HTMX target element ID with # prefix removed.
func GetHTMXTarget(r *http.Request) string {
	target := r.Header.Get("HX-Target")
	// Remove # prefix if present
	return strings.TrimPrefix(target, "#")
}
