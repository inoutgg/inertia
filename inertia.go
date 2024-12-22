// Package inertia implements the protocol for communication with
// the Inertia.js client-side framework.
//
// For detailed protocol documentation, visit https://inertiajs.com/the-protocol
package inertia

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
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

type ctxKey struct{}

var kCtxKey ctxKey = ctxKey{}

// Page represents an Inertia.js page that is sent to the client.
type Page struct {
	Component      string              `json:"component"`
	Props          map[string]any      `json:"props"`
	URL            string              `json:"url"`
	Version        string              `json:"version"`
	EncryptHistory bool                `json:"encryptHistory"`
	ClearHistory   bool                `json:"clearHistory"`
	DeferredProps  map[string][]string `json:"deferredProps,omitempty"`
	MergeProps     []string            `json:"mergeProps,omitempty"`
}

// Config represents the configuration for the Renderer.
type Config struct {
	// Version represents a version of the inertia build.
	Version string

	// RootViewID is an ID of the root HTML element.
	// Default is "app".
	RootViewID string

	// SsrClient is a server-side rendering client.
	// If SsrClient is nil, the server-side rendering is disabled.
	SsrClient SsrClient
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
		config = &Config{}
	}
	config.defaults()

	r := &Renderer{
		t:          t,
		ssrClient:  config.SsrClient,
		version:    config.Version,
		rootViewID: config.RootViewID,
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
	t          *template.Template
	ssrClient  SsrClient
	rootViewID string
	version    string
}

func (r *Renderer) newPage(req *http.Request, componentName string, opts *Options) *Page {
	rawProps := opts.Props
	props := r.makeProps(req, componentName, rawProps)
	deferredProps := r.makeDefferedProps(req, componentName, rawProps)
	mergeProps := r.makeMergeProps(
		req,
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
		ClearHistory:   opts.ClearHistory,
		EncryptHistory: opts.EncryptHistory,
	}
}

// Version returns a version of the inertia build.
func (r *Renderer) Version() string { return r.version }

// Render sends a page component using Inertia.js protocol.
// If the request is an Inertia.js request, the response will be JSON,
// otherwise, it will be an HTML response.
func (r *Renderer) Render(w http.ResponseWriter, req *http.Request, name string, opts *Options) error {
	p := r.newPage(req, name, opts)

	if isInertiaRequest(req) {
		w.Header().Set(inertiaheader.HeaderXInertia, "true")
		w.Header().Set(inertiaheader.HeaderContentType, contentTypeJSON)
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(p); err != nil {
			return fmt.Errorf("inertia: failed to encode JSON response: %w", err)
		}

		return nil
	}

	w.Header().Set(inertiaheader.HeaderContentType, contentTypeHTML)
	w.WriteHeader(http.StatusOK)

	data := TemplateData{T: opts.T}
	if r.ssrClient != nil {
		ssrData, err := r.ssrClient.Render(p)
		if err != nil {
			return err
		}

		data.InertiaHead = template.HTML(ssrData.Head)
		data.InertiaBody = template.HTML(ssrData.Body)
	} else {
		body, err := r.makeRootView(p)
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
func (r *Renderer) makeRootView(p *Page) (template.HTML, error) {
	var w bytes.Buffer

	_ = must.Must(w.WriteString(`<div id="`))
	_ = must.Must(w.WriteString(r.rootViewID))
	_ = must.Must(w.WriteString(`" `))
	_ = must.Must(w.WriteString(`data-page="`))

	pageBytes, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("inertia: an error occurred while rendering page: %w", err)
	}

	template.HTMLEscape(&w, pageBytes)
	_ = must.Must(w.WriteString(`"></div>`))

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

		for _, p := range props {
			k := p.key
			if p.ignorable {
				// It should be fine to go through slices here, as the number of props is expected to be small.
				if !slices.Contains(whitelist, k) ||
					slices.Contains(blacklist, k) {
					continue
				}
			}

			m[k] = p.value()
		}
	} else {
		for _, p := range props {
			// Skip lazy (deferred, optional) props on first render.
			if p.lazy {
				continue
			}

			m[p.key] = p.value()
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
	for _, p := range props {
		if !p.deferred {
			continue
		}

		if _, ok := m[p.group]; !ok {
			m[p.group] = []string{}
		}

		m[p.group] = append(m[p.group], p.key)
	}

	return m
}

// makeMergeProps creates a list of props that should be merged instead of
// being replaced on the client side.
func (r *Renderer) makeMergeProps(req *http.Request, props []*Prop, blacklist []string) []string {
	mergeProps := make([]string, 0, len(props))
	for _, p := range props {
		if slices.Contains(blacklist, p.key) && !p.mergeable {
			continue
		}

		mergeProps = append(mergeProps, p.key)
	}

	return mergeProps
}

// TemplateData represents the data that is passed to the HTML template.
type TemplateData struct {
	// InertiaHead is an HTML fragment that will be injected into the <head> tag.
	InertiaHead template.HTML

	// InertiaBody is an HTML fragment that will be injected into the <body> tag.
	InertiaBody template.HTML

	// T is an optional custom data that can be passed to the template.
	// It is copied from the Options struct to the template context.
	T any
}

// Options represents rendering options.
type Options struct {
	EncryptHistory bool
	ClearHistory   bool
	Props          []*Prop

	// T is an optional custom data that can be passed to the template.
	T any
}

// Option represents an optional function that can be used to configurate the rendering process.
type Option func(*Options)

// WithClearHistory sets the history clear.
func WithClearHistory() Option {
	return func(opt *Options) { opt.ClearHistory = true }
}

// WithEncryptHistory instructs the client to encrypt the history state.
func WithEncryptHistory() Option {
	return func(opt *Options) { opt.EncryptHistory = true }
}

// WithProps sets the props for the page.
func WithProps(props ...*Prop) Option {
	return func(opt *Options) { opt.Props = props }
}

// Render sends a page component using Inertia.js protocol, allowing server-side rendering
// of components that interact seamlessly with the Inertia.js client.
func Render(w http.ResponseWriter, r *http.Request, componentName string, opts ...Option) error {
	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}

	render, ok := r.Context().Value(kCtxKey).(Renderer)
	if !ok {
		return errors.New("inertia: renderer not found in request context - did you forget to use the middleware?")
	}
	if err := render.Render(w, r, componentName, &options); err != nil {
		return err
	}
	return nil
}

// MustRender is like Render, but panics if an error occurs.
func MustRender(w http.ResponseWriter, req *http.Request, name string, opts ...Option) {
	must.Must1(Render(w, req, name, opts...))
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

// extractHeaderValueList extracts a list of values from a comma-separated inertiaheader.Header value.
func extractHeaderValueList(h string) []string {
	if h == "" {
		return []string{}
	}

	fields := strings.Split(h, ",")
	for i, f := range fields {
		fields[i] = strings.TrimSpace(f)
	}

	return fields
}
