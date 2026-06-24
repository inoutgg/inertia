package inertia

import (
	"fmt"
	"html/template"
	"io/fs"

	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/must"

	"go.segfaultmedaddy.com/inertia/internal/inertiahttp"
)

// DefaultRootViewID is the default root HTML element ID to which
// the Inertia.js app is mounted.
const DefaultRootViewID = inertiahttp.DefaultRootViewID

// DefaultConcurrency is the default maximum concurrency for the ResultPool
// used to resolve concurrent props. A value of 0 means no limit (pond's "0 = unlimited"
// convention).
var DefaultConcurrency = inertiahttp.DefaultConcurrency //nolint:gochecknoglobals

// ErrInvalidInertiaRequest is the sentinel error returned (or wrapped) when an
// incoming Inertia request has malformed or invalid protocol headers.
// Callers can match it with errors.Is to distinguish protocol errors from
// other parse failures.
var ErrInvalidInertiaRequest = inertiahttp.ErrInvalidInertiaRequest

// Config configures the Renderer behavior and capabilities.
type Config = inertiahttp.Config

// Renderer handles Inertia.js page responses, supporting both client-side and server-side rendering.
// It manages HTML template rendering, JSON serialization, and prop resolution.
//
// Create a Renderer using New or FromFS constructor functions.
type Renderer = inertiahttp.Renderer

// New creates a Renderer with the provided HTML template and configuration.
//
// If config is nil, default values are used; see Config for details.
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
