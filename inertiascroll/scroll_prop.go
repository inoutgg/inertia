package inertiascroll

import (
	"context"
	"strings"

	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
)

var _ inertiaprop.Prop = (*Prop)(nil)

type Page interface {
	~int | ~int64 | ~string
}

type Config struct {
	previousPage any
	nextPage     any
	currentPage  any
	pageName     string
	wrapper      string
}

type Option func(*Config)

func WithPagination[T Page](previousPage, nextPage, currentPage *T) Option {
	return func(config *Config) {
		config.previousPage = previousPage
		config.nextPage = nextPage
		config.currentPage = currentPage
	}
}

func WithWrapper(wrapper string) Option {
	return func(config *Config) {
		config.wrapper = wrapper
	}
}

func WithPageName(pageName string) Option {
	return func(config *Config) {
		config.pageName = pageName
	}
}

type Prop struct {
	lval   inertiaprop.Lazy
	val    any
	merge  *inertiaprop.Mergeable
	scroll *inertiaprop.Scrollable
	key    string
}

func New(key string, value any, opts ...Option) *Prop {
	var config Config
	for _, opt := range opts {
		opt(&config)
	}

	var prop Prop

	if lval, ok := value.(inertiaprop.Lazy); ok {
		prop.lval = lval
	} else {
		prop.val = value
	}

	prop.scroll = &inertiaprop.Scrollable{
		Path:         qualifyPropPath(key, "data"),
		PageName:     config.pageName,
		PreviousPage: config.previousPage,
		NextPage:     config.nextPage,
		CurrentPage:  config.currentPage,
	}
	if config.wrapper != "" {
		prop.scroll.Path = qualifyPropPath(key, config.wrapper)
	}

	prop.merge = &inertiaprop.Mergeable{Append: true}
	prop.key = key

	return &prop
}

func (p *Prop) Key() string { return p.key }

func (p *Prop) Value(ctx context.Context) (any, error) {
	if p.lval != nil {
		return p.lval.Value(ctx) //nolint:wrapcheck
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

func qualifyPropPath(propKey, path string) string {
	if path == "" || strings.HasPrefix(path, propKey+".") || path == propKey {
		return path
	}

	return propKey + "." + path
}
