package inertia

import "go.segfaultmedaddy.com/inertia/internal/inertiaprop"

var _ Proper = (Props)(nil)

// Prop represents a single property passed to an Inertia page component.
// Props control data visibility, lazy loading, merging behavior, and resolution timing.
//
// To create a prop, use the specialized prop packages:
// inertiaprop, inertiaalways, inertiaoptional, inertiadeferred, and inertiascroll.
//
// Attach props to a page using WithProps option.
type (
	Prop = inertiaprop.Prop

	// Lazy represents a prop value that is resolved on-demand rather than eagerly.
	Lazy = inertiaprop.Lazy

	// LazyFunc is a function adapter that implements the Lazy interface.
	//
	// It allows using ordinary functions as lazy prop values.
	// The returned value must be JSON-serializable.
	LazyFunc = inertiaprop.LazyFunc

	// MergeKey identifies a prop key and an optional secondary match key used by
	// the client when merging server data into existing client-side state.
	MergeKey = inertiaprop.MergeKey

	// MergeOpts configures the append/prepend direction and per-key merge
	// behavior for a prop built with inertiaprop.WithMerge.
	MergeOpts = inertiaprop.MergeOpts

	// OnceOpts configures the once-fetch behavior for a prop built with
	// inertiaprop.WithOnce; chain Key, ExpiresAt, and Fresh on the returned
	// value to set the cache key, expiration timestamp, and force-refresh flag.
	OnceOpts = inertiaprop.OnceOpts
)

// NewMergeOpts returns a MergeOpts with append behavior enabled by default;
// pass the result to inertiaprop.WithMerge to opt a prop into client-side
// merging.
func NewMergeOpts() *MergeOpts { return inertiaprop.NewMergeOpts() }

// NewOnceOpts returns a OnceOpts; configure it via the chainable Key,
// ExpiresAt, and Fresh methods before passing it to inertiaprop.WithOnce.
func NewOnceOpts() *OnceOpts { return inertiaprop.NewOnceOpts() }

// Proper represents a collection of props that can be attached to a render context.
type Proper interface {
	// Props returns the underlying prop slice.
	Props() []Prop

	// Len returns the number of props in the collection.
	Len() int
}

// Props is a slice of Prop that satisfies the Proper interface, suitable for
// use with WithProps and WithSharedProps.
type Props []Prop

func (p Props) Len() int      { return len(p) }
func (p Props) Props() []Prop { return p }
