package inertiadeferred

import (
	"cmp"
	"context"

	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
)

const DefaultGroup = "default"

var _ inertiaprop.Prop = (*Prop)(nil)

type Config struct {
	merge      *inertiaprop.MergeOpts
	once       *inertiaprop.OnceOpts
	group      string
	rescue     bool
	concurrent bool
}

type Option func(*Config)

func WithGroup(group string) Option { return func(config *Config) { config.group = group } }

func WithRescue(rescue bool) Option { return func(config *Config) { config.rescue = rescue } }

func WithConcurrent(config *Config) { config.concurrent = true }

func WithMerge(merge *inertiaprop.MergeOpts) Option {
	return func(config *Config) { config.merge = merge }
}

func WithOnce(once *inertiaprop.OnceOpts) Option {
	return func(config *Config) { config.once = once }
}

type Prop struct {
	val      inertiaprop.Lazy
	once     *inertiaprop.Onceable
	deferred *inertiaprop.Deferrable
	merge    *inertiaprop.Mergeable
	key      string

	concurrent bool
}

func New(key string, val inertiaprop.Lazy, opts ...Option) *Prop {
	config := &Config{merge: nil, once: nil, group: DefaultGroup, rescue: false, concurrent: false}
	for _, opt := range opts {
		opt(config)
	}

	return &Prop{
		val:        val,
		once:       config.once.Onceable(),
		deferred:   &inertiaprop.Deferrable{Group: cmp.Or(config.group, DefaultGroup), Rescue: config.rescue},
		merge:      config.merge.Mergeable(),
		key:        key,
		concurrent: config.concurrent,
	}
}

func (p *Prop) Key() string                            { return p.key }
func (p *Prop) Value(ctx context.Context) (any, error) { return p.val.Value(ctx) } //nolint:wrapcheck

func (p *Prop) IsFirstLoadIgnorable() bool                  { return true }
func (p *Prop) BypassPartialFilters() bool                  { return false }
func (p *Prop) Deferrable() (*inertiaprop.Deferrable, bool) { return p.deferred, p.deferred != nil }
func (p *Prop) Mergeable() (*inertiaprop.Mergeable, bool)   { return p.merge, p.merge != nil }
func (p *Prop) Scrollable() (*inertiaprop.Scrollable, bool) { return nil, false }
func (p *Prop) Onceable() (*inertiaprop.Onceable, bool)     { return p.once, p.once != nil }
func (p *Prop) Concurrent() bool                            { return p.concurrent }
