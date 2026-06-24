// Package inertiaproto provides a high-level, message-oriented API for building
// Inertia.js applications whose request and response payloads are protobuf
// messages.
//
// Incoming JSON request bodies are unmarshalled with protojson, and outgoing
// responses are marshalled with protojson, so messages follow the canonical JSON
// mapping defined by the protobuf spec. Request bodies in
// application/x-www-form-urlencoded or multipart/form-data format are decoded
// with the standard form decoder.
//
// Every request is validated with protovalidate before the endpoint runs.
// Violations are reported back to the client as field-level validation errors
// following the Inertia.js validation flow: errors are stored in the flash
// session and the client is redirected back to the previous page, where they
// can be displayed. A non-validation error is handled by the configured error
// handler. Validation cannot be overridden or disabled per endpoint.
//
// A typical application creates one App via New and mounts multiple protobuf
// endpoints on it via Mount. Use NewMiddleware to wrap the HTTP mux in the
// Inertia.js protocol handler responsible for rendering and asset versioning.
package inertiaproto

import (
	"cmp"
	"errors"
	"fmt"

	"buf.build/go/protovalidate"
	"github.com/go-json-experiment/json"
	"github.com/go-playground/form/v4"
	"go.inout.gg/foundations/http/httphandler"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"go.segfaultmedaddy.com/inertia"
	"go.segfaultmedaddy.com/inertia/inertiaframe"
	"go.segfaultmedaddy.com/inertia/inertiaotel"
)

// DefaultErrorHandler is the error handler used when Config.ErrorHandler is nil.
//
// Validation errors are stored in the flash session and the client is
// redirected back to the previous page. Any other error is written as a generic
// HTTP error response.
var DefaultErrorHandler = inertiaframe.DefaultErrorHandler //nolint:gochecknoglobals

var jsonOptions = []json.Options{ //nolint:gochecknoglobals
	json.WithUnmarshalers(json.UnmarshalFunc(protojson.Unmarshal)),
	json.WithMarshalers(json.MarshalFunc(protojson.Marshal)),
}

// Re-exported types so endpoints can be built without importing inertiaframe
// directly.
type (
	// Meta describes an endpoint's routing configuration: HTTP method, URL
	// pattern, and whether the Precognition protocol is enabled.
	Meta = inertiaframe.Meta

	// Endpoint is a type-safe, message-oriented HTTP handler. M is the
	// protobuf request message type the endpoint accepts; its body is parsed
	// and validated before Execute is called.
	Endpoint[M proto.Message] = inertiaframe.Endpoint[M]
)

// Config holds application-wide configuration shared across all endpoints
// mounted on an App.
type Config struct {
	// ErrorHandler handles errors returned by mounted endpoints. If nil,
	// DefaultErrorHandler is used.
	ErrorHandler httphandler.ErrorHandler

	// FormDecoder decodes form and multipart request bodies into the message
	// type M. If nil, the default shared form decoder is used. Register custom
	// decoders here to support non-standard field types.
	FormDecoder *form.Decoder

	// TelemetryConfig configures OpenTelemetry tracing and metrics for mounted
	// endpoints. If nil, the default telemetry config is used.
	TelemetryConfig *inertiaotel.Config
}

func (c *Config) defaults() {
	c.ErrorHandler = cmp.Or(c.ErrorHandler, DefaultErrorHandler)
	c.FormDecoder = cmp.Or(c.FormDecoder, inertiaframe.DefaultFormDecoder)
	c.TelemetryConfig = cmp.Or(c.TelemetryConfig, inertiaotel.DefaultConfig)
}

// MountConfig holds per-endpoint configuration passed to Mount.
//
// The validator is always protovalidate and cannot be configured or disabled
// here; only middleware can be attached.
type MountConfig struct {
	// Middleware wraps the endpoint's HTTP handler in cross-cutting behavior
	// such as authentication, logging, or attaching shared props. Applied in
	// slice order; the first element is the outermost wrapper and runs first on
	// the request path.
	Middleware []inertiaframe.Middleware
}

// App is the entry point for mounting Inertia endpoints backed by protobuf
// messages.
//
// Create an App via New and reuse it to mount multiple endpoints. All mounted
// endpoints share the App's error handler, form decoder, telemetry, and
// protojson JSON options.
type App struct {
	frame *inertiaframe.App
}

// New creates a new App bound to the given Mux and Config.
//
// If config is nil, a zero-value Config is used. Sensible defaults are applied
// for any unset fields: DefaultErrorHandler, the default form decoder, and the
// default telemetry config. The JSON codec is always protojson and cannot be
// overridden.
func New(mux inertiaframe.Mux, config *Config) *App {
	if config == nil {
		//nolint:exhaustruct
		config = &Config{}
	}

	config.defaults()

	return &App{
		frame: inertiaframe.New(mux, &inertiaframe.Config{
			ErrorHandler:    config.ErrorHandler,
			FormDecoder:     config.FormDecoder,
			TelemetryConfig: config.TelemetryConfig,
			JSONOptions:     jsonOptions,
		}),
	}
}

// Mount registers an Endpoint on the App's Mux. The endpoint's Meta
// determines the HTTP method and path pattern used for registration.
//
// For each matching request the mounted handler parses the request body into
// the message type M (using protojson for JSON and the form decoder for forms),
// validates it with protovalidate, executes the endpoint, and renders the
// returned Response.
//
// Validation failures are reported to the client through the Inertia.js
// validation flow: the field-level errors are stored in the flash session and
// the client is redirected back, unless a custom ErrorHandler is set. Any other
// error is routed to the configured ErrorHandler.
//
// config configures this endpoint's middleware. It can be omitted; the
// validator is always protovalidate and cannot be overridden.
func Mount[M proto.Message](app *App, endpoint Endpoint[M], config *MountConfig) {
	if config == nil {
		//nolint:exhaustruct
		config = &MountConfig{}
	}

	inertiaframe.Mount(app.frame, endpoint, &inertiaframe.MountConfig[M]{
		Validator: inertiaframe.ValidatorFunc[M](func(data M) error {
			if err := protovalidate.Validate(data); err != nil {
				if verr, ok := errors.AsType[*protovalidate.ValidationError](err); ok {
					return convertValidationError(*verr)
				}

				return fmt.Errorf("failed request validation: %w", err)
			}

			return nil
		}),
		Middleware: config.Middleware,
	})
}

// convertValidationError converts a protobuf validation error into the
// field-level errors reported back to the client.
func convertValidationError(verr protovalidate.ValidationError) inertia.ValidationErrors {
	errs := make(inertia.ValidationErrors, 0, len(verr.Violations))
	for _, violation := range verr.Violations {
		errs = append(
			errs,
			inertia.NewValidationError(
				protovalidate.FieldPathString(violation.Proto.GetField()),
				violation.Proto.GetRuleId(),
			),
		)
	}

	return errs
}
