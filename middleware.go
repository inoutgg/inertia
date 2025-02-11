package inertia

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"go.inout.gg/foundations/http/httpmiddleware"
	"go.inout.gg/foundations/must"
	"go.inout.gg/inertia/internal/inertiaheader"
)

type ctxKey struct{}

var kCtxKey ctxKey = ctxKey{}

// https://inertiajs.com/redirects#303-response-code
var seeOtherMethods = []string{http.MethodPatch, http.MethodPut, http.MethodDelete}

type MiddlewareConfig struct {
	// EmptyResponseHandler is a function that is called when the response is empty.
	EmptyResponseHandler http.HandlerFunc

	// OnVersionMismatch is a function that is called when the version mismatch occurs.
	OnVersionMismatch http.HandlerFunc
}

func (m *MiddlewareConfig) defaults() {
	if m.EmptyResponseHandler == nil {
		m.EmptyResponseHandler = func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Empty response", http.StatusNoContent)
		}
	}

	if m.OnVersionMismatch == nil {
		m.OnVersionMismatch = func(w http.ResponseWriter, r *http.Request) {
			Location(w, r, r.RequestURI)
		}
	}
}

// Middleware provides the HTTP handling layer for Inertia.js server-side integration.
func Middleware(renderer *Renderer, opts ...func(*MiddlewareConfig)) httpmiddleware.MiddlewareFunc {
	config := MiddlewareConfig{}
	for _, opt := range opts {
		opt(&config)
	}

	config.defaults()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			r = r.WithContext(context.WithValue(r.Context(), kCtxKey, renderer))

			h.Set(inertiaheader.HeaderVary, inertiaheader.HeaderXInertia)

			if !isInertiaRequest(r) {
				next.ServeHTTP(w, r)
				return
			}

			externalVersion := r.Header.Get(inertiaheader.HeaderXInertiaVersion)
			if externalVersion != renderer.Version() {
				Location(w, r, r.RequestURI)
				return
			}

			ww := newResponseWriter(w)
			next.ServeHTTP(ww, r)

			if ww.statusCode == http.StatusFound &&
				slices.Contains(seeOtherMethods, r.Method) {
				ww.WriteHeader(http.StatusSeeOther)
			}

			if ww.Empty() {
				config.EmptyResponseHandler(w, r)
				return
			}

			_ = ww.flush()
		})
	}
}

// Context represents an Inertia.js page context.
type Context struct {
	EncryptHistory bool
	ClearHistory   bool
	Props          []*Prop

	// T is an optional custom data that can be passed to the template.
	T any
}

// Option configures rendering context.
type Option func(*Context)

// WithClearHistory sets the history clear.
func WithClearHistory() Option {
	return func(opt *Context) { opt.ClearHistory = true }
}

// WithEncryptHistory instructs the client to encrypt the history state.
func WithEncryptHistory() Option {
	return func(opt *Context) { opt.EncryptHistory = true }
}

// WithProps sets the props for the page.
// Calling this function multiple times will append the props.
func WithProps(props Proper) Option {
	return func(opt *Context) {
		if props == nil {
			return
		}

		if opt.Props == nil {
			opt.Props = make([]*Prop, 0, props.Len())
		}

		opt.Props = append(opt.Props, props.Props()...)
	}
}

// Render sends a page component using Inertia.js protocol, allowing server-side rendering
// of components that interact seamlessly with the Inertia.js client.
func Render(w http.ResponseWriter, r *http.Request, componentName string, opts ...Option) error {
	rCtx := Context{}
	for _, opt := range opts {
		opt(&rCtx)
	}

	render, ok := r.Context().Value(kCtxKey).(*Renderer)
	if !ok {
		return errors.New("inertia: renderer not found in request context - did you forget to use the middleware?")
	}
	if err := render.Render(w, r, componentName, &rCtx); err != nil {
		return err
	}
	return nil
}

// MustRender is like Render, but panics if an error occurs.
func MustRender(w http.ResponseWriter, req *http.Request, name string, opts ...Option) {
	must.Must1(Render(w, req, name, opts...))
}
