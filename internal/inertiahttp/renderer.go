package inertiahttp

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/alitto/pond/v2"
	"github.com/go-json-experiment/json"
	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/must"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.segfaultmedaddy.com/inertia/inertiaalways"
	"go.segfaultmedaddy.com/inertia/inertiaotel"
	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprotocol"
	"go.segfaultmedaddy.com/inertia/internal/inertiassr"
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

// SSRClient communicates with a server-side rendering service to pre-render Inertia pages.
type SSRClient = inertiassr.SSRClient

// Page represents an Inertia.js page that is sent to the client.
type Page = inertiaprotocol.Page

//nolint:gochecknoglobals
var bufPool = sync.Pool{New: func() any { return bytes.NewBuffer(nil) }}

// Config configures the Renderer behavior and capabilities.
//
//nolint:govet
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

	// Telemetry configures OpenTelemetry tracing and metrics.
	//
	// If zero, telemetry is a no-op.
	Telemetry *inertiaotel.Config
}

func (c *Config) defaults() {
	if c.Telemetry == nil {
		c.Telemetry = inertiaotel.DefaultConfig
	}

	c.RootViewID = cmp.Or(c.RootViewID, DefaultRootViewID)
	c.Concurrency = cmp.Or(c.Concurrency, DefaultConcurrency)
}

type RenderScope struct {
	renderer *Renderer
	req      Request
}

func (s *RenderScope) Request() *Request { return &s.req }

func (s *RenderScope) Render(
	ctx context.Context,
	name string,
	renderCtx RenderContext,
) (Response, error) {
	return s.renderer.Render(ctx, s.req, name, renderCtx)
}

// Renderer handles Inertia.js page responses, supporting both client-side and server-side rendering.
// It manages HTML template rendering, JSON serialization, and prop resolution.
//
// Create a Renderer using New or FromFS constructor functions.
//
//nolint:govet
type Renderer struct {
	ssrClient       SSRClient
	resultPool      pond.ResultPool[inertiaprotocol.Result]
	t               *template.Template
	rootViewID      string
	version         string
	jsonMarshalOpts []json.Options
	rootViewAttrs   []pair[[]byte, []byte]
	telemetry       *inertiaotel.Config
	renderDuration  metric.Float64Histogram
	protocol        *inertiaprotocol.Renderer
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

	r := &Renderer{ //nolint:exhaustruct
		t:               t,
		ssrClient:       config.SSRClient,
		jsonMarshalOpts: config.JSONMarshalOptions,
		version:         config.Version,
		rootViewID:      config.RootViewID,
		rootViewAttrs:   attrs,
		resultPool:      pond.NewResultPool[inertiaprotocol.Result](config.Concurrency),
		telemetry:       config.Telemetry,
		protocol:        inertiaprotocol.New(config.Telemetry),
	}

	var err error

	r.renderDuration, err = config.Telemetry.Meter().Float64Histogram(
		"inertia.render.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Duration of Inertia page rendering"),
	)
	if err != nil {
		d("failed to create inertia.render.duration histogram: %v", err)
	}

	debug.Assert(r.t != nil, "expected t to be defined")
	debug.Assert(r.rootViewID != "", "expected RootViewID to be defined")
	debug.Assert(r.telemetry != nil, "expected telemetry to be defined")

	return r
}

// Version returns the current asset version string used for client version validation.
func (r *Renderer) Version() string { return r.version }

// NewScope returns a scoped render context for the given request.
func (r *Renderer) NewScope(req *http.Request) (*RenderScope, error) {
	parsedReq, err := ParseRequest(req)
	if err != nil {
		return nil, err
	}

	return &RenderScope{
		renderer: r,
		req:      parsedReq,
	}, nil
}

// Render returns an Inertia response, automatically choosing the format:
//   - JSON for Inertia requests (XHR navigation)
//   - HTML for initial page loads or non-Inertia requests
//
// The renderCtx configures props, validation errors, and other page-specific settings.
func (r *Renderer) Render(
	ctx context.Context,
	req Request,
	name string,
	renderCtx RenderContext,
) (Response, error) {
	debug.Assert(r.t != nil, "Renderer template must be set")
	debug.Assert(name != "", "component name must be non-empty")

	start := time.Now()
	requestType := requestType(req)

	ctx, span := r.telemetry.Tracer().Start(
		ctx,
		"inertia.render",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("inertia.component", name),
			attribute.String("inertia.request.type", requestType),
			attribute.String("inertia.render.type", renderType(req)),
			attribute.Bool("inertia.ssr.enabled", r.ssrClient != nil),
		),
	)

	defer func() {
		span.End()

		r.renderDuration.Record(ctx, float64(time.Since(start).Milliseconds()),
			metric.WithAttributes(attribute.String("inertia.request.type", requestType)),
		)
	}()

	rawProps := make([]Prop, 0, len(renderCtx.SharedProps)+len(renderCtx.Props)+1)
	rawProps = append(rawProps, renderCtx.SharedProps...)
	rawProps = append(rawProps, renderCtx.Props...)
	rawProps = append(rawProps, makeValidationErrors(renderCtx.ValidationErrorer, renderCtx.ErrorBag))

	page, err := r.protocol.Render(ctx, inertiaprotocol.Request{
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
		return Response{}, fmt.Errorf("inertia: an error occurred while rendering page: %w", err)
	}

	debug.Assert(page != nil, "rendered page must not be nil")

	if req.IsInertia {
		d("Received inertia request, sending JSON response: %s", req.URL)

		body, err := json.Marshal(page, r.jsonMarshalOpts...)
		if err != nil {
			return Response{}, fmt.Errorf("inertia: failed to encode JSON response: %w", err)
		}

		return Response{
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
				return Response{}, fmt.Errorf("inertia: failed to create an HTML container: %w", err)
			}

			data.InertiaBody = body
		} else {
			pageScript, err := r.makePageScript(page)
			if err != nil {
				return Response{}, fmt.Errorf("inertia: failed to create an HTML page script: %w", err)
			}

			data.InertiaHead = template.HTML(ssrData.Head)              //nolint:gosec
			data.InertiaBody = pageScript + template.HTML(ssrData.Body) //nolint:gosec
		}
	} else {
		body, err := r.makeRootView(page)
		if err != nil {
			return Response{}, fmt.Errorf("inertia: failed to create an HTML container: %w", err)
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
		return Response{}, fmt.Errorf("inertia: failed to execute HTML template: %w", err)
	}

	return Response{
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

func makeValidationErrors(errorers []inertiaprop.ValidationErrorer, errorBag string) Prop {
	m := make(map[string]string)

	for _, errorer := range errorers {
		errs := errorer.ValidationErrors()
		for _, err := range errs {
			m[err.Field()] = err.Error()
		}
	}

	if errorBag != inertiaprop.DefaultErrorBag {
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

// pair is a key-value pair.
type pair[K any, V any] struct {
	key   K
	value V
}

func renderType(req Request) string {
	if req.IsInertia {
		return "json"
	}

	return "html"
}

// requestType returns a telemetry label describing the request kind:
// "non_inertia", "partial", or "full".
func requestType(req Request) string {
	if !req.IsInertia {
		return "non_inertia"
	}

	if req.PartialComponent != "" {
		return "partial"
	}

	return "full"
}
