package inertiaproto

import (
	"errors"
	"fmt"

	"buf.build/go/protovalidate"
	"github.com/go-json-experiment/json"
	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/http/httphandler"
	"go.inout.gg/foundations/http/httpmiddleware"
	"go.inout.gg/foundations/must"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"go.segfaultmedaddy.com/inertia"
	"go.segfaultmedaddy.com/inertia/contrib/vite"
	"go.segfaultmedaddy.com/inertia/inertiaframe"
)

// DefaultErrorHandler is the error handler used by Mount; it routes validation
// errors back to the previous page and forwards any other error to the standard
// HTTP error handler.
var DefaultErrorHandler = inertiaframe.DefaultErrorHandler //nolint:gochecknoglobals

// Endpoint is a inertiaframe.Endpoint that returns a protobuf message as a response.
type Endpoint[M proto.Message] = inertiaframe.Endpoint[M]

// Config configures the inertiaproto middleware: Vite integration, SSR
// client, error handler, and the client-visible bundle version.
type Config struct {
	// SSRClient enables server-side rendering of Inertia pages; if nil,
	// pages are rendered client-side only.
	SSRClient inertia.SSRClient

	// ErrorHandler handles execution errors. Defaults to DefaultErrorHandler.
	ErrorHandler httphandler.ErrorHandler

	// ViteConfig configures the html/template used for page rendering.
	ViteConfig *vite.Config

	// BundleVersion is the asset version reported to the client via
	// X-Inertia-Version for stale-asset detection.
	BundleVersion string

	// Concurrency sets the maximum number of props that can be resolved concurrently
	// by the renderer's internal ResultPool. It only affects props marked as concurrent.
	//
	// Defaults to runtime.GOMAXPROCS(0). A value of 0 means no limit.
	Concurrency int
}

// Option configures the middleware.
type Option func(*Config)

// WithViteConfig configures the middleware with a vite configuration.
func WithViteConfig(cfg *vite.Config) Option {
	return func(c *Config) { c.ViteConfig = cfg }
}

// WithSSRClient configures the middleware with an SSR client.
func WithSSRClient(client inertia.SSRClient) Option {
	return func(c *Config) { c.SSRClient = client }
}

// WithBundleVersion configures the inertia bundle version.
func WithBundleVersion(v string) Option {
	return func(c *Config) { c.BundleVersion = v }
}

// WithConcurrency sets the maximum number of props that can be resolved concurrently
// for this middleware. This only affects props marked as concurrent.
//
// A value of 0 uses the default (runtime.GOMAXPROCS(0)). A value of 0 also means
// no limit at the pool level.
func WithConcurrency(concurrency int) Option {
	return func(c *Config) { c.Concurrency = concurrency }
}

// NewMiddleware creates a new middleware that handles inertia requests
// using protobuf messages.
func NewMiddleware(template string, opts ...Option) httpmiddleware.MiddlewareFunc {
	var config Config
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
			JSONMarshalOptions: []json.Options{
				json.WithMarshalers(json.MarshalFunc(protojson.Marshal)),
			},
		},
	))
}

// Mount mounts a new inertia endpoint that uses protobuf messages for
// communication with a client.
//
// Incoming requests are validated with protovalidate and if the validation fails,
// the error is returned to the client as inertia validation error.
//
// The incoming and outgoing messages are automatically marshaled and unmarshaled
// from/to JSON using protojson.
func Mount[M proto.Message](mux inertiaframe.Mux, endpoint Endpoint[M]) {
	debug.Assert(mux != nil, "Mux must not be nil")
	debug.Assert(endpoint != nil, "Endpoint must not be nil")

	inertiaframe.New[M](&inertiaframe.Config{ //nolint:exhaustruct
		FormDecoder:  inertiaframe.DefaultFormDecoder,
		ErrorHandler: DefaultErrorHandler,
		JSONOptions: []json.Options{
			json.WithUnmarshalers(json.UnmarshalFunc(protojson.Unmarshal)),
		},
	}).Mount(mux, endpoint, &inertiaframe.MountConfig[M]{ //nolint:exhaustruct
		Validator: inertiaframe.ValidatorFunc[M](func(data M) error {
			if err := protovalidate.Validate(data); err != nil {
				if err, ok := errors.AsType[*protovalidate.ValidationError](err); ok {
					return convertValidationError(*err)
				}

				return fmt.Errorf("failed request validation: %w", err)
			}

			return nil
		}),
	})
}

// convertValidationError converts a protobuf validation error to an inertia validation error.
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
