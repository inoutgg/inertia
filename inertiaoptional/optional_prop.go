package inertiaoptional

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
func WithConcurrent(config *Config) { config.concurrent = true }

// Prop is a lazily-evaluated prop included only during partial reloads when explicitly requested.
type Prop struct {
	val        inertiaprop.Lazy
	key        string
	concurrent bool
}

// New creates a lazily-evaluated prop included only during partial reloads when explicitly requested.
//
// It is useful for expensive computations that aren't needed on every render.
// The value function is only called when the client specifically requests this prop.
// Options customize the prop's behavior.
func New(key string, val inertiaprop.Lazy, opts ...Option) *Prop {
	debug.Assert(key != "", "key must be non-empty")
	debug.Assert(val != nil, "p.val must not be nil")

	var cfg Config
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Prop{
		val:        val,
		key:        key,
		concurrent: cfg.concurrent,
	}
}

func (p *Prop) Key() string                            { return p.key }
func (p *Prop) Value(ctx context.Context) (any, error) { return p.val.Value(ctx) } //nolint:wrapcheck

func (p *Prop) IsFirstLoadIgnorable() bool                  { return true }
func (p *Prop) BypassPartialFilters() bool                  { return false }
func (p *Prop) Deferrable() (*inertiaprop.Deferrable, bool) { return nil, false }
func (p *Prop) Mergeable() (*inertiaprop.Mergeable, bool)   { return nil, false }
func (p *Prop) Scrollable() (*inertiaprop.Scrollable, bool) { return nil, false }
func (p *Prop) Onceable() (*inertiaprop.Onceable, bool)     { return nil, false }
func (p *Prop) Concurrent() bool                            { return p.concurrent }
