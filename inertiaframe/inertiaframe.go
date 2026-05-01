// Package inertiaframe provides a high-level, message-oriented API for building Inertia.js applications.
//
// It abstracts Inertia.js protocol details, providing automatic request parsing, validation,
// session management, and response handling. Build type-safe endpoints with minimal boilerplate.
package inertiaframe

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"

	"github.com/go-json-experiment/json"
	"github.com/go-playground/form/v4"
	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/http/httphandler"
	"go.inout.gg/foundations/http/httpmiddleware"
	"go.inout.gg/foundations/must"

	"go.segfaultmedaddy.com/inertia"
	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
	"go.segfaultmedaddy.com/inertia/internal/inertiaredirect"
)

var d = debug.Debuglog("inertiaframe") //nolint:gochecknoglobals

var DefaultFormDecoder = form.NewDecoder() //nolint:gochecknoglobals

var ErrEmptyResponse = errors.New("inertiaframe: empty response")

type (
	Middleware     = httpmiddleware.Middleware
	MiddlewareFunc = httpmiddleware.MiddlewareFunc
	Handler        = httphandler.Handler
	HandlerFunc    = httphandler.HandlerFunc
)

var (
	_ RawResponseWriter = (*redirectMessage)(nil)
	_ RawResponseWriter = (*redirectBackMessage)(nil)
	_ RawResponseWriter = (*externalRedirectMessage)(nil)
	_ Response          = (*resp)(nil)
)

type ctxKey struct{}

var kCtxKey = &ctxKey{} //nolint:gochecknoglobals

// WithProps attaches shared props to the request context for later merging with response props.
// Useful in middleware to provide global data (e.g., auth user, flash messages) to all pages.
//
// Response props take precedence over shared props when keys overlap.
// Prefer setting props directly in responses when possible; use this for cross-cutting concerns.
func WithProps(r *http.Request, props inertia.Proper) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), kCtxKey, props))
}

// RedirectBack redirects to the previous page using the Referer header.
// Falls back to the session-stored path if the header is missing, or "/" if no session.
func RedirectBack(w http.ResponseWriter, r *http.Request) {
	referer := r.Header.Get(inertiaheader.HeaderReferer)
	if referer == "" {
		sess, err := sessionFromRequest(r)
		if err != nil {
			d("failed to get session from request, using default '/'")

			referer = "/"
		} else {
			referer = sess.Referer()
		}
	}

	d("redirecting back to %s", referer)

	inertiaredirect.Redirect(w, r, referer)
}

// DefaultValidationErrorHandler handles validation errors by storing them in the session
// and redirecting back to the previous page where they can be displayed.
func DefaultValidationErrorHandler(w http.ResponseWriter, r *http.Request, errorer inertia.ValidationErrorer) {
	errorBag := inertia.ErrorBagFromRequest(r)
	sess := must.Must(sessionFromRequest(r))

	sess.ErrorBag_ = errorBag
	sess.ValidationErrors_ = errorer.ValidationErrors()

	must.Must1(sess.Save(w))

	RedirectBack(w, r)
}

//nolint:gochecknoglobals
var DefaultErrorHandler httphandler.ErrorHandler = httphandler.ErrorHandlerFunc(
	func(w http.ResponseWriter, r *http.Request, err error) {
		var errorer inertia.ValidationErrorer
		if errors.As(err, &errorer) {
			DefaultValidationErrorHandler(w, r, errorer)
			return
		}

		httphandler.DefaultErrorHandler(w, r, err)
	},
)

const (
	mediaTypeJSON      = "application/json"
	mediaTypeForm      = "application/x-www-form-urlencoded"
	mediaTypeMultipart = "multipart/form-data"
)

// Request represents a parsed and validated client request.
type Request[M any] struct {
	// Message is the decoded request payload (from JSON or form data).
	// If M implements RawRequestExtractor, custom extraction logic is used.
	Message M
}

// newRequest creates a new request.
func newRequest[M any](m M) *Request[M] {
	return &Request[M]{Message: m}
}

// ResponseOptions configures Inertia response behavior for a specific page.
type ResponseOptions struct {
	// ClearHistory instructs the client to clear its history stack.
	ClearHistory bool

	// EncryptHistory instructs the client to encrypt the history state.
	EncryptHistory bool

	// Concurrency sets the maximum concurrent lazy prop resolutions for this response.
	Concurrency int
}

func (opt *ResponseOptions) defaults() {
	opt.Concurrency = cmp.Or(opt.Concurrency, inertia.DefaultConcurrency)
}

// ResponseOption is used to configure inertia response.
type ResponseOption func(*ResponseOptions)

// Response represents an endpoint's response, instructing the client to render a component or redirect.
//
// If a Response implements RawResponseWriter, it bypasses normal Inertia rendering
// and writes directly to http.ResponseWriter (useful for downloads, APIs, etc.).
type Response interface {
	// Component returns the frontend component name to render.
	// Must be non-empty unless the response implements RawResponseWriter.
	Component() string

	// Proper returns the props to send to the component.
	// Can be nil for redirect responses or when no props are needed.
	Proper() inertia.Proper
}

// RawRequestExtractor allows custom request parsing logic.
// When a request message implements this interface, it bypasses the default
// JSON/form decoder and calls Extract instead.
type RawRequestExtractor interface {
	// Extract parses and populates fields from the raw HTTP request.
	Extract(*http.Request) error
}

// RawResponseWriter allows custom response writing logic.
// When a Response implements this interface, it bypasses normal Inertia rendering
// and calls Write directly. Useful for non-Inertia responses (downloads, APIs, etc.).
type RawResponseWriter interface {
	Write(http.ResponseWriter, *http.Request) error
}

// ResponseOptioner is an optional interface for Responses that need custom options
// (history management, concurrency). If implemented, Options() is called to configure the response.
type ResponseOptioner interface {
	Options() ResponseOptions
}

// Meta contains endpoint routing metadata used during mounting.
type Meta struct {
	// Pattern defines the HTTP method and url. It follows the same format
	// as the std Go HTTP muxer, i.e.: "<Method> <Path>".
	Pattern string
}

// Validator validates parsed request messages before execution.
type Validator[M any] interface {
	// Validate checks the request message for errors.
	// If validation fails, return a ValidationErrorer to send errors to the client,
	// or any other error to trigger error handling.
	Validate(M) error
}

// ValidatorFunc is a function that implements the Validator interface.
type ValidatorFunc[M any] func(M) error

func (f ValidatorFunc[M]) Validate(v M) error { return f(v) }

// Endpoint represents a type-safe, message-oriented HTTP handler.
type Endpoint[M any] interface {
	// Execute processes the validated request and returns a Response.
	// Errors are automatically handled based on type (e.g., ValidationErrorer).
	Execute(context.Context, *Request[M]) (Response, error)

	// Meta returns routing metadata (HTTP method and path pattern).
	Meta() Meta
}

// Mux represents an HTTP router compatible with http.ServeMux.
type Mux interface {
	// Handle registers a handler for a pattern (e.g., "POST /users/{id}").
	Handle(pattern string, h http.Handler)
}

// MountOpts configures endpoint mounting behavior.
type MountOpts[M any] struct {
	// Validator validates requests before execution. If nil, no validation is performed.
	Validator Validator[M]

	// FormDecoder parses form-urlencoded and multipart requests.
	// Defaults to DefaultFormDecoder if nil.
	FormDecoder *form.Decoder

	// ErrorHandler handles execution errors. Defaults to DefaultErrorHandler if nil.
	ErrorHandler httphandler.ErrorHandler

	// JSONUnmarshalOptions customizes JSON parsing (e.g., for protobuf).
	JSONUnmarshalOptions []json.Options
}

// Mount registers an Endpoint on a Mux, creating an HTTP handler that:
//   - Automatically parses JSON and form data into the message type M
//   - Validates requests using the configured Validator
//   - Executes the endpoint and renders the Response
//
// The endpoint's Meta() defines the HTTP method and path pattern.
func Mount[M any](mux Mux, endpoint Endpoint[M], opts *MountOpts[M]) {
	if opts == nil {
		//nolint:exhaustruct
		opts = &MountOpts[M]{}
	}

	opts.ErrorHandler = cmp.Or(opts.ErrorHandler, DefaultErrorHandler)
	opts.FormDecoder = cmp.Or(opts.FormDecoder, DefaultFormDecoder)

	debug.Assert(endpoint != nil, "Executor must not be nil")
	debug.Assert(opts.ErrorHandler != nil, "Executor must specify the error handler")

	meta := endpoint.Meta()

	mux.Handle(
		meta.Pattern,
		newHandler(
			endpoint,
			opts.ErrorHandler,
			opts.Validator,
			opts.FormDecoder,
			opts.JSONUnmarshalOptions,
		),
	)
}

// newHandler creates a new http.Handler for the given endpoint.
func newHandler[M any](
	endpoint Endpoint[M],
	errorHandler httphandler.ErrorHandler,
	validator Validator[M],
	formDecoder *form.Decoder,
	jsonUnmarshalOptions []json.Options,
) http.Handler {
	handleError := httphandler.WithErrorHandler(errorHandler)

	return handleError(httphandler.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		var (
			msg       M
			renderCtx inertia.RenderContext
		)

		ctx := r.Context()

		if extract, ok := any(msg).(RawRequestExtractor); ok {
			if err := extract.Extract(r); err != nil {
				return fmt.Errorf("inertiaframe: failed to extract request data: %w", err)
			}
		} else if r.Method != http.MethodGet {
			mediaType, _, err := mime.ParseMediaType(
				r.Header.Get(inertiaheader.HeaderContentType))
			if err != nil {
				return fmt.Errorf("inertiaframe: failed to parse Content-Type header: %w", err)
			}

			// Inertia accepts only JSON or multipart/form-data.
			switch mediaType {
			case mediaTypeJSON:
				{
					d("received JSON request")

					if err := json.UnmarshalRead(r.Body, &msg, jsonUnmarshalOptions...); err != nil {
						return fmt.Errorf("inertiaframe: failed to decode request: %w", err)
					}
				}
			case mediaTypeForm, mediaTypeMultipart:
				{
					d("received form request")

					if err := r.ParseForm(); err != nil {
						return fmt.Errorf("inertiaframe: failed to parse form data: %w", err)
					}

					if err := formDecoder.Decode(&msg, r.Form); err != nil {
						return fmt.Errorf("inertiaframe: failed to decode form data: %w", err)
					}
				}
			}
		}

		if validator != nil {
			if err := validator.Validate(msg); err != nil {
				d("failed to validate request")

				return fmt.Errorf("inertiaframe: failed to validate request: %w", err)
			}
		}

		resp, err := endpoint.Execute(ctx, newRequest(msg))
		if err != nil {
			return fmt.Errorf("inertiaframe: failed to execute: %w", err)
		}

		if resp == nil {
			d("received empty response")

			return ErrEmptyResponse
		}

		if writer, ok := resp.(RawResponseWriter); ok {
			if err := writer.Write(w, r); err != nil {
				return fmt.Errorf("inertiaframe: failed to write response: %w", err)
			}

			return nil
		}

		if optioner, ok := resp.(ResponseOptioner); ok {
			opts := optioner.Options()

			renderCtx.ClearHistory = opts.ClearHistory
			renderCtx.EncryptHistory = opts.EncryptHistory
			renderCtx.Concurrency = opts.Concurrency
		}

		var props []inertia.Prop

		if p, ok := r.Context().Value(kCtxKey).(inertia.Proper); ok {
			d("has shared props")

			props = p.Props()
		}

		if p := resp.Proper(); p.Len() > 0 {
			d("response has props")

			props = append(props, p.Props()...)
		}

		renderCtx.Props = props

		sess, err := sessionFromRequest(r)
		if err != nil {
			return fmt.Errorf("inertiaframe: failed to get session: %w", err)
		}

		errors := sess.ValidationErrors()

		if errors != nil {
			renderCtx.ErrorBag = sess.ErrorBag()
			renderCtx.AddValidationErrorer(inertia.ValidationErrors(errors))
		}

		component := resp.Component()
		debug.Assert(component != "", "component must not be empty, when using non RawResponseWriter")

		if err := inertia.Render(w, r, component, renderCtx); err != nil {
			return fmt.Errorf("inertiaframe: failed to render: %w", err)
		}

		return nil
	}))
}
