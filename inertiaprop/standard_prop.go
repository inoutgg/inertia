package inertiaprop

import (
	"context"

	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
)

var _ inertiaprop.Prop = (*Prop)(nil)

// Config holds the configuration for a Prop.
//
// Use Options to configure the behavior of the Prop.
type Config struct {
	merge *inertiaprop.MergeOpts
	once  *inertiaprop.OnceOpts
}

// Option is a function that configures a Prop.
//
// Use specialized options to configure the behavior of the Prop such as:
// WithOnce, WithMerge, etc.
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

type Prop struct {
	val   any
	once  *inertiaprop.Onceable
	merge *inertiaprop.Mergeable
	key   string
}

func New(key string, val any, opts ...Option) *Prop {
	var cfg Config
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Prop{
		val:   val,
		once:  cfg.once.Onceable(),
		merge: cfg.merge.Mergeable(),
		key:   key,
	}
}

func (p *Prop) Key() string                        { return p.key }
func (p *Prop) Value(context.Context) (any, error) { return p.val, nil }

func (p *Prop) IsFirstLoadIgnorable() bool                  { return false }
func (p *Prop) BypassPartialFilters() bool                  { return false }
func (p *Prop) Deferrable() (*inertiaprop.Deferrable, bool) { return nil, false }
func (p *Prop) Mergeable() (*inertiaprop.Mergeable, bool)   { return p.merge, p.merge != nil }
func (p *Prop) Scrollable() (*inertiaprop.Scrollable, bool) { return nil, false }
func (p *Prop) Onceable() (*inertiaprop.Onceable, bool)     { return p.once, p.once != nil }
func (p *Prop) Concurrent() bool                            { return false }
