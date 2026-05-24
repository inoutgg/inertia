package inertiaalways

import (
	"context"

	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
)

var _ inertiaprop.Prop = (*Prop)(nil)

// Prop is always included in responses, regardless of partial reload filters.
type Prop struct {
	valFn inertiaprop.Lazy
	val   any
	key   string
}

// New creates a prop that is always included in responses.
func New(key string, val any) *Prop {
	if lazy, ok := val.(inertiaprop.Lazy); ok {
		return &Prop{valFn: lazy, val: nil, key: key}
	}

	return &Prop{valFn: nil, val: val, key: key}
}

func (p *Prop) Key() string { return p.key }

func (p *Prop) Value(ctx context.Context) (any, error) {
	if p.valFn != nil {
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
func (p *Prop) Concurrent() bool                            { return false }
