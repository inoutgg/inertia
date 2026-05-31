package inertiaproto

import (
	"errors"
	"fmt"

	"buf.build/go/protovalidate"
	"github.com/go-json-experiment/json"
	"go.inout.gg/foundations/http/httphandler"
	"go.inout.gg/foundations/http/httpmiddleware"
	"go.inout.gg/foundations/must"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"go.segfaultmedaddy.com/inertia"
	"go.segfaultmedaddy.com/inertia/contrib/vite"
	"go.segfaultmedaddy.com/inertia/inertiaframe"
)

var DefaultErrorHandler = inertiaframe.DefaultErrorHandler //nolint:gochecknoglobals

// Endpoint is a inertiaframe.Endpoint that returns a protobuf message as a response.
type Endpoint[M proto.Message] = inertiaframe.Endpoint[M]

type Config struct {
	SSRClient     inertia.SSRClient
	ErrorHandler  httphandler.ErrorHandler
	ViteConfig    *vite.Config
	BundleVersion string
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
			RootViewID: inertia.DefaultRootViewID,

			Concurrency: inertia.DefaultConcurrency,
			Version:     config.BundleVersion,
			SSRClient:   config.SSRClient,
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
	inertiaframe.Mount(mux, endpoint, &inertiaframe.MountConfig[M]{ //nolint:exhaustruct
		Validator: inertiaframe.ValidatorFunc[M](func(data M) error {
			if err := protovalidate.Validate(data); err != nil {
				if err, ok := errors.AsType[*protovalidate.ValidationError](err); ok {
					return convertValidationError(*err)
				}

				return fmt.Errorf("failed request validation: %w", err)
			}

			return nil
		}),
		FormDecoder:  inertiaframe.DefaultFormDecoder,
		ErrorHandler: DefaultErrorHandler,
		JSONUnmarshalOptions: []json.Options{
			json.WithUnmarshalers(json.UnmarshalFunc(protojson.Unmarshal)),
		},
	})
}

// convertValidationError converts a protobuf validation error to an inertia validation error.
func convertValidationError(verr protovalidate.ValidationError) inertia.ValidationErrors {
	var errs inertia.ValidationErrors
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
