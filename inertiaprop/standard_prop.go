package inertiaprop

import (
	"context"

	"go.inout.gg/foundations/debug"

	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
)

var _ inertiaprop.Prop = (*Prop)(nil)

// Config holds the configuration for a Prop.
type Config struct {
	merge      inertiaprop.Merge
	once       *inertiaprop.OnceOpts
	concurrent bool
}

// Option is a function that configures a Prop.
type Option func(*Config)

// WithOnce sets the OnceOpts for the property.
//
// If OnceOpts is not nil, the property will be treated as onceable.
func WithOnce(once *inertiaprop.OnceOpts) Option {
	return func(config *Config) { config.once = once }
}

// WithMerge sets the MergeOpts for the property.
//
// If MergeOpts is not nil, the property will be treated as mergeable.
func WithMerge(merge *inertiaprop.MergeOpts) Option {
	return func(config *Config) { config.merge = merge }
}

// WithConcurrent enables concurrent resolution for the prop.
//
// Non-lazy prop is still resolved sequencially.
func WithConcurrent(config *Config) { config.concurrent = true }

// Prop is a standard prop: included on every full-page response and, during
// partial reloads, only when its key matches the client's only/except filter.
// Optional merge, once, and concurrent behavior is configured via the inertiaprop
// options passed to New or NewLazy.
type Prop struct {
	val        any
	valFn      inertiaprop.Lazy
	once       *inertiaprop.Onceable
	merge      *inertiaprop.Mergeable
	key        string
	concurrent bool
}

// New creates a standard prop.
//
// It is included on standard visits and optionally during partial reloads when requested via only/except.
// Options such as WithMerge and WithOnce customize the prop's behavior.
func New(key string, val any, opts ...Option) *Prop {
	var cfg Config
	for _, opt := range opts {
		opt(&cfg)
	}

	//nolint:exhaustruct
	return &Prop{
		key:   key,
		val:   val,
		once:  inertiaprop.ToOnceable(cfg.once),
		merge: inertiaprop.ToMergeable(cfg.merge),
	}
}

// NewLazy is like New but accepts a lazy value.
func NewLazy(key string, valFn inertiaprop.Lazy, opts ...Option) *Prop {
	var cfg Config
	for _, opt := range opts {
		opt(&cfg)
	}

	//nolint:exhaustruct
	return &Prop{
		key:        key,
		valFn:      valFn,
		once:       inertiaprop.ToOnceable(cfg.once),
		merge:      inertiaprop.ToMergeable(cfg.merge),
		concurrent: cfg.concurrent,
	}
}

func (p *Prop) Key() string { return p.key }

func (p *Prop) Value(ctx context.Context) (any, error) {
	if p.valFn != nil {
		debug.Assert(p.valFn != nil, "p.valFn must not be nil")

		return p.valFn.Value(ctx) //nolint:wrapcheck
	}

	return p.val, nil
}

func (p *Prop) IsFirstLoadIgnorable() bool                  { return false }
func (p *Prop) BypassPartialFilters() bool                  { return false }
func (p *Prop) Deferrable() (*inertiaprop.Deferrable, bool) { return nil, false }
func (p *Prop) Mergeable() (*inertiaprop.Mergeable, bool)   { return p.merge, p.merge != nil }
func (p *Prop) Scrollable() (*inertiaprop.Scrollable, bool) { return nil, false }
func (p *Prop) Onceable() (*inertiaprop.Onceable, bool)     { return p.once, p.once != nil }
func (p *Prop) Concurrent() bool                            { return p.concurrent }
