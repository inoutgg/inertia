// Package vite provides Vite integration for Inertia.js applications.
package vite

import (
	"cmp"
	"fmt"
	"html/template"
	"io/fs"

	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/must"
)

// DefaultViteAddress is the Vite dev server address used by Config when
// ViteAddress is empty.
const DefaultViteAddress = "http://localhost:5173"

// Config configures Vite integration: the production manifest used to resolve
// assets, the html/template name to register, and the dev server base URL.
type Config struct {
	// Manifest maps entry points to compiled assets and is used in production
	// builds to resolve viteResource calls; ignored in dev.
	Manifest Manifest

	// TemplateName is the html/template name given to the parsed template.
	//
	// Defaults to "inertia".
	TemplateName string

	// ViteAddress is the base URL of the Vite dev server, used in
	// non-production builds.
	//
	// Defaults to DefaultViteAddress.
	ViteAddress string
}

func (c *Config) defaults() {
	c.ViteAddress = cmp.Or(c.ViteAddress, DefaultViteAddress)
	c.TemplateName = cmp.Or(c.TemplateName, "inertia")

	debug.Assert(c.ViteAddress != "", "vite address must be set")
	debug.Assert(c.TemplateName != "", "template name must be set")
}

// NewTemplate creates an html/template with Vite support from a template string.
//
// Available template functions and sub-templates:
//   - {{viteResource "path/to/file.js"}}: Include an asset (dev: proxied URL, prod: manifest-resolved)
//   - {{template "viteClient"}}: Vite development client (dev only, blank in production)
//   - {{template "viteReactRefresh"}}: React Fast Refresh support (dev only, blank in production)
//
// In development mode, assets are loaded from the Vite dev server at ViteAddress.
// In production mode (build tag: -tags=production), assets are resolved from the manifest.
func NewTemplate(content string, config *Config) (*template.Template, error) {
	if config == nil {
		//nolint:exhaustruct
		config = &Config{}
	}

	config.defaults()

	t := newTemplate(config)
	if _, err := t.Parse(content); err != nil {
		return nil, fmt.Errorf("inertia: failed to parse template: %w", err)
	}

	return t, nil
}

// MustTemplate is like NewTemplate but panics on error.
func MustTemplate(content string, c *Config) *template.Template {
	return must.Must(NewTemplate(content, c))
}

// FromFS creates an html/template with Vite support by loading templates from a file system.
// See NewTemplate for available template functions and behavior.
func FromFS(fsys fs.FS, path string, cfg *Config) (*template.Template, error) {
	if cfg == nil {
		//nolint:exhaustruct
		cfg = &Config{}
	}

	cfg.defaults()

	t := newTemplate(cfg)
	if _, err := t.ParseFS(fsys, path); err != nil {
		return nil, fmt.Errorf("inertia: failed to parse template: %w", err)
	}

	return t, nil
}
