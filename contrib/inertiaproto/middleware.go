package inertiaproto

import (
	"go.inout.gg/foundations/http/httpmiddleware"
	"go.inout.gg/foundations/must"

	"go.segfaultmedaddy.com/inertia"
	"go.segfaultmedaddy.com/inertia/contrib/vite"
)

// MiddlewareConfig configures the Inertia.js HTTP middleware.
type MiddlewareConfig struct {
	// SSRClient enables server-side rendering of Inertia pages; if nil,
	// pages are rendered client-side only.
	SSRClient inertia.SSRClient

	// ViteConfig configures the html/template used for page rendering.
	ViteConfig *vite.Config

	// BundleVersion is the asset version reported to the client via
	// X-Inertia-Version for stale-asset detection.
	BundleVersion string

	// Concurrency sets the maximum number of props that can be resolved
	// concurrently. It only affects props marked as concurrent.
	//
	// Defaults to runtime.GOMAXPROCS(0). A value of 0 means no limit.
	Concurrency int
}

// Option configures the middleware.
type Option func(*MiddlewareConfig)

// WithViteConfig configures the middleware with a vite configuration.
func WithViteConfig(cfg *vite.Config) Option {
	return func(c *MiddlewareConfig) { c.ViteConfig = cfg }
}

// WithSSRClient configures the middleware with an SSR client.
func WithSSRClient(client inertia.SSRClient) Option {
	return func(c *MiddlewareConfig) { c.SSRClient = client }
}

// WithBundleVersion configures the inertia bundle version.
func WithBundleVersion(v string) Option {
	return func(c *MiddlewareConfig) { c.BundleVersion = v }
}

// WithConcurrency sets the maximum number of props that can be resolved
// concurrently. This only affects props marked as concurrent.
//
// A value of 0 uses the default (runtime.GOMAXPROCS(0)) and means no limit.
func WithConcurrency(concurrency int) Option {
	return func(c *MiddlewareConfig) { c.Concurrency = concurrency }
}

// NewMiddleware returns an HTTP middleware that enables Inertia.js protocol
// handling for the given HTML template. Responses are marshalled with protojson
// so protobuf messages follow the canonical JSON mapping.
//
// Wrap the mux returned by App with this middleware to handle asset version
// checks, Inertia request detection, and rendering of component responses.
func NewMiddleware(template string, opts ...Option) httpmiddleware.MiddlewareFunc {
	var config MiddlewareConfig
	for _, opt := range opts {
		opt(&config)
	}

	return inertia.NewMiddleware(inertia.New(
		must.Must(vite.NewTemplate(
			template,
			config.ViteConfig,
		)),
		//nolint:exhaustruct
		&inertia.Config{
			RootViewID:  inertia.DefaultRootViewID,
			Version:     config.BundleVersion,
			SSRClient:   config.SSRClient,
			Concurrency: config.Concurrency,
			JSONOptions: jsonOptions,
		},
	))
}
