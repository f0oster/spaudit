// Package handlers render provides HTTP response and HTMX utilities.
package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
)

// RenderResponse renders Templ components to HTTP responses.
func RenderResponse(ctx context.Context, w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(ctx, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// IsHTMXRequest checks if the request came from HTMX.
func IsHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// IsHTMXPartialRequest checks if this is a targeted HTMX partial update.
func IsHTMXPartialRequest(r *http.Request) bool {
	return IsHTMXRequest(r) && r.Header.Get("HX-Target") != ""
}

// GetHTMXTarget returns the HTMX target element ID.
func GetHTMXTarget(r *http.Request) string {
	target := r.Header.Get("HX-Target")
	// Remove # prefix if present
	return strings.TrimPrefix(target, "#")
}
