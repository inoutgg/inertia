package inertia

import "cmp"

type Prop struct {
	mergeable bool
	deferred  bool
	lazy      bool // optional, deferred
	ignorable bool // false if always prop

	key   string
	group string // deferred
	val   any
	valFn func() any // optional, deferred
}

type DeferredOptions struct {
	Group string
	Merge bool
}

// NewDeferred creates a new deferred prop that is resolved only when
// it's requested.
//
// If the maybeGroup is not provided, it defaults to "default".
func NewDeferred(key string, fn func() any, opts *DeferredOptions) *Prop {
	p := &Prop{
		deferred:  true,
		lazy:      true,
		ignorable: true,
		key:       key,
		valFn:     fn,
		group:     "default",
	}

	if opts != nil {
		p.group = cmp.Or(opts.Group, "default")
		p.mergeable = opts.Merge
	}

	return p
}

// NewAlways create a new props that is always included in the response.
// It ignores the X-Inertia-Partial-Data and X-Inertia-Partial-Except headers.
func NewAlways(key string, value any) *Prop {
	return &Prop{
		ignorable: false,
		key:       key,
		val:       value,
	}
}

// NewOptional creates a new prop that is included in the response only if
// it's requested.
func NewOptional(key string, fn func() any) *Prop {
	return &Prop{
		ignorable: true,
		lazy:      true,
		key:       key,
		valFn:     fn,
	}
}

// PropOptions is the options for the prop.
type PropOptions struct {
	// Merge indicates whether the prop can be merged with other props.
	Merge bool
}

// NewProp creates a new regular prop.
func NewProp(key string, val any, opts *PropOptions) *Prop {
	p := &Prop{
		ignorable: true,
		key:       key,
		val:       val,
	}

	if opts != nil {
		p.mergeable = opts.Merge
	}

	return p
}

// value returns the prop value.
func (p *Prop) value() any {
	if p.valFn != nil {
		p.val = p.valFn()
		p.valFn = nil
	}

	return p.val
}
