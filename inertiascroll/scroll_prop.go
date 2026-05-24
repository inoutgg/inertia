package inertiascroll

import (
	"cmp"
	"context"
	"strings"

	"go.segfaultmedaddy.com/inertia"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
)

var _ inertia.Prop = (*Prop)(nil)

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
	valFn  inertia.Lazy
	val    any
	scroll *inertiaprop.Scrollable
	merge  *inertiaprop.Mergeable
	key    string
}

func New(key string, value any, opts ...Option) *Prop {
	config := &Config{previousPage: nil, nextPage: nil, currentPage: nil, pageName: "", wrapper: ""}
	for _, opt := range opts {
		opt(config)
	}

	val := value

	var valFn inertia.Lazy

	if lazy, ok := value.(inertia.Lazy); ok {
		val = nil
		valFn = lazy
	}

	prop := &Prop{
		valFn: valFn,
		val:   val,
		scroll: &inertiaprop.Scrollable{
			PreviousPage: nil,
			NextPage:     nil,
			CurrentPage:  nil,
			PageName:     "",
			Path:         key + ".data",
		},
		merge: &inertiaprop.Mergeable{
			AppendKeys:  nil,
			PrependKeys: nil,
			Append:      true,
			Prepend:     false,
		},
		key:   key,
	}

	if config.previousPage != nil || config.nextPage != nil || config.currentPage != nil || config.pageName != "" {
		prop.scroll.PageName = config.pageName
		prop.scroll.PreviousPage = config.previousPage
		prop.scroll.NextPage = config.nextPage
		prop.scroll.CurrentPage = config.currentPage
		prop.scroll.Path = qualifyPropPath(key, cmp.Or(config.wrapper, "data"))
	}

	return prop
}

func (p *Prop) Key() string { return p.key }

func (p *Prop) Value(ctx context.Context) (any, error) {
	if p.valFn != nil {
		return p.valFn.Value(ctx) //nolint:wrapcheck
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
