package inertia

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/must"

	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
)

type (
	ctxKey struct{}
)

//nolint:gochecknoglobals
var kCtxKey = ctxKey{}

// https://inertiajs.com/redirects#303-response-code
//
//nolint:gochecknoglobals
var seeOtherMethods = []string{http.MethodPatch, http.MethodPut, http.MethodDelete}

// DefaultEmptyResponseHandler is invoked by NewMiddleware when an inner
// handler produced no response body; it writes HTTP 204 No Content with the
// body "Empty response".
//
//nolint:gochecknoglobals
var DefaultEmptyResponseHandler = func(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Empty response", http.StatusNoContent)
}

// DefaultVersionMismatchHandler is invoked by NewMiddleware when a GET
// request's X-Inertia-Version header does not match the renderer's version;
// it performs an external redirect back to the request URL so the client
// reloads the page with fresh assets.
//
//nolint:gochecknoglobals
var DefaultVersionMismatchHandler = func(w http.ResponseWriter, r *http.Request) {
	Location(w, r, r.RequestURI)
}

// DefaultInvalidRequestHandler is invoked by NewMiddleware when an Inertia
// request fails header validation; it writes HTTP 400 Bad Request with the
// validation error as the body.
//
//nolint:gochecknoglobals
var DefaultInvalidRequestHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusBadRequest)
}

// MiddlewareConfig configures the behavior of the Inertia.js middleware.
type MiddlewareConfig struct {
	// EmptyResponseHandler is called when a handler produces no response body.
	//
	// If nil, defaults to returning HTTP 204 No Content with an error message.
	EmptyResponseHandler http.HandlerFunc

	// VersionMismatchHandler is called when the client's asset version doesn't match the server's.
	//
	// If nil, defaults to redirecting the client to the current URL to reload the page with fresh assets.
	VersionMismatchHandler http.HandlerFunc

	// InvalidRequestHandler is called before the handler when Inertia request headers are invalid.
	InvalidRequestHandler func(http.ResponseWriter, *http.Request, error)
}

func (m *MiddlewareConfig) defaults() {
	if m.EmptyResponseHandler == nil {
		m.EmptyResponseHandler = DefaultEmptyResponseHandler
	}

	if m.VersionMismatchHandler == nil {
		m.VersionMismatchHandler = DefaultVersionMismatchHandler
	}

	if m.InvalidRequestHandler == nil {
		m.InvalidRequestHandler = DefaultInvalidRequestHandler
	}

	debug.Assert(m.EmptyResponseHandler != nil, "EmptyResponseHandler must be set")
	debug.Assert(m.VersionMismatchHandler != nil, "VersionMismatchHandler must be set")
	debug.Assert(m.InvalidRequestHandler != nil, "InvalidRequestHandler must be set")
}

// NewMiddleware creates an HTTP middleware that enables Inertia.js protocol handling.
// It intercepts requests to determine if they are Inertia requests, handles version validation,
// and manages response formatting (JSON for subsequent Inertia requests, HTML otherwise).
//
// The middleware automatically handles HTTP 302 redirects by converting them to 303 for PUT/PATCH/DELETE
// requests as per the Inertia.js specification.
//
// Once the middleware is set up, Render can be used to create Inertia responses.
func NewMiddleware(renderer *Renderer, opts ...func(*MiddlewareConfig)) func(http.Handler) http.Handler {
	debug.Assert(renderer != nil, "Renderer must not be nil")

	var config MiddlewareConfig
	for _, opt := range opts {
		opt(&config)
	}

	config.defaults()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			req, err := parseRequest(r)
			if err != nil {
				config.InvalidRequestHandler(w, r, err)
				return
			}

			h := w.Header()
			r = r.WithContext(context.WithValue(r.Context(), kCtxKey, renderer))

			h.Set(inertiaheader.HeaderVary, inertiaheader.HeaderXInertia)

			if !req.IsInertia {
				next.ServeHTTP(w, r)
				return
			}

			serverVersion := renderer.Version()
			if r.Method == http.MethodGet && req.Version != serverVersion {
				d("version mismatch: client=%q server=%q for %s",
					req.Version, serverVersion, r.URL.Path)
				config.VersionMismatchHandler(w, r)

				return
			}

			rww := newResponseWriter(w)
			next.ServeHTTP(rww, r)

			if rww.statusCode == http.StatusFound &&
				slices.Contains(seeOtherMethods, r.Method) {
				d("upgrading 302 -> 303 for %s %s", r.Method, r.URL.Path)
				rww.WriteHeader(http.StatusSeeOther)
			}

			if rww.Empty() {
				d("empty response from handler for %s %s",
					r.Method, r.URL.Path)
				config.EmptyResponseHandler(w, r)

				return
			}

			rww.flush()
		})
	}
}

// RenderContext contains all configuration and data for rendering an Inertia.js page response.
// It includes props, validation errors, history management options, and performance settings.
type RenderContext struct {
	// T is custom data passed to the HTML template via html/template.
	T any

	// Props are the properties sent to the page component.
	Props []Prop

	// SharedProps are globally shared properties sent to the page component.
	SharedProps []Prop

	// ErrorBag specifies the validation error bag name for scoped error handling.
	ErrorBag string

	// ValidationErrorer contains validation errors to be sent to the client.
	ValidationErrorer []ValidationErrorer

	// EncryptHistory instructs the client to encrypt the history state for this page.
	EncryptHistory bool

	// ClearHistory instructs the client to clear the history stack.
	ClearHistory bool

	// PreserveFragment instructs the client to preserve the current URL fragment.
	PreserveFragment bool
}

// NewRenderContext creates a RenderContext configured with the provided options.
// Options are applied in order and can be combined to build up the desired page state.
func NewRenderContext(opts ...Option) RenderContext {
	var ctx RenderContext
	for _, opt := range opts {
		opt(&ctx)
	}

	return ctx
}

// AddValidationErrorer appends validation errors to the context.
// Multiple calls accumulate errors into a single error bag.
func (ctx *RenderContext) AddValidationErrorer(err ValidationErrorer) {
	if ctx.ValidationErrorer == nil {
		ctx.ValidationErrorer = make([]ValidationErrorer, 0, 1)
	}

	ctx.ValidationErrorer = append(ctx.ValidationErrorer, err)
}

// Option is a function that configures a RenderContext.
type Option func(*RenderContext)

// WithClearHistory instructs the client to clear its history stack when rendering this page.
func WithClearHistory() Option {
	return func(opt *RenderContext) { opt.ClearHistory = true }
}

// WithEncryptHistory instructs the client to encrypt the history state.
func WithEncryptHistory() Option {
	return func(opt *RenderContext) { opt.EncryptHistory = true }
}

// WithPreserveFragment instructs the client to preserve the current URL fragment.
func WithPreserveFragment() Option {
	return func(opt *RenderContext) { opt.PreserveFragment = true }
}

// WithProps adds properties to the page component.
//
// Multiple calls append additional props to the existing set.
func WithProps(props Proper) Option {
	return func(renderCtx *RenderContext) {
		if props == nil {
			return
		}

		if renderCtx.Props == nil {
			renderCtx.Props = make([]Prop, 0, props.Len())
		}

		renderCtx.Props = append(renderCtx.Props, props.Props()...)
	}
}

// WithSharedProps adds shared properties to the page component.
func WithSharedProps(props Proper) Option {
	return func(renderCtx *RenderContext) {
		if props == nil {
			return
		}

		if renderCtx.SharedProps == nil {
			renderCtx.SharedProps = make([]Prop, 0, props.Len())
		}

		renderCtx.SharedProps = append(renderCtx.SharedProps, props.Props()...)
	}
}

// WithValidationErrors adds validation errors to be displayed on the page.
// Multiple calls append errors to the same or different error bags.
//
// The errorBag parameter allows scoping errors to specific forms on the same page.
func WithValidationErrors(errorers ValidationErrorer, errorBag string) Option {
	return func(renderCtx *RenderContext) {
		if errorers == nil {
			return
		}

		renderCtx.AddValidationErrorer(errorers)
		renderCtx.ErrorBag = errorBag
	}
}

// Render sends an Inertia.js page response with the specified component and context.
// It automatically detects whether to send JSON (for Inertia requests) or HTML (for full page loads).
//
// This function requires the Inertia middleware to be installed in the request chain.
// Returns an error if the middleware is not found or if rendering fails.
func Render(w http.ResponseWriter, r *http.Request, componentName string, rCtx RenderContext) error {
	debug.Assert(w != nil, "ResponseWriter must not be nil")
	debug.Assert(r != nil, "Request must not be nil")
	debug.Assert(componentName != "", "component name must be non-empty")

	render, ok := r.Context().Value(kCtxKey).(*Renderer)
	if !ok {
		return errors.New(
			"inertia: renderer not found in request context - did you forget to use the middleware?",
		)
	}

	req, err := parseRequest(r)
	if err != nil {
		return err
	}

	resp, err := render.render(r.Context(), req, componentName, rCtx)
	if err != nil {
		return err
	}

	for key, value := range resp.Headers {
		w.Header().Set(key, value)
	}

	w.WriteHeader(http.StatusOK)
	//nolint:gosec // The renderer generated this response body from trusted server-side templates/JSON encoding.
	must.Must(w.Write(resp.Body))

	return nil
}

// MustRender is like Render, but panics if an error occurs.
func MustRender(w http.ResponseWriter, req *http.Request, name string, r RenderContext) {
	debug.Assert(w != nil, "ResponseWriter must not be nil")
	debug.Assert(req != nil, "Request must not be nil")
	debug.Assert(name != "", "component name must be non-empty")

	must.Must1(Render(w, req, name, r))
}
