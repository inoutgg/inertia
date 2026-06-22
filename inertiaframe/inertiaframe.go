// Package inertiaframe provides a high-level, message-oriented API for building Inertia.js applications.
//
// It abstracts Inertia.js protocol details, providing automatic request parsing, validation,
// session management, and response handling. Build type-safe endpoints with minimal boilerplate.
package inertiaframe

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"slices"
	"time"

	"github.com/go-json-experiment/json"
	"github.com/go-playground/form/v4"
	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/http/httphandler"
	"go.inout.gg/foundations/http/httpmiddleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.segfaultmedaddy.com/inertia"
	"go.segfaultmedaddy.com/inertia/inertiaotel"
	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
	"go.segfaultmedaddy.com/inertia/internal/inertiahttp"
	"go.segfaultmedaddy.com/inertia/internal/inertiaredirect"
)

var d = debug.Debuglog("inertiaframe") //nolint:gochecknoglobals

// DefaultFormDecoder is the form decoder used by Mount when MountConfig.FormDecoder is nil.
// It is shared across every endpoint that does not supply its own decoder, so it can be
// configured once at startup (for example, to register custom type decoders).
var DefaultFormDecoder = form.NewDecoder() //nolint:gochecknoglobals

// ErrEmptyResponse is produced when an endpoint's Execute method returns a nil Response.
// It signals that the endpoint finished without producing any output for the client.
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

var kCtxKey = ctxKey{} //nolint:gochecknoglobals

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
//
// If the session cannot be loaded or saved, the underlying error is forwarded to
// httphandler.DefaultErrorHandler instead of panicking.
func DefaultValidationErrorHandler(w http.ResponseWriter, r *http.Request, errorer inertia.ValidationErrorer) error {
	sess, err := sessionFromRequest(r)
	if err != nil {
		d("failed to get session from request: %v", err)

		return err
	}

	sess.ErrorBag_ = inertia.ErrorBagFromRequest(r)
	sess.ValidationErrors_ = errorer.ValidationErrors()

	if err := sess.Save(w); err != nil {
		d("failed to save session: %v", err)

		return err
	}

	RedirectBack(w, r)

	return nil
}

// DefaultErrorHandler is the error handler used by Mount when MountConfig.ErrorHandler is nil.
// Validation errors are routed to DefaultValidationErrorHandler, which stores them in the
// session and redirects the client back; any other error is forwarded to the standard
// HTTP error handler.
//
//nolint:gochecknoglobals
var DefaultErrorHandler httphandler.ErrorHandler = httphandler.ErrorHandlerFunc(
	func(w http.ResponseWriter, r *http.Request, err error) {
		if verr, ok := errors.AsType[inertia.ValidationErrorer](err); ok {
			err = DefaultValidationErrorHandler(w, r, verr)
			if err == nil {
				// If no error is returned return normally, otherwise fall-through to the
				// default error handler.
				return
			}
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

// NewResponse creates a Response with the specified component and props.
// Optional ResponseOption functions can customize history behavior.
func NewResponse(component string, proper inertia.Proper, opts ...ResponseOption) Response {
	debug.Assert(component != "", "component must be non-empty")

	var options ResponseOptions

	if len(opts) > 0 {
		for _, opt := range opts {
			opt(&options)
		}
	}

	return &resp{proper, component, options}
}

// resp represents a response to an Inertia request.
//
// It is a helper that implements the Response interface and is used
// to create a response from a struct or an inertia.Proper.
type resp struct {
	proper    inertia.Proper
	component string
	opts      ResponseOptions
}

func (r *resp) Component() string        { return r.component }
func (r *resp) Proper() inertia.Proper   { return r.proper }
func (r *resp) Options() ResponseOptions { return r.opts }

type rawResp struct{ h Handler }

// NewRawResponse creates a Response that bypasses Inertia rendering.
// The provided handler has full control over the HTTP response.
// Useful for file downloads, API endpoints, or custom authentication flows.
func NewRawResponse(h Handler) Response {
	debug.Assert(h != nil, "Handler must not be nil")

	return &rawResp{h}
}

func (*rawResp) Component() string      { return "<raw>" }
func (*rawResp) Proper() inertia.Proper { return nil }

func (*rawResp) Options() ResponseOptions {
	//nolint:exhaustruct
	return ResponseOptions{}
}

func (rr *rawResp) Write(w http.ResponseWriter, r *http.Request) error {
	//nolint:wrapcheck
	return rr.h.ServeHTTP(w, r)
}

type externalRedirectMessage struct{ url string }

// NewExternalRedirectResponse creates a Response that redirects to an external URL
// (outside the Inertia app). Uses the Location protocol for proper client handling.
func NewExternalRedirectResponse(url string) Response {
	return &externalRedirectMessage{url: url}
}

func (m *externalRedirectMessage) Proper() inertia.Proper { return nil }
func (m *externalRedirectMessage) Component() string      { return "" }

func (m *externalRedirectMessage) Write(w http.ResponseWriter, r *http.Request) error {
	inertia.Location(w, r, m.url)
	return nil
}

type redirectBackMessage struct{}

// NewRedirectBackResponse creates a Response that redirects to the previous page.
// Uses the Referer header or session-stored path.
func NewRedirectBackResponse() Response {
	return &redirectBackMessage{}
}

func (m *redirectBackMessage) Proper() inertia.Proper { return nil }
func (m *redirectBackMessage) Component() string      { return "" }

func (m *redirectBackMessage) Write(w http.ResponseWriter, r *http.Request) error {
	RedirectBack(w, r)
	return nil
}

type redirectMessage struct{ url string }

// NewRedirectResponse creates a Response that redirects to the specified URL within the Inertia app.
func NewRedirectResponse(url string) Response {
	return &redirectMessage{url: url}
}

func (m *redirectMessage) Proper() inertia.Proper { return nil }
func (m *redirectMessage) Component() string      { return "" }

func (m *redirectMessage) Write(w http.ResponseWriter, r *http.Request) error {
	inertiaredirect.Redirect(w, r, m.url)
	return nil
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
// (history management). If implemented, Options() is called to configure the response.
type ResponseOptioner interface {
	Options() ResponseOptions
}

// Meta contains endpoint routing metadata used during mounting.
type Meta struct {
	// Method is the HTTP method (GET, POST, PUT, DELETE, etc.).
	Method string

	// Path is the URL pattern, following http.ServeMux syntax (e.g., "/users/{id}").
	Path string

	// Precognition enables Precognition validation requests for this route.
	Precognition bool
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

// MountConfig configures endpoint mounting behavior.
//
//nolint:govet
type MountConfig[M any] struct {
	// Validator validates requests before execution.
	//
	// If nil, no validation is performed.
	Validator Validator[M]

	// FormDecoder parses form-urlencoded and multipart requests.
	//
	// Defaults to DefaultFormDecoder if nil.
	FormDecoder *form.Decoder

	// ErrorHandler handles execution errors.
	//
	// Defaults to DefaultErrorHandler if nil.
	ErrorHandler httphandler.ErrorHandler

	// JSONOptions customizes JSON parsing (e.g., for protobuf).
	JSONOptions []json.Options

	// Middleware wraps the endpoint's HTTP handler in cross-cutting behavior (e.g., logging, auth).
	// Applied in slice order, with the first element being the outermost wrapper.
	Middleware []Middleware

	// TelemetryConfig configures OpenTelemetry tracing and metrics.
	//
	// If zero, telemetry is a no-op.
	TelemetryConfig *inertiaotel.Config
}

func (c *MountConfig[M]) defaults() {
	c.ErrorHandler = cmp.Or(c.ErrorHandler, DefaultErrorHandler)
	c.FormDecoder = cmp.Or(c.FormDecoder, DefaultFormDecoder)

	if c.TelemetryConfig == nil {
		c.TelemetryConfig = inertiaotel.DefaultConfig
	}
}

// Mount registers an Endpoint on a Mux, creating an HTTP handler that:
//   - Automatically parses JSON and form data into the message type M
//   - Validates requests using the configured Validator
//   - Executes the endpoint and renders the Response
//
// The endpoint's Meta() defines the HTTP method and path pattern.
func Mount[M any](mux Mux, endpoint Endpoint[M], opts *MountConfig[M]) {
	if opts == nil {
		//nolint:exhaustruct
		opts = &MountConfig[M]{}
	}

	opts.defaults()

	debug.Assert(mux != nil, "Mux must not be nil")
	debug.Assert(endpoint != nil, "Executor must not be nil")
	debug.Assert(opts.ErrorHandler != nil, "Executor must specify the error handler")
	debug.Assert(opts.FormDecoder != nil, "FormDecoder must be set")

	m := endpoint.Meta()

	debug.Assert(m.Method != "", "Executor must specify the HTTP method")
	debug.Assert(m.Path != "", "Executor must specify the HTTP path")

	pattern := fmt.Sprintf("%s %s", m.Method, m.Path)

	d("Mounting executor on pattern: %s", pattern)

	h := newHandler(
		endpoint,
		opts.ErrorHandler,
		opts.Validator,
		m.Precognition,
		opts.FormDecoder,
		opts.JSONOptions,
		opts.TelemetryConfig,
	)

	if len(opts.Middleware) > 0 {
		h = httpmiddleware.NewChain(opts.Middleware...).Middleware(h)
	}

	mux.Handle(pattern, h)
}

func handlePrecognition(
	w http.ResponseWriter,
	ineriaReq inertiahttp.Request,
	verr inertia.ValidationErrorer,
	jsonOptions []json.Options,
) error {
	h := w.Header()
	h.Set(inertiaheader.HeaderPrecognition, inertiaheader.HeaderValueTrue)
	h.Add(inertiaheader.HeaderVary, inertiaheader.HeaderPrecognition)

	if verr != nil {
		errorsMap := make(map[string][]string, verr.Len())

		errs := verr.ValidationErrors()
		for _, err := range errs {
			field := err.Field()

			// TODO: implement pattern matching for fields validation since laravel allows it.
			if (len(ineriaReq.PrecognitionProps) > 0 &&
				slices.Contains(ineriaReq.PrecognitionProps, field)) ||
				len(ineriaReq.PrecognitionProps) == 0 {
				errorsMap[field] = append(errorsMap[field], err.Error())
			}
		}

		// Unfortunately, we don't have an easy way to apply the filter in validation
		// function similar to Laravel, so we have to apply filter aftreward.
		if len(errorsMap) > 0 {
			w.Header().Set(inertiaheader.HeaderContentType, inertiaheader.ContentTypeJSON)
			w.WriteHeader(http.StatusUnprocessableEntity)

			// Use buf here to defer write to the response to prevent partial response write.
			var buf bytes.Buffer

			buf.WriteString(`{"errors":`)

			if err := json.MarshalWrite(&buf, errorsMap, jsonOptions...); err != nil {
				return fmt.Errorf("inertiaframe: failed to serialize errors: %w", err)
			}

			buf.WriteByte('}')

			if _, err := buf.WriteTo(w); err != nil {
				return fmt.Errorf("inertiaframe: failed to write response: %w", err)
			}

			return nil
		}
	}

	h.Set(inertiaheader.HeaderPrecognitionSuccess, inertiaheader.HeaderValueTrue)
	w.WriteHeader(http.StatusNoContent)

	return nil
}

func decodeBody[M any](
	r *http.Request,
	formDecoder *form.Decoder,
	jsonOptions []json.Options,
) (M, error) {
	var msg M

	mediaType, _, err := mime.ParseMediaType(r.Header.Get(inertiaheader.HeaderContentType))
	if err != nil {
		return msg, fmt.Errorf("inertiaframe: failed to parse Content-Type header: %w", err)
	}
	defer r.Body.Close()

	span := trace.SpanFromContext(r.Context())
	span.SetAttributes(attribute.String("inertiaframe.request.mime", mediaType))

	// Inertia accepts only JSON or multipart/form-data.
	switch mediaType {
	case mediaTypeJSON:
		{
			d("received JSON request")

			if err := json.UnmarshalRead(r.Body, &msg, jsonOptions...); err != nil {
				return msg, fmt.Errorf("inertiaframe: failed to decode request: %w", err)
			}
		}
	case mediaTypeForm, mediaTypeMultipart:
		{
			d("received form request")

			if err := r.ParseForm(); err != nil {
				return msg, fmt.Errorf("inertiaframe: failed to parse form data: %w", err)
			}

			if err := formDecoder.Decode(&msg, r.Form); err != nil {
				return msg, fmt.Errorf("inertiaframe: failed to decode form data: %w", err)
			}
		}
	default:
		d("unknown Content-Type %q, leaving message empty", mediaType)
	}

	return msg, nil
}

// withPrecognition intercepts precognition requests, rendering a 204 (success)
// or 422 (validation errors) response instead of executing the endpoint.
//
// Non-precognition requests, non-validation errors, and handlePrecognition
// failures are returned unchanged for the caller to dispatch.
func withPrecognition(inner httphandler.HandlerFunc, jsonOptions []json.Options) httphandler.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		err := inner(w, r)

		req := inertiahttp.RenderScopeFromRequest(r).Request()
		if !req.Precognition {
			return err
		}

		// precognition requests are a special type of requests: the request
		// goes through the regular flow (middleware, validation, etc.), but
		// the controller is never executed.
		//
		// Precognition should handle the response only when it is a validation
		// error; otherwise the error is returned for the error dispatcher to
		// handle.
		if err != nil {
			if verr, ok := errors.AsType[inertia.ValidationErrorer](err); ok {
				return handlePrecognition(w, req, verr, jsonOptions)
			}

			return err
		}

		return handlePrecognition(w, req, nil, jsonOptions)
	}
}

// extractRequest parses the request body into the message type M.
//
// If M implements RawRequestExtractor, its Extract method is used instead of the
// default JSON/form decoder. GET requests skip body decoding entirely.
func extractRequest[M any](
	r *http.Request,
	formDecoder *form.Decoder,
	jsonOptions []json.Options,
) (M, error) {
	var msg M

	if extract, ok := any(msg).(RawRequestExtractor); ok {
		if err := extract.Extract(r); err != nil {
			return msg, fmt.Errorf("inertiaframe: failed to extract request data: %w", err)
		}

		return msg, nil
	}

	if r.Method == http.MethodGet {
		d("GET %s, skipping body decode", r.URL.Path)
		return msg, nil
	}

	return decodeBody[M](r, formDecoder, jsonOptions)
}

// render writes resp to w via the Inertia renderer.
//
// RawResponseWriter responses bypass Inertia rendering and write directly to w.
// Otherwise the response's options, props (response-provided and context-shared),
// and session-stored validation errors are merged into renderCtx before rendering.
func render(
	w http.ResponseWriter,
	r *http.Request,
	executionResponse Response,
	metrics *metrics,
) error {
	span := trace.SpanFromContext(r.Context())

	var renderCtx inertia.RenderContext

	if writer, ok := executionResponse.(RawResponseWriter); ok {
		d("writing raw response for %s %s", r.Method, r.URL.Path)

		if err := writer.Write(w, r); err != nil {
			return fmt.Errorf("inertiaframe: failed to write response: %w", err)
		}

		return nil
	}

	if optioner, ok := executionResponse.(ResponseOptioner); ok {
		opts := optioner.Options()

		renderCtx.ClearHistory = opts.ClearHistory
		renderCtx.EncryptHistory = opts.EncryptHistory

		span.SetAttributes(
			attribute.Bool("inertiaframe.response.clear_history", opts.ClearHistory),
			attribute.Bool("inertiaframe.response.encrypt_history", opts.EncryptHistory),
		)
	}

	if proper, ok := r.Context().Value(kCtxKey).(inertia.Proper); ok {
		renderCtx.SharedProps = proper.Props()
	}

	if proper := executionResponse.Proper(); proper != nil && proper.Len() > 0 {
		renderCtx.Props = proper.Props()
	}

	sess, err := sessionFromRequest(r)
	if err != nil {
		return fmt.Errorf("inertiaframe: failed to get session: %w", err)
	}

	if errs := sess.ValidationErrors(); errs != nil {
		renderCtx.ErrorBag = sess.ErrorBag()
		renderCtx.AddValidationErrorer(inertia.ValidationErrors(errs))

		metrics.validationErrorsCounter.Add(
			r.Context(),
			int64(len(errs)),
			metric.WithAttributes(
				attribute.String("inertiaframe.error_bag", renderCtx.ErrorBag),
			),
		)
	}

	componentName := executionResponse.Component()
	debug.Assert(componentName != "", "component name must not be empty, when using non RawResponseWriter")

	span.SetAttributes(
		attribute.String("inertiaframe.response.component", componentName),
	)

	d(
		"rendering %q with %d props and %d shared props",
		componentName,
		len(renderCtx.Props),
		len(renderCtx.SharedProps),
	)

	if err := inertia.Render(w, r, componentName, renderCtx); err != nil {
		return fmt.Errorf("inertiaframe: failed to render: %w", err)
	}

	return nil
}

// newHandler creates a new http.Handler for the given endpoint.
func newHandler[M any](
	endpoint Endpoint[M],
	errorHandler httphandler.ErrorHandler,
	validator Validator[M],
	hasPrecognition bool,
	formDecoder *form.Decoder,
	jsonOptions []json.Options,
	telemetry *inertiaotel.Config,
) http.Handler {
	debug.Assert(endpoint != nil, "Endpoint must not be nil")
	debug.Assert(errorHandler != nil, "ErrorHandler must be set")
	debug.Assert(formDecoder != nil, "FormDecoder must be set")
	debug.Assert(telemetry != nil, "Telemetry must be set")

	metrics, err := newMetrics(telemetry.Meter())
	if err != nil {
		d("failed to initialize metrics: %v", err)
	}

	inner := httphandler.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		ctx, span := telemetry.Tracer().Start(
			r.Context(),
			"inertiaframe.endpoint",
			trace.WithSpanKind(trace.SpanKindInternal),
			trace.WithAttributes(
				attribute.String("inertiaframe.endpoint.method", endpoint.Meta().Method),
				attribute.String("inertiaframe.endpoint.path", endpoint.Meta().Path),
			),
		)
		defer span.End()

		r = r.WithContext(ctx)

		msg, err := extractRequest[M](r, formDecoder, jsonOptions)
		if err != nil {
			return err
		}

		if validator != nil {
			if err := validator.Validate(msg); err != nil {
				d("failed to validate request")

				metrics.validationErrorsCounter.Add(
					ctx,
					1,
					metric.WithAttributes(
						attribute.String(
							"inertiaframe.error_bag",
							inertia.ErrorBagFromRequest(r),
						),
					),
				)

				return fmt.Errorf("inertiaframe: validation failed for request: %w", err)
			}
		}

		inertiaReq := inertiahttp.RenderScopeFromRequest(r).Request()

		// Precognition requests must not execute the main endpoint logic, but instead
		// validate everything up to the logic.
		//
		// If validation fails for a precognition request the wrapping handler will
		// take care of it, otherwise it will just pass it with 204.
		if hasPrecognition && inertiaReq.Precognition {
			return nil
		}

		execStart := time.Now()
		result, err := endpoint.Execute(ctx, newRequest(msg))
		metrics.endpointDuration.Record(ctx, float64(time.Since(execStart).Milliseconds()))

		if err != nil {
			return fmt.Errorf("inertiaframe: failed to execute: %w", err)
		}

		if result == nil {
			d("received empty response")

			metrics.emptyResponsesCounter.Add(ctx, 1)

			return ErrEmptyResponse
		}

		span.SetAttributes(
			attribute.String("inertiaframe.response.type", responseType(result)),
		)

		return render(w, r, result, metrics)
	})

	if hasPrecognition {
		inner = withPrecognition(inner, jsonOptions)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := inner(w, r); err != nil {
			metrics.errorResponsesCounter.Add(r.Context(), 1)
			errorHandler.ServeHTTP(w, r, err)
		}
	})
}

func responseType(resp Response) string {
	switch resp.(type) {
	case *rawResp:
		return "raw"
	case *redirectMessage:
		return "redirect"
	case *redirectBackMessage:
		return "redirect_back"
	case *externalRedirectMessage:
		return "external_redirect"
	default:
		return "inertia"
	}
}
