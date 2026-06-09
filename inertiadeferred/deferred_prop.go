package inertiadeferred

import (
	"cmp"
	"context"

	"go.inout.gg/foundations/debug"

	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
)

// DefaultGroup is the deferred-fetch group assigned to a Prop when WithGroup
// is not used; the client groups all props sharing the same group into a
// single follow-up request.
const DefaultGroup = "default"

var _ inertiaprop.Prop = (*Prop)(nil)

// Config holds the configuration for a Prop.
type Config struct {
	merge      inertiaprop.Merge
	once       *inertiaprop.OnceOpts
	group      string
	rescue     bool
	concurrent bool
}

// Option is a function that configures a Prop.
type Option func(*Config)

// WithGroup sets the deferred group name.
// Props in the same group are fetched together in a single parallel request.
func WithGroup(group string) Option { return func(config *Config) { config.group = group } }

// WithRescue enables or disables rescue mode.
// When rescue is true, resolution errors are caught and the prop is omitted from the response.
func WithRescue(rescue bool) Option { return func(config *Config) { config.rescue = rescue } }

// WithConcurrent enables concurrent resolution for the prop.
func WithConcurrent(config *Config) { config.concurrent = true }

// WithMerge sets the merge configuration for the prop.
//
// Accepts any type that implements Merge, such as AppendRootMergeOpts,
// PrependRootMergeOpts, PathMergeOpts, or DeepMergeOpts.
func WithMerge(merge inertiaprop.Merge) Option {
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
	debug.Assert(val != nil, "p.val must not be nil")

	cfg := Config{group: DefaultGroup} //nolint:exhaustruct
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Prop{
		key:        key,
		val:        val,
		once:       inertiaprop.ToOnceable(cfg.once),
		merge:      inertiaprop.ToMergeable(cfg.merge),
		deferred:   &inertiaprop.Deferrable{Group: cmp.Or(cfg.group, DefaultGroup), Rescue: cfg.rescue},
		concurrent: cfg.concurrent,
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
