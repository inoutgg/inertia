// Package inertia implements the protocol for communication with
// the Inertia.js client-side framework.
//
// For detailed protocol documentation, visit https://inertiajs.com/the-protocol
package inertia

import (
	"cmp"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"slices"
	"strings"

	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/must"

	"go.inout.gg/inertia/internal/inertiaheader"
)

const (
	contentTypeHTML = "text/html"
	contentTypeJSON = "application/json"
)

const DefaultRootViewID = "app"

// Page represents an Inertia.js page that is sent to the client.
type Page struct {
	Props          map[string]any      `json:"props"`
	DeferredProps  map[string][]string `json:"deferredProps,omitempty"`
	Component      string              `json:"component"`
	URL            string              `json:"url"`
	Version        string              `json:"version"`
	MergeProps     []string            `json:"mergeProps,omitempty"`
	EncryptHistory bool                `json:"encryptHistory"`
	ClearHistory   bool                `json:"clearHistory"`
}

// Config represents the configuration for the Renderer.
type Config struct {
	SsrClient     SsrClient
	RootViewAttrs map[string]string
	Version       string
	RootViewID    string
}

// defaults sets the default values for the configuration.
func (c *Config) defaults() {
	c.RootViewID = cmp.Or(c.RootViewID, DefaultRootViewID)
}

// New creates a new Renderer instance.
//
// If config is nil, the default configuration is used.
func New(t *template.Template, config *Config) *Renderer {
	if config == nil {
		//nolint:exhaustruct
		config = &Config{}
	}

	config.defaults()

	r := &Renderer{
		t:             t,
		ssrClient:     config.SsrClient,
		version:       config.Version,
		rootViewID:    config.RootViewID,
		rootViewAttrs: config.RootViewAttrs,
	}

	debug.Assert(r.t != nil, "expected t to be defined")
	debug.Assert(r.rootViewID != "", "expected RootViewID to be defined")

	return r
}

// FromFS creates a new Renderer instance from the given file system.
// If the config is nil, the default configuration is used.
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

// Renderer is a renderer that sends Inertia.js responses.
// It uses html/template to render HTML responses.
// Optionally, it supports server-side rendering using a SsrClient.
//
// To create a new Renderer, use the New or FromFS functions.
type Renderer struct {
	t *template.Template

	ssrClient     SsrClient
	rootViewID    string
	rootViewAttrs map[string]string
	version       string
}

func (r *Renderer) newPage(req *http.Request, componentName string, rCtx *RenderContext) *Page {
	rawProps := make([]*Prop, 0, len(rCtx.props)+1)
	rawProps = append(rawProps, rCtx.props...)
	rawProps = append(rawProps, r.makeValidationErrors(req, rCtx.validationErrorer))

	props := r.makeProps(req, componentName, rawProps)
	deferredProps := r.makeDefferedProps(req, componentName, rawProps)
	mergeProps := r.makeMergeProps(
		rawProps,
		extractHeaderValueList(req.Header.Get(inertiaheader.HeaderXInertiaReset)),
	)

	return &Page{
		Component:      componentName,
		Props:          props,
		DeferredProps:  deferredProps,
		MergeProps:     mergeProps,
		URL:            req.RequestURI,
		Version:        r.version,
		ClearHistory:   rCtx.clearHistory,
		EncryptHistory: rCtx.encryptHistory,
	}
}

// Version returns a version of the inertia build.
func (r *Renderer) Version() string { return r.version }

// Render sends a page component using Inertia.js protocol.
// If the request is an Inertia.js request, the response will be JSON,
// otherwise, it will be an HTML response.
func (r *Renderer) Render(w http.ResponseWriter, req *http.Request, name string, renderCtx *RenderContext) error {
	page := r.newPage(req, name, renderCtx)

	if isInertiaRequest(req) {
		w.Header().Set(inertiaheader.HeaderXInertia, "true")
		w.Header().Set(inertiaheader.HeaderContentType, contentTypeJSON)
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(page); err != nil {
			return fmt.Errorf("inertia: failed to encode JSON response: %w", err)
		}

		return nil
	}

	w.Header().Set(inertiaheader.HeaderContentType, contentTypeHTML)
	w.WriteHeader(http.StatusOK)

	data := TemplateData{T: renderCtx.T, InertiaHead: "", InertiaBody: ""}

	if r.ssrClient != nil {
		ssrData, err := r.ssrClient.Render(req.Context(), page)
		if err != nil {
			return fmt.Errorf("inertia: failed to render SSR data: %w", err)
		}

		data.InertiaHead = template.HTML(ssrData.Head) //nolint:gosec
		data.InertiaBody = template.HTML(ssrData.Body) //nolint:gosec
	} else {
		body, err := r.makeRootView(page)
		if err != nil {
			return fmt.Errorf("inertia: failed to create an HTML container: %w", err)
		}

		data.InertiaBody = body
	}

	if err := r.t.Execute(w, &data); err != nil {
		return fmt.Errorf("inertia: failed to execute HTML template: %w", err)
	}

	return nil
}

// makeRootView creates a root view element with the given page data.
func (r *Renderer) makeRootView(page *Page) (template.HTML, error) {
	var w strings.Builder

	_ = must.Must(w.WriteString(`<div id="`))
	_ = must.Must(w.WriteString(r.rootViewID))
	_ = must.Must(w.WriteString(`" `))

	_ = must.Must(w.WriteString(`data-page="`))

	pageBytes, err := json.Marshal(page)
	if err != nil {
		return "", fmt.Errorf("inertia: an error occurred while rendering page: %w", err)
	}

	template.HTMLEscape(&w, pageBytes)
	_ = must.Must(w.WriteString(`" `))

	if r.rootViewAttrs != nil {
		for k, v := range r.rootViewAttrs {
			// Skip the data-page attribute as it's already set.
			if k == "data-page" {
				continue
			}

			_ = must.Must(w.WriteString(k))
			_ = must.Must(w.WriteString(`="`))
			template.HTMLEscape(&w, []byte(v))
			_ = must.Must(w.WriteString(`" `))
		}
	}

	_ = must.Must(w.WriteString(`></div>`))

	//nolint:gosec
	return template.HTML(w.String()), nil
}

func (r *Renderer) makeProps(req *http.Request, componentName string, props []*Prop) map[string]any {
	m := make(map[string]any, len(props))

	// If the request is a partial, we need to filter the props.
	if isPartialComponentRequest(req, componentName) {
		whitelist := extractHeaderValueList(req.Header.Get(
			inertiaheader.HeaderXInertiaPartialData))
		blacklist := extractHeaderValueList(req.Header.Get(
			inertiaheader.HeaderXInertiaPartialExcept))

		for _, prop := range props {
			key := prop.key
			if prop.ignorable {
				// It should be fine to go through slices here, as the number of props is expected to be small.
				if len(whitelist) > 0 && !slices.Contains(whitelist, key) ||
					len(blacklist) > 0 && slices.Contains(blacklist, key) {
					continue
				}
			}

			m[key] = prop.value()
		}
	} else {
		for _, prop := range props {
			// Skip lazy (deferred, optional) props on the first render.
			if prop.lazy {
				continue
			}

			m[prop.key] = prop.value()
		}
	}

	return m
}

// makeDefferedProps creates a map of deferred props that should be resolved
// on the client side.
func (r *Renderer) makeDefferedProps(req *http.Request, componentName string, props []*Prop) map[string][]string {
	// If the request is partial, then the client already got information
	// about the deferred props in the initial request so we don't need to
	// send them again.
	if isPartialComponentRequest(req, componentName) {
		return nil
	}

	m := make(map[string][]string, len(props))

	for _, prop := range props {
		if !prop.deferred {
			continue
		}

		if _, ok := m[prop.group]; !ok {
			m[prop.group] = []string{}
		}

		m[prop.group] = append(m[prop.group], prop.key)
	}

	return m
}

// makeMergeProps creates a list of props that should be merged instead of
// being replaced on the client side.
func (r *Renderer) makeMergeProps(props []*Prop, blacklist []string) []string {
	mergeProps := make([]string, 0, len(props))

	for _, p := range props {
		if len(blacklist) > 0 && slices.Contains(blacklist, p.key) || !p.mergeable {
			continue
		}

		mergeProps = append(mergeProps, p.key)
	}

	return mergeProps
}

func (r *Renderer) makeValidationErrors(req *http.Request, errorers []ValidationErrorer) *Prop {
	errorBag := extractErrorBag(req)
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

// TemplateData represents the data that is passed to the HTML template.
type TemplateData struct {
	T           any
	InertiaHead template.HTML
	InertiaBody template.HTML
}

// Location sends a redirect response to the client.
func Location(w http.ResponseWriter, r *http.Request, url string) {
	if isInertiaRequest(r) {
		h := w.Header()

		h.Del(inertiaheader.HeaderVary)
		h.Del(inertiaheader.HeaderXInertia)
		h.Set(inertiaheader.HeaderXInertiaLocation, url) // redirect URL
		w.WriteHeader(http.StatusConflict)               // 409 Conflict

		return
	}

	http.Redirect(w, r, url, http.StatusFound)
}

// isInertiaRequest checks if the request is made by Inertia.js.
func isInertiaRequest(req *http.Request) bool {
	return req.Header.Get(inertiaheader.HeaderXInertia) == "true"
}

// isPartialComponentRequest checks if the request is a partial component request
// matching the given componentName.
func isPartialComponentRequest(req *http.Request, componentName string) bool {
	return req.Header.Get(inertiaheader.HeaderXInertiaPartialComponent) == componentName
}

func extractErrorBag(req *http.Request) string {
	errorBag := req.Header.Get(inertiaheader.HeaderXInertiaErrorBag)
	if errorBag == "" {
		return DefaultErrorBag
	}

	return errorBag
}

// extractHeaderValueList extracts a list of values from a comma-separated inertiaheader.Header value.
func extractHeaderValueList(h string) []string {
	if h == "" {
		return nil
	}

	fields := strings.Split(h, ",")
	for i, f := range fields {
		fields[i] = strings.TrimSpace(f)
	}

	return fields
}
