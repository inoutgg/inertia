package inertia

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/must"

	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
	"go.segfaultmedaddy.com/inertia/internal/inertiahttp"
	"go.segfaultmedaddy.com/inertia/internal/inertiaredirect"
)

var (
	_ http.ResponseWriter                       = (*responseWriter)(nil)
	_ interface{ Unwrap() http.ResponseWriter } = (*responseWriter)(nil)
)

//nolint:gochecknoglobals
var bufPool = sync.Pool{New: func() any { return bytes.NewBuffer(nil) }}

// responseWriter is a wrapper around http.ResponseWriter that defer
// response writing until the flush method is called.
type responseWriter struct {
	http.ResponseWriter

	buf        *bytes.Buffer
	statusCode int
	size       int
	flushed    bool
	dirty      bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	debug.Assert(w != nil, "underlying ResponseWriter must not be nil")

	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		size:           0,
		flushed:        false,
		dirty:          false,

		//nolint:forcetypeassert
		buf: bufPool.Get().(*bytes.Buffer),
	}
}

func (w *responseWriter) WriteHeader(code int) {
	w.dirty = true
	w.statusCode = code
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.dirty = true

	n, err := w.buf.Write(b)
	w.size += n

	if err != nil {
		//nolint:wrapcheck
		return n, err
	}

	return n, nil
}

func (w *responseWriter) Empty() bool {
	if w.dirty {
		return false
	}

	return w.size == 0
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// flush writes the buffered response to the underlying http.ResponseWriter.
func (w *responseWriter) flush() {
	if w.flushed {
		return
	}

	defer func() {
		w.buf.Reset()
		bufPool.Put(w.buf)
		w.buf = nil
	}()

	w.flushed = true

	d("flush status=%d bytes=%d", w.statusCode, w.size)

	w.ResponseWriter.WriteHeader(w.statusCode)
	_, _ = w.ResponseWriter.Write(w.buf.Bytes())
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

	scope := inertiahttp.RenderScopeFromRequest(r)

	resp, err := scope.Render(r.Context(), componentName, rCtx)
	if err != nil {
		return err //nolint:wrapcheck
	}

	for key, value := range resp.Headers {
		w.Header().Set(key, value)
	}

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(resp.Body); err != nil {
		return fmt.Errorf("inertia: failed to write response: %w", err)
	}

	return nil
}

// MustRender is like Render, but panics if an error occurs.
func MustRender(w http.ResponseWriter, req *http.Request, name string, r RenderContext) {
	debug.Assert(w != nil, "ResponseWriter must not be nil")
	debug.Assert(req != nil, "Request must not be nil")
	debug.Assert(name != "", "component name must be non-empty")
	must.Must1(Render(w, req, name, r))
}

// Location redirects to an external URL outside of the Inertia app.
//
// For Inertia requests, it uses a 409 Conflict response with X-Inertia-Location header.
// For regular requests, it performs a standard HTTP redirect.
func Location(w http.ResponseWriter, r *http.Request, url string) {
	debug.Assert(w != nil, "ResponseWriter must not be nil")
	debug.Assert(r != nil, "Request must not be nil")
	debug.Assert(url != "", "url must be non-empty")

	if r.Header.Get(inertiaheader.HeaderXInertia) == inertiaheader.HeaderValueTrue {
		h := w.Header()

		h.Del(inertiaheader.HeaderVary)
		h.Del(inertiaheader.HeaderXInertia)
		h.Set(inertiaheader.HeaderXInertiaLocation, url)
		w.WriteHeader(http.StatusConflict)

		return
	}

	inertiaredirect.Redirect(w, r, url)
}

// Redirect redirects to a URL within the Inertia app, preserving the current
// URL fragment on the client.
//
// For Inertia requests, it uses a 409 Conflict response with X-Inertia-Redirect header.
// For regular requests, it performs a standard HTTP redirect.
func Redirect(w http.ResponseWriter, r *http.Request, url string, preserveFragment bool) {
	debug.Assert(w != nil, "ResponseWriter must not be nil")
	debug.Assert(r != nil, "Request must not be nil")
	debug.Assert(url != "", "url must be non-empty")

	if r.Header.Get(inertiaheader.HeaderXInertia) == inertiaheader.HeaderValueTrue {
		if preserveFragment && r.URL.Fragment != "" && !strings.Contains(url, "#") {
			url = url + "#" + r.URL.Fragment
		}

		h := w.Header()

		h.Del(inertiaheader.HeaderVary)
		h.Del(inertiaheader.HeaderXInertia)
		h.Set(inertiaheader.HeaderXInertiaRedirect, url)
		w.WriteHeader(http.StatusConflict)

		return
	}

	inertiaredirect.Redirect(w, r, url)
}
