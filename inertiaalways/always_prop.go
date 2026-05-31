package inertiaalways

import (
	"context"

	"go.inout.gg/foundations/debug"

	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
)

var _ inertiaprop.Prop = (*Prop)(nil)

// Prop is always included in responses, regardless of partial reload filters.
type Prop struct {
	val any
	key string
}

// New creates a prop that is always included in responses.
//
// It bypasses partial reload filters, ensuring the prop is present even when
// the client requests only specific props via only/except.
func New(key string, val any) *Prop {
	return &Prop{val: val, key: key}
}

func (p *Prop) Key() string { return p.key }

func (p *Prop) Value(ctx context.Context) (any, error) {
	if lazy, ok := p.val.(inertiaprop.Lazy); ok {
		debug.Assert(lazy != nil, "p.val must not be nil")

		return lazy.Value(ctx) //nolint:wrapcheck
	}

	return p.val, nil
}

func (p *Prop) IsFirstLoadIgnorable() bool                  { return false }
func (p *Prop) BypassPartialFilters() bool                  { return true }
func (p *Prop) Deferrable() (*inertiaprop.Deferrable, bool) { return nil, false }
func (p *Prop) Mergeable() (*inertiaprop.Mergeable, bool)   { return nil, false }
func (p *Prop) Scrollable() (*inertiaprop.Scrollable, bool) { return nil, false }
func (p *Prop) Onceable() (*inertiaprop.Onceable, bool)     { return nil, false }
func (p *Prop) Concurrent() bool                            { return false }
