package inertia

import (
	"go.segfaultmedaddy.com/inertia/internal/inertiahttp"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
)

type (
	Prop = inertiaprop.Prop

	// Lazy represents a prop value that is resolved on-demand rather than eagerly.
	Lazy = inertiaprop.Lazy

	// LazyFunc is a function adapter that implements the Lazy interface.
	//
	// It allows using ordinary functions as lazy prop values.
	// The returned value must be JSON-serializable.
	LazyFunc = inertiaprop.LazyFunc

	// Proper represents a collection of props that can be attached to a render context.
	Proper = inertiahttp.Proper

	// Props is a slice of Prop that satisfies the Proper interface, suitable for
	// use with WithProps and WithSharedProps.
	Props = inertiahttp.Props
)
