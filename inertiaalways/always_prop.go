package inertiaalways

import (
	"context"

	"go.inout.gg/foundations/debug"

	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
)

var _ inertiaprop.Prop = (*Prop)(nil)

// Config holds the configuration for a Prop.
type Config struct {
	concurrent bool
}

// Option is a function that configures a Prop.
type Option func(*Config)

// WithConcurrent enables concurrent resolution for the prop.
//
// Non-lazy prop is still resolved sequencially.
func WithConcurrent(config *Config) { config.concurrent = true }

// Prop is always included in responses, regardless of partial reload filters.
type Prop struct {
	val        any
	valFn      inertiaprop.Lazy
	key        string
	concurrent bool
}

// New creates a prop that is always included in responses.
//
// It bypasses partial reload filters, ensuring the prop is present even when
// the client requests only specific props via only/except.
func New(key string, val any) *Prop {
	return &Prop{val: val, key: key} //nolint:exhaustruct
}

// NewLazy is like New but accepts a lazy value.
func NewLazy(key string, valFn inertiaprop.Lazy, opts ...Option) *Prop {
	debug.Assert(valFn != nil, "valFn must not be nil")

	var cfg Config
	for _, opt := range opts {
		opt(&cfg)
	}

	//nolint:exhaustruct
	return &Prop{
		key:        key,
		valFn:      valFn,
		concurrent: cfg.concurrent,
	}
}

func (p *Prop) Key() string { return p.key }

func (p *Prop) Value(ctx context.Context) (any, error) {
	if p.valFn != nil {
		debug.Assert(p.valFn != nil, "valFn must not be nil")

		return p.valFn.Value(ctx) //nolint:wrapcheck
	}

	return p.val, nil
}

func (p *Prop) IsFirstLoadIgnorable() bool                  { return false }
func (p *Prop) BypassPartialFilters() bool                  { return true }
func (p *Prop) Deferrable() (*inertiaprop.Deferrable, bool) { return nil, false }
func (p *Prop) Mergeable() (*inertiaprop.Mergeable, bool)   { return nil, false }
func (p *Prop) Scrollable() (*inertiaprop.Scrollable, bool) { return nil, false }
func (p *Prop) Onceable() (*inertiaprop.Onceable, bool)     { return nil, false }
func (p *Prop) Concurrent() bool                            { return p.concurrent }
