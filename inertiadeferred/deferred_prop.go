package inertiadeferred

import (
	"cmp"
	"context"

	"go.inout.gg/foundations/debug"

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

// WithGroup sets the deferred group name.
// Props in the same group are fetched together in a single parallel request.
func WithGroup(group string) Option { return func(config *Config) { config.group = group } }

// WithRescue enables or disables rescue mode.
// When rescue is true, resolution errors are caught and the prop is omitted from the response.
func WithRescue(rescue bool) Option { return func(config *Config) { config.rescue = rescue } }

// WithConcurrent enables concurrent resolution for the prop.
func WithConcurrent(config *Config) { config.concurrent = true }

// WithMerge sets the merge options for the prop.
func WithMerge(merge *inertiaprop.MergeOpts) Option {
	return func(config *Config) { config.merge = merge }
}

// WithOnce sets the once options for the prop.
func WithOnce(once *inertiaprop.OnceOpts) Option {
	return func(config *Config) { config.once = once }
}

// Prop is a lazily-evaluated prop loaded after the initial page render.
//
// Deferred props improve perceived performance by allowing the initial render to happen quickly.
// The value is resolved in a separate request after the page is first displayed.
type Prop struct {
	val      inertiaprop.Lazy
	once     *inertiaprop.Onceable
	deferred *inertiaprop.Deferrable
	merge    *inertiaprop.Mergeable
	key      string

	concurrent bool
}

// New creates a lazily-evaluated prop loaded after the initial page render.
//
// Deferred props are fetched separately after the initial page loads, improving perceived performance.
// Options such as WithGroup and WithRescue customize the prop's behavior.
func New(key string, val inertiaprop.Lazy, opts ...Option) *Prop {
	config := Config{group: DefaultGroup} //nolint:exhaustruct
	for _, opt := range opts {
		opt(&config)
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

func (p *Prop) Key() string { return p.key }

func (p *Prop) Value(ctx context.Context) (any, error) {
	val, err := p.val.Value(ctx)
	if err != nil && p.deferred.Rescue {
		return nil, &inertiaprop.RescueError{Key: p.key, Err: err}
	}

	return val, err //nolint:wrapcheck
}

func (p *Prop) IsFirstLoadIgnorable() bool { return true }
func (p *Prop) BypassPartialFilters() bool { return false }
func (p *Prop) Deferrable() (*inertiaprop.Deferrable, bool) {
	debug.Assert(p.deferred != nil, "p.deferred must not be nil")
	return p.deferred, true
}
func (p *Prop) Mergeable() (*inertiaprop.Mergeable, bool)   { return p.merge, p.merge != nil }
func (p *Prop) Scrollable() (*inertiaprop.Scrollable, bool) { return nil, false }
func (p *Prop) Onceable() (*inertiaprop.Onceable, bool)     { return p.once, p.once != nil }
func (p *Prop) Concurrent() bool                            { return p.concurrent }
