package inertiaframe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

// ErrEmptyResponse is returned when an endpoint's Execute returns a nil
// Response, indicating the endpoint produced no output for the client.
var ErrEmptyResponse = errors.New("inertiaframe: empty response")

// Re-exported handler and middleware types from the foundations http packages
// so endpoints can be built without importing them directly.
type (
	Middleware     = httpmiddleware.Middleware
	MiddlewareFunc = httpmiddleware.MiddlewareFunc
	Handler        = httphandler.Handler
	HandlerFunc    = httphandler.HandlerFunc
)

type ctxKey struct{}

var kCtxKey = ctxKey{} //nolint:gochecknoglobals

// WithProps attaches shared props to the request context so they are merged
// into every Inertia response rendered during this request.
//
// Useful in middleware to provide global data (e.g., the authenticated user,
// flash messages) to all pages without each endpoint setting it explicitly.
// When a response provides props with overlapping keys, the response's props
// take precedence over the shared props.
func WithProps(r *http.Request, props inertia.Proper) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), kCtxKey, props))
}

// RedirectBack redirects the client to the previous page.
//
// The destination is resolved in order: the Referer header on the request, the
// path stored in the session, or "/" as a final fallback.
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

// DefaultValidationErrorHandler handles a validation error by storing the
// validation errors in the session and redirecting the client back to the
// previous page, where the errors can be displayed to the user.
//
// If the session cannot be loaded or saved, the underlying error is returned
// so the caller can fall back to the default error handling strategy.
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

// DefaultErrorHandler is the error handler used by Mount when
// Config.ErrorHandler is nil.
//
// Validation errors are routed to DefaultValidationErrorHandler, which stores
// them in the session and redirects the client back. Any other error is written
// as a generic HTTP error response.
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

// Meta describes an endpoint's routing configuration.
type Meta struct {
	// Method is the HTTP method the endpoint responds to (e.g. "GET", "POST").
	Method string

	// Path is the URL pattern, following http.ServeMux syntax (e.g. "/users/{id}").
	Path string

	// Precognition enables Inertia's Precognition protocol for this endpoint.
	// When enabled, clients can send validation-only requests that skip
	// endpoint execution and respond with 204 No Content when valid, or 422
	// Unprocessable Entity with the field errors when invalid.
	Precognition bool
}

// Validator checks a parsed request message for errors before the endpoint runs.
type Validator[M any] interface {
	// Validate inspects the message and returns an error if it is invalid.
	//
	// Return a value implementing inertia.ValidationErrorer to send field-level
	// errors back to the client as validation errors. Return any other error to
	// trigger generic error handling.
	Validate(M) error
}

// ValidatorFunc is a function adapter for the Validator interface.
type ValidatorFunc[M any] func(M) error

func (f ValidatorFunc[M]) Validate(v M) error { return f(v) }

// Endpoint is a type-safe, message-oriented HTTP handler.
//
// M is the request message type the endpoint accepts. Incoming request bodies
// are parsed into M before Execute is called.
type Endpoint[M any] interface {
	// Execute processes the request and returns a Response to render, or an
	// error to be handled by the configured ErrorHandler.
	Execute(context.Context, *Request[M]) (Response, error)

	// Meta returns the routing metadata (HTTP method and path) used to mount
	// the endpoint.
	Meta() Meta
}

// Mux is an HTTP router that registers handlers by pattern. It is satisfied by
// *http.ServeMux.
type Mux interface {
	// Handle registers h for the given pattern (e.g. "POST /users/{id}").
	Handle(pattern string, h http.Handler)
}

// handler is an http.Handler that serves a single mounted endpoint.
type handler[M any] struct {
	endpoint        Endpoint[M]
	errorHandler    httphandler.ErrorHandler
	validator       Validator[M]
	formDecoder     *form.Decoder
	telemetry       *inertiaotel.Config
	metrics         *metrics
	jsonOptions     []json.Options
	hasPrecognition bool
}

func (h *handler[M]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h.serve(w, r); err != nil {
		h.metrics.errorResponsesCounter.Add(r.Context(), 1)
		h.errorHandler.ServeHTTP(w, r, err)
	}
}

func (h *handler[M]) serve(w http.ResponseWriter, r *http.Request) error {
	err := h.execute(w, r)
	if !h.hasPrecognition {
		return err
	}

	req := inertiahttp.RenderScopeFromRequest(r).Request()
	if !req.Precognition {
		return err
	}

	if err != nil {
		if verr, ok := errors.AsType[inertia.ValidationErrorer](err); ok {
			return h.handlePrecognition(w, req, verr)
		}

		return err
	}

	return h.handlePrecognition(w, req, nil)
}

func (h *handler[M]) execute(w http.ResponseWriter, r *http.Request) error {
	ctx, span := h.telemetry.Tracer().Start(
		r.Context(),
		"inertiaframe.endpoint",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("inertiaframe.endpoint.method", h.endpoint.Meta().Method),
			attribute.String("inertiaframe.endpoint.path", h.endpoint.Meta().Path),
		),
	)
	defer span.End()

	r = r.WithContext(ctx)

	msg, err := h.extractRequest(r)
	if err != nil {
		return err
	}

	if h.validator != nil {
		if err := h.validator.Validate(msg); err != nil {
			d("failed to validate request")

			h.metrics.validationErrorsCounter.Add(
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
	if h.hasPrecognition && inertiaReq.Precognition {
		return nil
	}

	execStart := time.Now()
	execResp, err := h.endpoint.Execute(ctx, newRequest(msg))
	h.metrics.endpointDuration.Record(ctx, float64(time.Since(execStart).Milliseconds()))

	if err != nil {
		return fmt.Errorf("inertiaframe: failed to execute: %w", err)
	}

	if execResp == nil {
		d("received empty response")

		h.metrics.emptyResponsesCounter.Add(ctx, 1)

		return ErrEmptyResponse
	}

	span.SetAttributes(
		attribute.String("inertiaframe.response.type", responseType(execResp)),
	)

	return h.render(w, r, execResp)
}

// render writes resp to w. RawResponseWriter responses write directly to the
// client; other responses are rendered through Inertia with their props,
// options, and any session-stored validation errors.
func (h *handler[M]) render(
	w http.ResponseWriter,
	r *http.Request,
	execResp Response,
) error {
	span := trace.SpanFromContext(r.Context())

	var renderCtx inertia.RenderContext

	if writer, ok := execResp.(RawResponseWriter); ok {
		d("writing raw response for %s %s", r.Method, r.URL.Path)

		if err := writer.Write(w, r); err != nil {
			return fmt.Errorf("inertiaframe: failed to write response: %w", err)
		}

		return nil
	}

	if optioner, ok := execResp.(ResponseOptioner); ok {
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

	if proper := execResp.Proper(); proper != nil && proper.Len() > 0 {
		renderCtx.Props = proper.Props()
	}

	sess, err := sessionFromRequest(r)
	if err != nil {
		return fmt.Errorf("inertiaframe: failed to get session: %w", err)
	}

	if errs := sess.ValidationErrors(); errs != nil {
		renderCtx.ErrorBag = sess.ErrorBag()
		renderCtx.AddValidationErrorer(inertia.ValidationErrors(errs))

		h.metrics.validationErrorsCounter.Add(
			r.Context(),
			int64(len(errs)),
			metric.WithAttributes(
				attribute.String("inertiaframe.error_bag", renderCtx.ErrorBag),
			),
		)
	}

	componentName := execResp.Component()
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

func (h *handler[M]) handlePrecognition(
	w http.ResponseWriter,
	ineriaReq inertiahttp.Request,
	verr inertia.ValidationErrorer,
) error {
	header := w.Header()
	header.Set(inertiaheader.HeaderPrecognition, inertiaheader.HeaderValueTrue)
	header.Add(inertiaheader.HeaderVary, inertiaheader.HeaderPrecognition)

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
			header.Set(inertiaheader.HeaderContentType, inertiaheader.ContentTypeJSON)
			w.WriteHeader(http.StatusUnprocessableEntity)

			// Use buf here to defer write to the response to prevent partial response write.
			var buf bytes.Buffer

			buf.WriteString(`{"errors":`)

			if err := json.MarshalWrite(&buf, errorsMap, h.jsonOptions...); err != nil {
				return fmt.Errorf("inertiaframe: failed to serialize errors: %w", err)
			}

			buf.WriteByte('}')

			if _, err := buf.WriteTo(w); err != nil {
				return fmt.Errorf("inertiaframe: failed to write response: %w", err)
			}

			return nil
		}
	}

	header.Set(inertiaheader.HeaderPrecognitionSuccess, inertiaheader.HeaderValueTrue)
	w.WriteHeader(http.StatusNoContent)

	return nil
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
