package inertiascroll

import (
	"context"

	"go.inout.gg/foundations/debug"

	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprotocol"
)

var _ inertiaprop.Prop = (*Prop)(nil)

// Page is a constraint for valid pagination page types.
type Page interface {
	~int | ~int64 | ~string
}

// Config holds the configuration for a scrollable Prop.
type Config struct {
	previousPage any
	nextPage     any
	currentPage  any
	pageName     string
	wrapper      string
}

// Option is a function that configures a Prop.
type Option func(*Config)

// WithPagination sets the previous, next, and current page values for pagination.
func WithPagination[T Page](previousPage, nextPage, currentPage *T) Option {
	return func(config *Config) {
		config.previousPage = previousPage
		config.nextPage = nextPage
		config.currentPage = currentPage
	}
}

// WithWrapper sets a custom wrapper path for the scrollable data.
func WithWrapper(wrapper string) Option {
	return func(config *Config) { config.wrapper = wrapper }
}

// WithPageName sets the query parameter name used for pagination.
func WithPageName(pageName string) Option {
	return func(config *Config) { config.pageName = pageName }
}

// Prop is a scrollable prop for infinite scroll pagination.
//
// It merges new data with existing data instead of replacing it on partial reloads.
type Prop struct {
	val    any
	merge  *inertiaprop.Mergeable
	scroll *inertiaprop.Scrollable
	key    string
}

// New creates a scrollable prop for infinite scroll pagination.
//
// Options such as WithPagination and WithPageName customize the prop's behavior.
func New(key string, value any, opts ...Option) *Prop {
	var config Config
	for _, opt := range opts {
		opt(&config)
	}

	prop := &Prop{
		val: value,
		scroll: &inertiaprop.Scrollable{
			Path:         inertiaprotocol.QualifyPath(key, "data"),
			PageName:     config.pageName,
			PreviousPage: config.previousPage,
			NextPage:     config.nextPage,
			CurrentPage:  config.currentPage,
		},
		key: key,
		merge: &inertiaprop.Mergeable{
			Append:      true,
			Prepend:     false,
			AppendKeys:  nil,
			PrependKeys: nil,
		},
	}

	if config.wrapper != "" {
		prop.scroll.Path = inertiaprotocol.QualifyPath(key, config.wrapper)
	}

	return prop
}

func (p *Prop) Key() string { return p.key }

func (p *Prop) Value(ctx context.Context) (any, error) {
	if lazy, ok := p.val.(inertiaprop.Lazy); ok {
		debug.Assert(lazy != nil, "p.val must not be nil")
		return lazy.Value(ctx) //nolint:wrapcheck
	}

	return p.val, nil
}

func (p *Prop) IsFirstLoadIgnorable() bool                  { return false }
func (p *Prop) BypassPartialFilters() bool                  { return false }
func (p *Prop) Deferrable() (*inertiaprop.Deferrable, bool) { return nil, false }
func (p *Prop) Mergeable() (*inertiaprop.Mergeable, bool)   { return p.merge, p.merge != nil }
func (p *Prop) Scrollable() (*inertiaprop.Scrollable, bool) { return p.scroll, p.scroll != nil }
func (p *Prop) Onceable() (*inertiaprop.Onceable, bool)     { return nil, false }
func (p *Prop) Concurrent() bool                            { return false }
