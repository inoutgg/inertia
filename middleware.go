package inertia

import (
	"net/http"
	"slices"

	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/must"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"go.segfaultmedaddy.com/inertia/inertiaotel"
	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
	"go.segfaultmedaddy.com/inertia/internal/inertiahttp"
)

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

	// TelemetryConfig configures OpenTelemetry tracing and metrics.
	//
	// If zero, telemetry is a no-op.
	TelemetryConfig *inertiaotel.Config
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

	if m.TelemetryConfig == nil {
		m.TelemetryConfig = inertiaotel.DefaultConfig
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
			span := trace.SpanFromContext(r.Context())

			scope, err := renderer.NewScope(r)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				config.InvalidRequestHandler(w, r, err)

				return
			}

			req := scope.Request()
			r = inertiahttp.WithRenderScope(r, scope)
			h := w.Header()

			h.Add(inertiaheader.HeaderVary, inertiaheader.HeaderXInertia)

			if !req.IsInertia {
				next.ServeHTTP(w, r)
				return
			}

			serverVersion := renderer.Version()
			if r.Method == http.MethodGet && req.Version != serverVersion {
				span.SetAttributes(
					attribute.Bool("inertia.version.mismatch", req.Version != renderer.Version()),
				)

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
type RenderContext = inertiahttp.RenderContext

// Render sends an Inertia.js page response with the specified component and context.
// It automatically detects whether to send JSON (for Inertia requests) or HTML (for full page loads).
//
// This function requires the Inertia middleware to be installed in the request chain.
// Returns an error if the middleware is not found or if rendering fails.
func Render(w http.ResponseWriter, r *http.Request, componentName string, rCtx RenderContext) error {
	debug.Assert(w != nil, "ResponseWriter must not be nil")
	debug.Assert(r != nil, "Request must not be nil")
	debug.Assert(componentName != "", "component name must be non-empty")

	scope := inertiahttp.RenderScopeFromRequest(r)

	resp, err := scope.Render(r.Context(), componentName, rCtx)
	if err != nil {
		return err //nolint:wrapcheck
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
