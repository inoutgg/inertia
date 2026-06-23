// Package inertiaframe provides a high-level, message-oriented API for building Inertia.js applications.
//
// It abstracts Inertia.js protocol details, providing automatic request parsing, validation,
// session management, and response handling. Build type-safe endpoints with minimal boilerplate.
package inertiaframe

import (
	"cmp"
	"fmt"
	"net/http"

	"github.com/go-json-experiment/json"
	"github.com/go-playground/form/v4"
	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/http/httphandler"
	"go.inout.gg/foundations/http/httpmiddleware"

	"go.segfaultmedaddy.com/inertia/inertiaotel"
)

// Config holds application-wide configuration shared across all endpoints
// mounted on an App.
type Config struct {
	// ErrorHandler handles errors returned by mounted endpoints. If nil,
	// DefaultErrorHandler is used.
	ErrorHandler httphandler.ErrorHandler

	// FormDecoder decodes form and multipart request bodies into the message
	// type M. If nil, DefaultFormDecoder is used. Register custom decoders here
	// to support non-standard field types.
	FormDecoder *form.Decoder

	// TelemetryConfig configures OpenTelemetry tracing and metrics for mounted
	// endpoints. If nil, inertiaotel.DefaultConfig is used.
	TelemetryConfig *inertiaotel.Config

	// JSONOptions configures JSON encoding and decoding applied to incoming
	// request bodies and outgoing JSON error responses.
	JSONOptions []json.Options
}

func (c *Config) defaults() {
	c.ErrorHandler = cmp.Or(c.ErrorHandler, DefaultErrorHandler)
	c.FormDecoder = cmp.Or(c.FormDecoder, DefaultFormDecoder)
	c.TelemetryConfig = cmp.Or(c.TelemetryConfig, inertiaotel.DefaultConfig)
}

// MountConfig holds per-endpoint configuration passed to App.Mount.
type MountConfig[M any] struct {
	// Validator validates the parsed request message before the endpoint runs.
	// If nil, no validation is performed and the endpoint executes on any input.
	Validator Validator[M]

	// Middleware wraps the endpoint's HTTP handler in cross-cutting behavior
	// such as authentication, logging, or attaching shared props. Applied in
	// slice order; the first element is the outermost wrapper and runs first on
	// the request path.
	Middleware []Middleware
}

// App is the entry point for mounting Inertia endpoints that share a common
// message type M.
//
// Create an App via New and reuse it to mount multiple endpoints with the same
// Config. To mount endpoints of different message types, create a separate App
// per type and pass the same *Config to share configuration.
type App[M any] struct {
	metrics *metrics
	config  Config
}

// New creates a new App bound to the given Config.
//
// If config is nil, a zero-value Config is used. Sensible defaults are applied
// for any unset fields: DefaultErrorHandler, DefaultFormDecoder, and
// inertiaotel.DefaultConfig for telemetry.
func New[M any](config *Config) *App[M] {
	if config == nil {
		//nolint:exhaustruct
		config = &Config{}
	}

	config.defaults()

	metrics, err := newMetrics(config.TelemetryConfig.Meter())
	if err != nil {
		d("failed to initialize metrics: %v", err)
	}

	return &App[M]{
		config:  *config,
		metrics: metrics,
	}
}

// Mount registers an Endpoint on a Mux. The endpoint's Meta determines the
// HTTP method and path pattern used for registration.
//
// For each matching request the mounted handler parses the request body into
// the message type M, runs the configured Validator, executes the endpoint, and
// renders the returned Response. Errors are routed to the configured
// ErrorHandler; validation errors are stored in the session and the client is
// redirected back, unless a custom ErrorHandler is set.
//
// mountCfg configures this endpoint's validator and middleware. It can be omitted.
func (a *App[M]) Mount(mux Mux, endpoint Endpoint[M], mountCfg *MountConfig[M]) {
	if mountCfg == nil {
		//nolint:exhaustruct
		mountCfg = &MountConfig[M]{}
	}

	debug.Assert(mux != nil, "Mux must not be nil")
	debug.Assert(endpoint != nil, "Executor must not be nil")
	debug.Assert(a.config.ErrorHandler != nil, "Executor must specify the error handler")
	debug.Assert(a.config.FormDecoder != nil, "FormDecoder must be set")

	m := endpoint.Meta()

	debug.Assert(m.Method != "", "Executor must specify the HTTP method")
	debug.Assert(m.Path != "", "Executor must specify the HTTP path")

	pattern := fmt.Sprintf("%s %s", m.Method, m.Path)

	d("Mounting executor on pattern: %s", pattern)

	h := http.Handler(&handler[M]{
		endpoint:        endpoint,
		errorHandler:    a.config.ErrorHandler,
		validator:       mountCfg.Validator,
		hasPrecognition: m.Precognition,
		formDecoder:     a.config.FormDecoder,
		jsonOptions:     a.config.JSONOptions,
		telemetry:       a.config.TelemetryConfig,
		metrics:         a.metrics,
	})

	if len(mountCfg.Middleware) > 0 {
		h = httpmiddleware.NewChain(mountCfg.Middleware...).Middleware(h)
	}

	mux.Handle(pattern, h)
}
