package inertia

import (
	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
)

type (
	// Prop represents a single property passed to an Inertia page component.
	//
	// To create a prop, use the specialized prop packages:
	// inertiaprop, inertiaalways, inertiaoptional, inertiadeferred, and inertiascroll.
	//
	// Attach props to a page using WithProps option.
	Prop = inertiaprop.Prop

	// Lazy represents a prop value that is resolved on-demand rather than eagerly.
	Lazy = inertiaprop.Lazy

	// LazyFunc is a function adapter that implements the Lazy interface.
	//
	// It allows using ordinary functions as lazy prop values.
	// The returned value must be JSON-serializable.
	LazyFunc = inertiaprop.LazyFunc

	// Proper represents a collection of props that can be attached to a render context.
	Proper = inertiaprop.Proper

	// Props is a slice of Prop that satisfies the Proper interface, suitable for
	// use with WithProps and WithSharedProps.
	Props = inertiaprop.Props
)
