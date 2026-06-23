package inertiaframe

import (
	"net/http"

	"go.inout.gg/foundations/debug"

	"go.segfaultmedaddy.com/inertia"
	"go.segfaultmedaddy.com/inertia/internal/inertiaredirect"
)

var (
	_ RawResponseWriter = (*redirectMessage)(nil)
	_ RawResponseWriter = (*redirectBackMessage)(nil)
	_ RawResponseWriter = (*externalRedirectMessage)(nil)
	_ Response          = (*resp)(nil)
)

// ResponseOptions configures Inertia client-side behavior for a response.
type ResponseOptions struct {
	// ClearHistory instructs the client to clear its navigation history stack
	// when processing this response.
	ClearHistory bool

	// EncryptHistory instructs the client to encrypt the history state for this
	// visit, so sensitive data is not persisted in plaintext.
	EncryptHistory bool
}

// ResponseOption configures a Response's options.
type ResponseOption func(*ResponseOptions)

// Response represents the result of executing an Endpoint.
//
// Most responses render a frontend component, optionally with props. A Response
// may also implement RawResponseWriter to bypass Inertia rendering and write
// directly to the HTTP response (e.g., for file downloads or non-Inertia
// endpoints).
type Response interface {
	// Component returns the frontend component name to render.
	// May be empty for responses that implement RawResponseWriter.
	Component() string

	// Proper returns the props to pass to the rendered component.
	// May be nil when no props are needed (e.g., redirect responses).
	Proper() inertia.Proper
}

// NewResponse creates a Response that renders the given frontend component with
// the provided props.
//
// component must be non-empty. proper may be nil if the component needs no
// props. Optional ResponseOption values customize history behavior.
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

type resp struct {
	proper    inertia.Proper
	component string
	opts      ResponseOptions
}

func (r *resp) Component() string        { return r.component }
func (r *resp) Proper() inertia.Proper   { return r.proper }
func (r *resp) Options() ResponseOptions { return r.opts }

type rawResp struct{ h Handler }

// NewRawResponse creates a Response that bypasses Inertia rendering and gives
// the provided Handler full control over the HTTP response.
//
// Use this for non-Inertia responses such as file downloads,
// custom authentication flows, etc. Output is written directly to the client without
// any Inertia-specific headers or rendering.
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

// NewExternalRedirectResponse creates a Response that redirects the client to
// an external URL outside the Inertia app.
//
// The redirect uses Inertia's location mechanism so the client performs a full
// page navigation rather than an in-app visit. Use this for redirecting to
// external domains.
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

// NewRedirectBackResponse creates a Response that redirects the client to the
// previous page, determined by the Referer header or the session-stored path.
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

// NewRedirectResponse creates a Response that redirects the client to the given
// URL within the Inertia app.
func NewRedirectResponse(url string) Response {
	return &redirectMessage{url: url}
}

func (m *redirectMessage) Proper() inertia.Proper { return nil }
func (m *redirectMessage) Component() string      { return "" }

func (m *redirectMessage) Write(w http.ResponseWriter, r *http.Request) error {
	inertiaredirect.Redirect(w, r, m.url)
	return nil
}

// RawResponseWriter is an optional interface that a Response can implement to
// bypass Inertia rendering and write directly to the HTTP response.
//
// When a Response implements this interface, Write is called instead of
// rendering a component. Useful for downloads, API responses, and other
// non-Inertia output.
type RawResponseWriter interface {
	// Write writes the response to the provided ResponseWriter. Return an error
	// to trigger error handling.
	Write(http.ResponseWriter, *http.Request) error
}

// ResponseOptioner is an optional interface that a Response can implement to
// supply ResponseOptions such as history management.
//
// When implemented, Options is called during rendering to read the response's
// history configuration.
type ResponseOptioner interface {
	Options() ResponseOptions
}
