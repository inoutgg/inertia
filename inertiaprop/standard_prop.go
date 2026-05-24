package inertiaprop

import (
	"context"

	"go.segfaultmedaddy.com/inertia"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
)

var _ inertia.Prop = (*Prop)(nil)

type Config struct {
	merge *inertiaprop.MergeOpts
	once  *inertiaprop.OnceOpts
}

type Option func(*Config)

func WithOnce(once *inertiaprop.OnceOpts) Option {
	return func(config *Config) { config.once = once }
}

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
