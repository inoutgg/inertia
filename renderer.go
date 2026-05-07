package inertia

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"runtime"
	"slices"
	"strings"

	"github.com/alitto/pond/v2"
	"github.com/go-json-experiment/json"
	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/must"

	"go.segfaultmedaddy.com/inertia/internal/inertiabase"
	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
	"go.segfaultmedaddy.com/inertia/internal/inertiaredirect"
)

const (
	// DefaultRootViewID is the default root HTML element ID to which
	// the Inertia.js app is mounted.
	DefaultRootViewID = "app"

	ScrollMergeIntentAppend  = "append"
	ScrollMergeIntentPrepend = "prepend"
)

var ErrInvalidScrollMergeIntent = errors.New("inertia: invalid infinite scroll merge intent")

// DefaultConcurrency is the default concurrency level for props resolution
// marked as concurrently resolvable.
var DefaultConcurrency = runtime.GOMAXPROCS(0) //nolint:gochecknoglobals

// Page represents an Inertia.js page that is sent to the client.
type Page = inertiabase.Page

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

	// Concurrency sets the default maximum number of props that can be resolved concurrently.
	// It only affects props marked as concurrent.
	//
	// Defaults to runtime.GOMAXPROCS(0).
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
	ssrClient          SSRClient
	jsonMarshalOptions []json.Options
	t                  *template.Template
	rootViewID         string
	version            string
	rootViewAttrs      []pair[[]byte, []byte]
	concurrency        int
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
		t:                  t,
		ssrClient:          config.SSRClient,
		jsonMarshalOptions: config.JSONMarshalOptions,
		version:            config.Version,
		rootViewID:         config.RootViewID,
		rootViewAttrs:      attrs,
		concurrency:        config.Concurrency,
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
	renderCtx.Concurrency = max(cmp.Or(renderCtx.Concurrency, r.concurrency), 0)

	page, err := r.newPage(ctx, req, name, renderCtx)
	if err != nil {
		return response{}, err
	}

	if req.IsInertia {
		d("Received inertia request, sending JSON response: %s", req.URL)

		body, err := json.Marshal(page, r.jsonMarshalOptions...)
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

func (r *Renderer) newPage(
	ctx context.Context,
	req request,
	componentName string,
	renderCtx RenderContext,
) (*Page, error) {
	rawProps := make([]Prop, 0, len(renderCtx.SharedProps)+len(renderCtx.Props)+1)
	rawProps = append(rawProps, renderCtx.SharedProps...)
	rawProps = append(rawProps, renderCtx.Props...)
	rawProps = append(rawProps, makeValidationErrors(renderCtx.ValidationErrorer, renderCtx.ErrorBag))

	props, err := makeProps(ctx, req, componentName, rawProps, renderCtx.Concurrency)
	if err != nil {
		return nil, err
	}

	deferredProps := makeDeferredProps(req, componentName, rawProps)
	onceProps := makeOnceProps(rawProps)
	scrollProps := makeScrollProps(rawProps)
	mergeProps := makeMergeProps(
		rawProps,
		req.ResetProps,
		req.ScrollMergeIntent,
	)

	return &Page{
		Component:        componentName,
		Props:            props,
		DeferredProps:    deferredProps,
		ScrollProps:      scrollProps,
		OnceProps:        onceProps,
		MergeProps:       mergeProps.append,
		PrependProps:     mergeProps.prepend,
		DeepMergeProps:   mergeProps.deepMerge,
		MatchPropsOn:     mergeProps.matchOn,
		SharedProps:      makeSharedProps(renderCtx.SharedProps),
		URL:              req.URL,
		Version:          r.version,
		PreserveFragment: renderCtx.PreserveFragment,
		ClearHistory:     renderCtx.ClearHistory,
		EncryptHistory:   renderCtx.EncryptHistory,
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
	_ = must.Must(w.WriteString(r.rootViewID))
	_ = must.Must(w.WriteString(`" type="application/json">`))

	pageBytes, err := json.Marshal(page, r.jsonMarshalOptions...)
	if err != nil {
		return "", fmt.Errorf("inertia: an error occurred while rendering page: %w", err)
	}

	_ = must.Must(w.WriteString(escapeScriptJSON(pageBytes)))
	_ = must.Must(w.WriteString(`</script>`))

	return template.HTML(w.String()), nil //nolint:gosec
}

func escapeScriptJSON(b []byte) string {
	s := string(b)
	s = strings.ReplaceAll(s, "&", `\u0026`)
	s = strings.ReplaceAll(s, "<", `\u003c`)
	s = strings.ReplaceAll(s, ">", `\u003e`)
	s = strings.ReplaceAll(s, "\u2028", `\u2028`)
	s = strings.ReplaceAll(s, "\u2029", `\u2029`)

	return s
}

func makeProps(
	ctx context.Context,
	req request,
	componentName string,
	props []Prop,
	concurrency int,
) (map[string]any, error) {
	// If the request is a partial, we need to filter the props.
	if req.PartialComponent == componentName {
		return resolvePartialComponentRequest(
			ctx,
			props,
			req.PartialData,
			req.PartialExcept,
			req.ExceptOnceProps,
			concurrency,
		)
	}

	m := make(map[string]any, len(props))

	for _, prop := range props {
		if shouldSkipOnceProp(prop, req.ExceptOnceProps, nil) {
			continue
		}

		// Skip deferred and optional props on the first render.
		if shouldIgnoreFirstLoad(prop) {
			continue
		}

		val, err := prop.Value(ctx)
		if err != nil {
			return nil, fmt.Errorf("inertia: failed to resolve prop %s: %w", prop.Key(), err)
		}

		m[prop.Key()] = val
	}

	return m, nil
}

func resolvePartialComponentRequest(
	ctx context.Context,
	props []Prop,
	whitelist, blacklist, exceptOnceProps []string,
	concurrency int,
) (map[string]any, error) {
	m := make(map[string]any, len(props))
	concurrentProps := make([]Prop, 0, len(props))

	for _, prop := range props {
		key := prop.Key()
		if shouldSkipOnceProp(prop, exceptOnceProps, whitelist) {
			continue
		}

		if !shouldBypassPartialFilters(prop) {
			// It should be fine to go through slices here, as the number of props is expected to be small.
			if len(whitelist) > 0 && !slices.Contains(whitelist, key) ||
				len(blacklist) > 0 && slices.Contains(blacklist, key) {
				continue
			}
		}

		if isConcurrent(prop) {
			concurrentProps = append(concurrentProps, prop)
		} else {
			val, err := prop.Value(ctx)
			if err != nil {
				return nil, fmt.Errorf("inertia: failed to resolve prop %s: %w", prop.Key(), err)
			}

			m[key] = val
		}
	}

	if len(concurrentProps) > 0 {
		pool := pond.NewResultPool[pair[string, any]](concurrency)
		group := pool.NewGroupContext(ctx)

		for _, prop := range concurrentProps {
			group.SubmitErr(func() (pair[string, any], error) {
				var kv pair[string, any]

				val, err := prop.Value(ctx)
				if err != nil {
					return kv, fmt.Errorf(
						"inertia: failed to resolve prop %s: %w",
						prop.Key(),
						err,
					)
				}

				kv.key = prop.Key()
				kv.value = val

				return kv, nil
			})
		}

		result, err := group.Wait()
		if err != nil {
			return nil, fmt.Errorf("inertia: failed to resolve concurrent props: %w", err)
		}

		for i, prop := range concurrentProps {
			m[prop.Key()] = result[i].value
		}
	}

	return m, nil
}

// makeDeferredProps creates a map of deferred props that should be resolved
// on the client side.
func makeDeferredProps(req request, componentName string, props []Prop) map[string][]string {
	// If the request is partial, then the client already got information
	// about the deferred props in the initial request so we don't need to
	// send them again.
	if req.PartialComponent == componentName {
		return nil
	}

	m := make(map[string][]string, len(props))

	for _, prop := range props {
		deferred, ok := getDeferrable(prop)
		if !ok {
			continue
		}

		if _, ok := m[deferred.group]; !ok {
			m[deferred.group] = []string{}
		}

		m[deferred.group] = append(m[deferred.group], prop.Key())
	}

	return m
}

func makeOnceProps(props []Prop) map[string]inertiabase.OnceProp {
	m := make(map[string]inertiabase.OnceProp)

	for _, prop := range props {
		once, ok := getOnceable(prop)
		if !ok {
			continue
		}

		m[once.key] = inertiabase.OnceProp{
			Prop:      prop.Key(),
			ExpiresAt: once.expiresAt,
		}
	}

	if len(m) == 0 {
		return nil
	}

	return m
}

func shouldSkipOnceProp(prop Prop, exceptOnceProps, whitelist []string) bool {
	once, ok := getOnceable(prop)
	if !ok || once.fresh || !slices.Contains(exceptOnceProps, once.key) {
		return false
	}

	return !slices.Contains(whitelist, prop.Key())
}

func makeSharedProps(props []Prop) []string {
	if len(props) == 0 {
		return nil
	}

	sharedProps := make([]string, 0, len(props))
	for _, prop := range props {
		sharedProps = append(sharedProps, prop.Key())
	}

	return sharedProps
}

// makeMergeProps creates a list of props that should be merged instead of
// being replaced on the client side.
func makeMergeProps(props []Prop, blacklist []string, scrollMergeIntent string) mergeProps {
	var m mergeProps

	for _, prop := range props {
		merge, ok := getMergeable(prop)
		if len(blacklist) > 0 && slices.Contains(blacklist, prop.Key()) || !ok {
			continue
		}

		if scroll, ok := getScrollable(prop); ok {
			if scrollMergeIntent == ScrollMergeIntentPrepend {
				m.prepend = append(m.prepend, scroll.path)
			} else {
				m.append = append(m.append, scroll.path)
			}

			continue
		}

		switch {
		case merge.deepMerge:
			m.deepMerge = append(m.deepMerge, prop.Key())
		case merge.prepend:
			m.prepend = append(m.prepend, prop.Key())
		default:
			m.append = append(m.append, prop.Key())
		}

		for _, matchOn := range merge.matchOn {
			m.matchOn = append(m.matchOn, qualifyPropPath(prop.Key(), matchOn))
		}
	}

	return m
}

func makeScrollProps(props []Prop) map[string]inertiabase.ScrollProp {
	m := make(map[string]inertiabase.ScrollProp)

	for _, prop := range props {
		scroll, ok := getScrollable(prop)
		if !ok {
			continue
		}

		m[prop.Key()] = inertiabase.ScrollProp{
			PageName:     scroll.PageName,
			PreviousPage: scroll.PreviousPage,
			NextPage:     scroll.NextPage,
			CurrentPage:  scroll.CurrentPage,
		}
	}

	if len(m) == 0 {
		return nil
	}

	return m
}

func qualifyPropPath(propKey, path string) string {
	if path == "" || strings.HasPrefix(path, propKey+".") || path == propKey {
		return path
	}

	return propKey + "." + path
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
		return NewAlways(errorBag, map[string]map[string]string{"errors": m})
	}

	return NewAlways("errors", m)
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
	inertiaredirect.Redirect(w, r, url)
}

// RedirectPreserveFragment redirects while instructing Inertia to preserve the current URL fragment.
func RedirectPreserveFragment(w http.ResponseWriter, r *http.Request, url string) {
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

type mergeProps struct {
	append    []string
	prepend   []string
	deepMerge []string
	matchOn   []string
}
