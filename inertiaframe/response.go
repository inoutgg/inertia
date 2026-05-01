package inertiaframe

import (
	"fmt"
	"net/http"

	"go.segfaultmedaddy.com/inertia"
	"go.segfaultmedaddy.com/inertia/internal/inertiaredirect"
)

// resp represents a response to an Inertia request.
//
// It is a helper that implements the Response interface and is used
// to create a response from a struct or an inertia.Proper.
type resp struct {
	proper    inertia.Proper
	component string
	opts      ResponseOptions
}

// NewStructResponse creates a Response by parsing inertia struct tags on m.
// See inertia.ParseStruct for tag documentation.
// Returns an error if the struct tags are invalid.
func NewStructResponse(component string, m any, opts ...ResponseOption) (*resp, error) {
	proper, err := inertia.ParseStruct(m)
	if err != nil {
		return nil, fmt.Errorf("inertiaframe: failed to parse props: %w", err)
	}

	var options ResponseOptions

	if len(opts) > 0 {
		for _, opt := range opts {
			opt(&options)
		}
	}

	options.defaults()

	return &resp{proper, component, options}, nil
}

// NewResponse creates a Response with the specified component and props.
// Optional ResponseOption functions can customize history and concurrency behavior.
func NewResponse(component string, proper inertia.Proper, opts ...ResponseOption) *resp {
	var options ResponseOptions

	if len(opts) > 0 {
		for _, opt := range opts {
			opt(&options)
		}
	}

	options.defaults()

	return &resp{proper, component, options}
}

// Component returns the name of the client component to render.
func (r *resp) Component() string { return r.component }

// Proper returns a Proper that returns a set of props required by client.
func (r *resp) Proper() inertia.Proper { return r.proper }

// Options represents some optional parameters for inertia response such as
// history encryption, etc.
func (r *resp) Options() ResponseOptions { return r.opts }

type rawResp struct{ h Handler }

// NewRawResponse creates a Response that bypasses Inertia rendering.
// The provided handler has full control over the HTTP response.
// Useful for file downloads, API endpoints, or custom authentication flows.
func NewRawResponse(h Handler) Response {
	return &rawResp{h}
}

func (*rawResp) Component() string      { return "<raw>" }
func (*rawResp) Proper() inertia.Proper { return nil }

func (*rawResp) Options() ResponseOptions {
	var opts ResponseOptions
	return opts
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
