package inertia

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"

	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/must"

	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
	"go.segfaultmedaddy.com/inertia/internal/inertiahttp"
	"go.segfaultmedaddy.com/inertia/internal/inertiaredirect"
)

// DefaultRootViewID is the default root HTML element ID to which
// the Inertia.js app is mounted.
const DefaultRootViewID = inertiahttp.DefaultRootViewID

// DefaultConcurrency is the default maximum concurrency for the ResultPool
// used to resolve concurrent props. A value of 0 means no limit (pond's "0 = unlimited"
// convention).
var DefaultConcurrency = inertiahttp.DefaultConcurrency //nolint:gochecknoglobals

// Config configures the Renderer behavior and capabilities.
type Config = inertiahttp.Config

// Renderer handles Inertia.js page responses, supporting both client-side and server-side rendering.
// It manages HTML template rendering, JSON serialization, and prop resolution.
//
// Create a Renderer using New or FromFS constructor functions.
type Renderer = inertiahttp.Renderer

// New creates a Renderer with the provided HTML template and configuration.
//
// If config is nil, default values are used:
//   - RootViewID: "app"
//   - Concurrency: GOMAXPROCS(0)
func New(t *template.Template, config *Config) *Renderer {
	return inertiahttp.New(t, config)
}

// FromFS creates a Renderer by loading an HTML template from a file system.
//
// If config is nil, default values are used.
func FromFS(fsys fs.FS, path string, config *Config) (*Renderer, error) {
	debug.Assert(fsys != nil, "expected fsys to be defined")
	debug.Assert(path != "", "expected path to be defined")

	t := template.New("inertia")

	t, err := t.ParseFS(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("inertia: failed to parse templates: %w", err)
	}

	return New(t, config), nil
}

// MustFromFS is like FromFS, but panics if an error occurs.
func MustFromFS(fsys fs.FS, path string, config *Config) *Renderer {
	return must.Must(FromFS(fsys, path, config))
}

// Location redirects to an external URL outside of the Inertia app.
//
// For Inertia requests, it uses a 409 Conflict response with X-Inertia-Location header.
// For regular requests, it performs a standard HTTP redirect.
func Location(w http.ResponseWriter, r *http.Request, url string) {
	debug.Assert(w != nil, "ResponseWriter must not be nil")
	debug.Assert(r != nil, "Request must not be nil")
	debug.Assert(url != "", "url must be non-empty")

	if r.Header.Get(inertiaheader.HeaderXInertia) == inertiaheader.HeaderValueTrue {
		h := w.Header()

		h.Del(inertiaheader.HeaderVary)
		h.Del(inertiaheader.HeaderXInertia)
		h.Set(inertiaheader.HeaderXInertiaLocation, url) // redirect URL
		w.WriteHeader(http.StatusConflict)               // 409 Conflict

		return
	}

	inertiaredirect.Redirect(w, r, url)
}

// Redirect sends a redirect response to the Inertia app page.
func Redirect(w http.ResponseWriter, r *http.Request, url string) {
	debug.Assert(w != nil, "ResponseWriter must not be nil")
	debug.Assert(r != nil, "Request must not be nil")
	debug.Assert(url != "", "url must be non-empty")

	inertiaredirect.Redirect(w, r, url)
}

// RedirectPreserveFragment redirects while instructing Inertia to preserve the current URL fragment.
func RedirectPreserveFragment(w http.ResponseWriter, r *http.Request, url string) {
	debug.Assert(w != nil, "ResponseWriter must not be nil")
	debug.Assert(r != nil, "Request must not be nil")
	debug.Assert(url != "", "url must be non-empty")

	if r.Header.Get(inertiaheader.HeaderXInertia) == inertiaheader.HeaderValueTrue {
		h := w.Header()

		h.Del(inertiaheader.HeaderVary)
		h.Del(inertiaheader.HeaderXInertia)
		h.Set(inertiaheader.HeaderXInertiaRedirect, url)
		w.WriteHeader(http.StatusConflict)

		return
	}

	inertiaredirect.Redirect(w, r, url)
}

// ErrorBagFromRequest extracts the error bag name from the X-Inertia-Error-Bag header.
//
// Returns the default error bag (empty string) if the header is not present.
// Used to scope validation errors to specific forms on a page.
func ErrorBagFromRequest(r *http.Request) string {
	errorBag := r.Header.Get(inertiaheader.HeaderXInertiaErrorBag)
	if errorBag == "" {
		return DefaultErrorBag
	}

	return errorBag
}
