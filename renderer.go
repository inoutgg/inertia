package inertia

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"runtime"
	"strings"

	"github.com/alitto/pond/v2"
	"github.com/go-json-experiment/json"
	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/must"

	"go.segfaultmedaddy.com/inertia/inertiaalways"
	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprotocol"
	"go.segfaultmedaddy.com/inertia/internal/inertiaredirect"
)

const (
	// DefaultRootViewID is the default root HTML element ID to which
	// the Inertia.js app is mounted.
	DefaultRootViewID = "app"
)

// DefaultConcurrency is the default maximum concurrency for the ResultPool
// used to resolve concurrent props. A value of 0 means no limit (pond's "0 = unlimited"
// convention).
var DefaultConcurrency = runtime.GOMAXPROCS(0) //nolint:gochecknoglobals

// Page represents an Inertia.js page that is sent to the client.
type Page = inertiaprotocol.Page

// Config configures the Renderer behavior and capabilities.
type Config struct {
	// SSRClient enables server-side rendering of Inertia pages.
	//
	// If nil, only client-side rendering is used.
	SSRClient SSRClient

	// RootViewAttrs are HTML attributes applied to the root element.
	RootViewAttrs map[string]string

	// Version identifies the current asset version (e.g., build hash or timestamp).
	Version string

	// RootViewID is the HTML element ID where the Inertia app mounts.
	//
	// Defaults to "app" if not specified.
	RootViewID string

	// JSONMarshalOptions configures JSON serialization for page props and data.
	JSONMarshalOptions []json.Options

	// Concurrency sets the maximum number of props that can be resolved concurrently
	// by the renderer's internal ResultPool. It only affects props marked as concurrent.
	//
	// Defaults to runtime.GOMAXPROCS(0). A value of 0 means no limit.
	Concurrency int
}

func (c *Config) defaults() {
	c.RootViewID = cmp.Or(c.RootViewID, DefaultRootViewID)
	c.Concurrency = cmp.Or(c.Concurrency, DefaultConcurrency)

	debug.Assert(c.RootViewID != "", "RooViewID must be non-empty string")
}

// Renderer handles Inertia.js page responses, supporting both client-side and server-side rendering.
// It manages HTML template rendering, JSON serialization, and prop resolution.
//
// Create a Renderer using New or FromFS constructor functions.
type Renderer struct {
	ssrClient       SSRClient
	resultPool      pond.ResultPool[inertiaprotocol.Result]
	t               *template.Template
	rootViewID      string
	version         string
	jsonMarshalOpts []json.Options
	rootViewAttrs   []pair[[]byte, []byte]
}

// New creates a Renderer with the provided HTML template and configuration.
//
// If config is nil, default values are used:
//   - RootViewID: "app"
//   - Concurrency: GOMAXPROCS(0)
func New(t *template.Template, config *Config) *Renderer {
	if config == nil {
		//nolint:exhaustruct
		config = &Config{}
	}

	config.defaults()

	attrs := make([]pair[[]byte, []byte], 0, len(config.RootViewAttrs))
	for key, value := range config.RootViewAttrs {
		attrs = append(attrs, pair[[]byte, []byte]{[]byte(key), []byte(value)})
	}

	r := &Renderer{
		t:               t,
		ssrClient:       config.SSRClient,
		jsonMarshalOpts: config.JSONMarshalOptions,
		version:         config.Version,
		rootViewID:      config.RootViewID,
		rootViewAttrs:   attrs,
		resultPool:      pond.NewResultPool[inertiaprotocol.Result](config.Concurrency),
	}

	debug.Assert(r.t != nil, "expected t to be defined")
	debug.Assert(r.rootViewID != "", "expected RootViewID to be defined")

	return r
}

// FromFS creates a Renderer by loading an HTML template from a file system.
//
// If config is nil, default values are used.
func FromFS(fsys fs.FS, path string, config *Config) (*Renderer, error) {
	debug.Assert(fsys != nil, "expected fsys to be defined")
	debug.Assert(path != "", "expected path to be defined")

	t := template.New("inertia")

	t, err := t.ParseFS(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("inertia: failed to parse templates: %w", err)
	}

	return New(t, config), nil
}

// MustFromFS is like FromFS, but panics if an error occurs.
func MustFromFS(fsys fs.FS, path string, config *Config) *Renderer {
	return must.Must(FromFS(fsys, path, config))
}

// Version returns the current asset version string used for client version validation.
func (r *Renderer) Version() string { return r.version }

// render returns an Inertia response, automatically choosing the format:
//   - JSON for Inertia requests (XHR navigation)
//   - HTML for initial page loads or non-Inertia requests
//
// The renderCtx configures props, validation errors, and other page-specific settings.
func (r *Renderer) render(
	ctx context.Context,
	req request,
	name string,
	renderCtx RenderContext,
) (response, error) {
	debug.Assert(r.t != nil, "Renderer template must be set")
	debug.Assert(name != "", "component name must be non-empty")

	rawProps := make([]Prop, 0, len(renderCtx.SharedProps)+len(renderCtx.Props)+1)
	rawProps = append(rawProps, renderCtx.SharedProps...)
	rawProps = append(rawProps, renderCtx.Props...)
	rawProps = append(rawProps, makeValidationErrors(renderCtx.ValidationErrorer, renderCtx.ErrorBag))

	page, err := inertiaprotocol.Render(ctx, inertiaprotocol.Request{
		URL:               req.URL,
		PartialComponent:  req.PartialComponent,
		ScrollMergeIntent: req.ScrollMergeIntent,
		PartialData:       req.PartialData,
		PartialExcept:     req.PartialExcept,
		ResetProps:        req.ResetProps,
		ExceptOnceProps:   req.ExceptOnceProps,
	}, inertiaprotocol.Context{
		Component:        name,
		Version:          r.version,
		Props:            rawProps,
		SharedProps:      renderCtx.SharedProps,
		PreserveFragment: renderCtx.PreserveFragment,
		ClearHistory:     renderCtx.ClearHistory,
		EncryptHistory:   renderCtx.EncryptHistory,
		ResultPool:       r.resultPool,
	})
	if err != nil {
		return response{}, fmt.Errorf("inertia: an error occurred while rendering page: %w", err)
	}

	debug.Assert(page != nil, "rendered page must not be nil")

	if req.IsInertia {
		d("Received inertia request, sending JSON response: %s", req.URL)

		body, err := json.Marshal(page, r.jsonMarshalOpts...)
		if err != nil {
			return response{}, fmt.Errorf("inertia: failed to encode JSON response: %w", err)
		}

		return response{
			Headers: map[string]string{
				inertiaheader.HeaderXInertia:    inertiaheader.HeaderValueTrue,
				inertiaheader.HeaderContentType: inertiaheader.ContentTypeJSON,
			},
			Body: body,
		}, nil
	}

	data := TemplateData{T: renderCtx.T, InertiaHead: "", InertiaBody: ""}

	if r.ssrClient != nil {
		ssrData, err := r.ssrClient.Render(ctx, page)
		if err != nil {
			body, err := r.makeRootView(page)
			if err != nil {
				return response{}, fmt.Errorf("inertia: failed to create an HTML container: %w", err)
			}

			data.InertiaBody = body
		} else {
			pageScript, err := r.makePageScript(page)
			if err != nil {
				return response{}, fmt.Errorf("inertia: failed to create an HTML page script: %w", err)
			}

			data.InertiaHead = template.HTML(ssrData.Head)              //nolint:gosec
			data.InertiaBody = pageScript + template.HTML(ssrData.Body) //nolint:gosec
		}
	} else {
		body, err := r.makeRootView(page)
		if err != nil {
			return response{}, fmt.Errorf("inertia: failed to create an HTML container: %w", err)
		}

		data.InertiaBody = body
	}

	body := bufPool.Get().(*bytes.Buffer) //nolint:forcetypeassert
	body.Reset()

	defer func() {
		body.Reset()
		bufPool.Put(body)
	}()

	if err := r.t.Execute(body, &data); err != nil {
		return response{}, fmt.Errorf("inertia: failed to execute HTML template: %w", err)
	}

	return response{
		Headers: map[string]string{
			inertiaheader.HeaderContentType: inertiaheader.ContentTypeHTML,
		},
		Body: append([]byte(nil), body.Bytes()...),
	}, nil
}

// makeRootView creates a root view element with the given page data.
func (r *Renderer) makeRootView(page *Page) (template.HTML, error) {
	pageScript, err := r.makePageScript(page)
	if err != nil {
		return "", err
	}

	var w strings.Builder

	_ = must.Must(w.WriteString(string(pageScript)))

	_ = must.Must(w.WriteString(`<div id="`))
	_ = must.Must(w.WriteString(r.rootViewID))
	_ = must.Must(w.WriteRune('"'))
	_ = must.Must(w.WriteRune(' '))

	if r.rootViewAttrs != nil {
		for _, kv := range r.rootViewAttrs {
			// Skip generated attributes as they're already set.
			if bytes.Equal(kv.key, []byte("data-page")) || bytes.Equal(kv.key, []byte("id")) {
				continue
			}

			_ = must.Must(w.Write(kv.key))
			_ = must.Must(w.WriteRune('='))
			_ = must.Must(w.WriteRune('"'))
			template.HTMLEscape(&w, kv.value)
			_ = must.Must(w.WriteRune('"'))
			_ = must.Must(w.WriteRune(' '))
		}
	}

	_ = must.Must(w.WriteString(`></div>`))

	//nolint:gosec
	return template.HTML(w.String()), nil
}

func (r *Renderer) makePageScript(page *Page) (template.HTML, error) {
	var w strings.Builder

	_ = must.Must(w.WriteString(`<script data-page="`))
	_ = must.Must(w.WriteString(strings.TrimSpace(r.rootViewID)))
	_ = must.Must(w.WriteString(`" type="application/json">`))

	b, err := json.Marshal(page, r.jsonMarshalOpts...)
	if err != nil {
		return "", fmt.Errorf("inertia: an error occurred while rendering page: %w", err)
	}

	b = bytes.ReplaceAll(b, []byte("&"), []byte(`\u0026`))
	b = bytes.ReplaceAll(b, []byte("<"), []byte(`\u003c`))
	b = bytes.ReplaceAll(b, []byte(">"), []byte(`\u003e`))
	b = bytes.ReplaceAll(b, []byte("\u2028"), []byte(`\u2028`))
	b = bytes.ReplaceAll(b, []byte("\u2029"), []byte(`\u2029`))

	_ = must.Must(w.Write(b))
	_ = must.Must(w.WriteString(`</script>`))

	return template.HTML(w.String()), nil //nolint:gosec
}

func makeValidationErrors(errorers []ValidationErrorer, errorBag string) Prop {
	m := make(map[string]string)

	for _, errorer := range errorers {
		errs := errorer.ValidationErrors()
		for _, err := range errs {
			m[err.Field()] = err.Error()
		}
	}

	if errorBag != DefaultErrorBag {
		return inertiaalways.New(errorBag, map[string]map[string]string{"errors": m})
	}

	return inertiaalways.New("errors", m)
}

// TemplateData contains the data passed to the HTML template during rendering.
type TemplateData struct {
	// T is custom application data available to the template.
	T any

	// InertiaHead contains SSR-generated head elements (title, meta tags, etc.).
	InertiaHead template.HTML

	// InertiaBody contains the rendered page content.
	InertiaBody template.HTML
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
		h.Set(inertiaheader.HeaderXInertiaLocation, url) // redirect URL
		w.WriteHeader(http.StatusConflict)               // 409 Conflict

		return
	}

	inertiaredirect.Redirect(w, r, url)
}

// Redirect sends a redirect response to the Inertia app page.
func Redirect(w http.ResponseWriter, r *http.Request, url string) {
	debug.Assert(w != nil, "ResponseWriter must not be nil")
	debug.Assert(r != nil, "Request must not be nil")
	debug.Assert(url != "", "url must be non-empty")

	inertiaredirect.Redirect(w, r, url)
}

// RedirectPreserveFragment redirects while instructing Inertia to preserve the current URL fragment.
func RedirectPreserveFragment(w http.ResponseWriter, r *http.Request, url string) {
	debug.Assert(w != nil, "ResponseWriter must not be nil")
	debug.Assert(r != nil, "Request must not be nil")
	debug.Assert(url != "", "url must be non-empty")

	if r.Header.Get(inertiaheader.HeaderXInertia) == inertiaheader.HeaderValueTrue {
		h := w.Header()

		h.Del(inertiaheader.HeaderVary)
		h.Del(inertiaheader.HeaderXInertia)
		h.Set(inertiaheader.HeaderXInertiaRedirect, url)
		w.WriteHeader(http.StatusConflict)

		return
	}

	inertiaredirect.Redirect(w, r, url)
}

// ErrorBagFromRequest extracts the error bag name from the X-Inertia-Error-Bag header.
//
// Returns the default error bag (empty string) if the header is not present.
// Used to scope validation errors to specific forms on a page.
func ErrorBagFromRequest(r *http.Request) string {
	errorBag := r.Header.Get(inertiaheader.HeaderXInertiaErrorBag)
	if errorBag == "" {
		return DefaultErrorBag
	}

	return errorBag
}

// pair is a key-value pair.
type pair[K any, V any] struct {
	key   K
	value V
}
