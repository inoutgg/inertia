package inertia

import (
	"cmp"
)

var (
	_ Proper = (Props)(nil)
	_ Proper = (*Prop)(nil)
)

const DefaultDeferredGroup = "default"

// Prop represents a single page property.
//
// Use convinient intstanciation functions to create a new property
// such as NewProp, NewDeferred, NewAlways and NewOptional.
//
// Props can be attached to a rendering context using WithProps helper.
type Prop struct {
	val       any
	valFn     func() any // optional, deferred
	key       string
	group     string // deferred
	mergeable bool
	deferred  bool
	lazy      bool // optional, deferred
	ignorable bool // false if always prop
}

// DeferredOptions represents a.
type DeferredOptions struct {
	// Group defines deferred prop resolution group.
	//
	// If Group is not provided, it defaults to DefaultDeferredGroup.
	Group string

	// Merge defines props update resolution. If it is false prop is
	// overridden, otherwise merged.
	//
	// Default to false.
	Merge bool
}

// NewDeferred creates a new deferred prop that is resolved only when
// it's requested.
//
// If opts is nil, default options is used.
func NewDeferred(key string, fn func() any, opts *DeferredOptions) *Prop {
	//nolint:exhaustruct
	prop := &Prop{
		deferred:  true, // important
		lazy:      true, // important
		ignorable: true, // important
		key:       key,
		valFn:     fn,
		group:     DefaultDeferredGroup,
	}

	if opts != nil {
		prop.group = cmp.Or(opts.Group, DefaultDeferredGroup)
		prop.mergeable = opts.Merge
	}

	return prop
}

// NewAlways create a new props that is always included in the response.
// It ignores the X-Inertia-Partial-Data and X-Inertia-Partial-Except headers.
func NewAlways(key string, value any) *Prop {
	//nolint:exhaustruct
	return &Prop{
		ignorable: false, // important
		key:       key,
		val:       value,
	}
}

// NewOptional creates a new prop that is included in the response only if
// it's requested.
func NewOptional(key string, fn func() any) *Prop {
	//nolint:exhaustruct
	return &Prop{
		ignorable: true, // important
		lazy:      true, // important
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
// opts can be nil.
func NewProp(key string, val any, opts *PropOptions) *Prop {
	//nolint:exhaustruct
	prop := &Prop{
		ignorable: true, // important
		key:       key,
		val:       val,
	}

	if opts != nil {
		prop.mergeable = opts.Merge
	}

	return prop
}

// value returns the prop value.
func (p *Prop) value() any {
	if p.valFn != nil {
		p.val = p.valFn()
		p.valFn = nil
	}

	return p.val
}

func (p *Prop) Props() []*Prop { return []*Prop{p} }
func (p *Prop) Len() int       { return 1 }

// Proper is an interface that represents a collection of props.
// It is used to attach props to the rendering context.
type Proper interface {
	// Props returns the list of props.
	Props() []*Prop

	// Len returns the number of props.
	Len() int
}

// Props is a collection of props.
type Props []*Prop

func (p Props) Len() int       { return len(p) }
func (p Props) Props() []*Prop { return p }
