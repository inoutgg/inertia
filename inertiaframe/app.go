// Package inertiaframe provides a high-level, message-oriented API for building
// Inertia.js applications.
//
// Incoming JSON request bodies are unmarshalled into the endpoint's message
// type M, and form/multipart bodies are decoded with the standard form decoder.
// A message may implement RawRequestExtractor to take over body parsing
// entirely. GET requests skip body decoding.
//
// A Validator runs before the endpoint executes; validation errors are
// reported back to the client through the Inertia.js validation flow: they are
// stored in the flash session and the client is redirected back to the previous
// page, where they can be displayed. Any other error is handled by the
// configured ErrorHandler.
//
// Endpoints return a Response, which is rendered as an Inertia page (a
// frontend component with props) or, for responses implementing
// RawResponseWriter, written directly to the client. Redirect and
// external-redirect responses are provided as built-in Response constructors.
//
// A typical application creates one App via New (bound to a Mux) and mounts
// multiple endpoints on it via Mount. Wrap the Mux with the Inertia.js
// protocol middleware from the root inertia package to handle rendering and
// asset versioning.
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
	// type M. If nil, DefaultFormDecoder is used. Register custom decoders on
	// it to support non-standard field types.
	FormDecoder *form.Decoder

	// TelemetryConfig configures OpenTelemetry tracing and metrics for mounted
	// endpoints. If nil, the default telemetry config is used.
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

// MountConfig holds per-endpoint configuration passed to Mount.
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

// App is the entry point for mounting Inertia endpoints. It binds a Mux to a
// shared Config, so every endpoint mounted via Mount reuses the same error
// handler, form decoder, telemetry, and JSON options.
//
// Create an App via New and reuse it to mount multiple endpoints. Endpoints of
// different message types can be mounted on the same App: Mount is generic in M,
// so each call picks its own message type while sharing the App's configuration.
type App struct {
	mux     Mux
	metrics *metrics
	config  *Config
}

// New creates a new App bound to the given Mux and Config.
//
// If config is nil, a zero-value Config is used. Sensible defaults are applied
// for any unset fields: DefaultErrorHandler, DefaultFormDecoder, and
// inertiaotel.DefaultConfig for telemetry.
func New(mux Mux, config *Config) *App {
	debug.Assert(mux != nil, "Mux must not be nil")

	if config == nil {
		//nolint:exhaustruct
		config = &Config{}
	}

	config.defaults()

	metrics, err := newMetrics(config.TelemetryConfig.Meter())
	if err != nil {
		d("failed to initialize metrics: %v", err)
	}

	return &App{
		mux:     mux,
		config:  config,
		metrics: metrics,
	}
}

// Mount registers an Endpoint on the App's Mux. The endpoint's Meta
// determines the HTTP method and path pattern used for registration.
//
// For each matching request the mounted handler parses the request body into
// the message type M, runs the configured Validator, executes the endpoint, and
// renders the returned Response. Errors are routed to the configured
// ErrorHandler; validation errors are stored in the session and the client is
// redirected back, unless a custom ErrorHandler is set.
//
// config configures this endpoint's validator and middleware. It can be omitted,
// in which case no validator runs and no middleware is applied.
func Mount[M any](app *App, endpoint Endpoint[M], config *MountConfig[M]) {
	if config == nil {
		//nolint:exhaustruct
		config = &MountConfig[M]{}
	}

	debug.Assert(endpoint != nil, "Executor must not be nil")
	debug.Assert(app.config.ErrorHandler != nil, "Executor must specify the error handler")
	debug.Assert(app.config.FormDecoder != nil, "FormDecoder must be set")

	m := endpoint.Meta()

	debug.Assert(m.Method != "", "Executor must specify the HTTP method")
	debug.Assert(m.Path != "", "Executor must specify the HTTP path")

	pattern := fmt.Sprintf("%s %s", m.Method, m.Path)

	d("Mounting executor on pattern: %s", pattern)

	h := http.Handler(&handler[M]{
		endpoint:        endpoint,
		errorHandler:    app.config.ErrorHandler,
		validator:       config.Validator,
		hasPrecognition: m.Precognition,
		formDecoder:     app.config.FormDecoder,
		jsonOptions:     app.config.JSONOptions,
		telemetry:       app.config.TelemetryConfig,
		metrics:         app.metrics,
	})

	if len(config.Middleware) > 0 {
		h = httpmiddleware.NewChain(config.Middleware...).Middleware(h)
	}

	app.mux.Handle(pattern, h)
}
