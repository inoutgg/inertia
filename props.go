package inertia

import (
	"cmp"
	"context"
)

var (
	_ Proper = (Props)(nil)
	_ Proper = (*Prop)(nil)
)

const DefaultDeferredGroup = "default"

// Prop represents a single property passed to an Inertia page component.
// Props control data visibility, lazy loading, merging behavior, and resolution timing.
//
// To create a prop use constructor functions:
//   - NewProp: Standard prop, included on initial render
//   - NewAlways: Always included, ignores partial reload filters
//   - NewOptional: Lazy-loaded, resolved when explicitly requested by a client
//   - NewDeferred: Lazy-loaded, requested by a client after initial render
//
// Attach props to a page using WithProps option.
type Prop struct {
	scrollMeta scrollMetadata
	valFn      Lazy
	val        any
	expiresAt  *int64
	onceKey    string
	key        string
	group      string
	scrollPath string
	matchOn    []string
	once       bool
	deepMerge  bool
	fresh      bool
	scroll     bool
	prepend    bool
	mergeable  bool
	deferred   bool
	lazy       bool
	ignorable  bool
	concurrent bool
}

// DeferredOptions configures the behavior of deferred props.
type DeferredOptions struct {
	Once       *OnceOptions
	Group      string
	MatchOn    []string
	Merge      bool
	Prepend    bool
	DeepMerge  bool
	Concurrent bool
}

type (
	// Lazy represents a prop value that is resolved on-demand rather than eagerly.
	Lazy interface {
		// Value resolves and returns the prop's value.
		//
		// The returned value must be JSON-serializable.
		Value(context.Context) (any, error)
	}

	// LazyFunc is a function adapter that implements the Lazy interface.
	//
	// It allows using ordinary functions as lazy prop values.
	// The returned value must be JSON-serializable.
	LazyFunc func(context.Context) (any, error)
)

// Value calls `fn()`.
func (fn LazyFunc) Value(ctx context.Context) (any, error) { return fn(ctx) }

// NewDeferred creates a deferred prop that is lazy-loaded by the client after initial render.
// Deferred props reduce initial page load time by deferring expensive computations.
//
// If opts is nil, default options are used (default group, no merging, sequential resolution).
func NewDeferred(key string, fn Lazy, opts *DeferredOptions) Prop {
	//nolint:exhaustruct
	prop := Prop{
		deferred:   true, // important
		lazy:       true, // important
		ignorable:  true, // important
		key:        key,
		valFn:      fn,
		group:      DefaultDeferredGroup,
		concurrent: false,
	}

	if opts != nil {
		prop.group = cmp.Or(opts.Group, DefaultDeferredGroup)
		prop.mergeable = opts.Merge
		prop.prepend = opts.Prepend
		prop.deepMerge = opts.DeepMerge
		prop.matchOn = opts.MatchOn
		prop = applyOnceOptions(prop, key, opts.Once)
		prop.concurrent = opts.Concurrent
	}

	return prop
}

// NewAlways creates a prop that is always included in responses.
// Unlike regular props, it ignores partial reload filters (X-Inertia-Partial-Data/Except headers).
//
// It is particularly useful to enforce load of critical data that must always be present,
// such as authentication state or global config.
func NewAlways(key string, value any) Prop {
	//nolint:exhaustruct
	return Prop{
		ignorable: false, // important
		key:       key,
		val:       value,
	}
}

// NewOptional creates a lazily-evaluated prop included only during partial reloads when explicitly requested.
// Useful for expensive computations that aren't needed on every render.
//
// The value function is only called when the client specifically requests this prop.
func NewOptional(key string, fn Lazy) Prop {
	//nolint:exhaustruct
	return Prop{
		ignorable: true, // important
		lazy:      true, // important
		key:       key,
		valFn:     fn,
	}
}

// OnceOptions configures once prop behavior.
type OnceOptions struct {
	ExpiresAt *int64
	Key       string
	Fresh     bool
}

// NewOnce creates a prop remembered by the client and skipped on subsequent visits.
func NewOnce(key string, fn Lazy, opts *OnceOptions) Prop {
	//nolint:exhaustruct
	prop := Prop{
		ignorable: true, // important
		key:       key,
		valFn:     fn,
	}

	if opts == nil {
		//nolint:exhaustruct
		opts = &OnceOptions{}
	}

	return applyOnceOptions(prop, key, opts)
}

// ScrollPage is a page number or cursor supported by Inertia infinite scroll metadata.
type ScrollPage interface {
	~int | ~int64 | ~string
}

// ScrollMetadata configures pagination metadata for infinite scroll props.
type ScrollMetadata[T ScrollPage] struct {
	PreviousPage *T
	NextPage     *T
	CurrentPage  *T
	PageName     string
}

type scrollMetadata struct {
	PreviousPage any
	NextPage     any
	CurrentPage  any
	PageName     string
}

// ScrollOptions configures infinite scroll prop behavior.
type ScrollOptions struct {
	Metadata scrollMetadata
	Wrapper  string
}

// NewScrollOptions creates type-safe infinite scroll options.
func NewScrollOptions[T ScrollPage](wrapper string, metadata ScrollMetadata[T]) *ScrollOptions {
	return &ScrollOptions{
		Wrapper: wrapper,
		Metadata: scrollMetadata{
			PageName:     metadata.PageName,
			PreviousPage: metadata.PreviousPage,
			NextPage:     metadata.NextPage,
			CurrentPage:  metadata.CurrentPage,
		},
	}
}

// NewScroll creates an infinite scroll prop with v3 scroll metadata.
func NewScroll(key string, value any, opts *ScrollOptions) Prop {
	prop := NewProp(key, value, &PropOptions{
		Once:      nil,
		MatchOn:   nil,
		Merge:     true,
		Prepend:   false,
		DeepMerge: false,
	})
	if lazy, ok := value.(Lazy); ok {
		prop.val = nil
		prop.valFn = lazy
	}

	prop.scroll = true
	prop.scrollPath = key + ".data"

	if opts != nil {
		prop.scrollPath = qualifyPropPath(key, cmp.Or(opts.Wrapper, "data"))
		prop.scrollMeta = opts.Metadata
	}

	return prop
}

// PropOptions configures standard prop behavior.
type PropOptions struct {
	Once      *OnceOptions
	MatchOn   []string
	Merge     bool
	Prepend   bool
	DeepMerge bool
}

// NewProp creates a standard prop included on initial page load and partial reloads.
//
// If opts is nil, default options are used (no merging).
func NewProp(key string, val any, opts *PropOptions) Prop {
	//nolint:exhaustruct
	prop := Prop{
		ignorable: true, // important
		key:       key,
		val:       val,
	}

	if opts != nil {
		prop.mergeable = opts.Merge
		prop.prepend = opts.Prepend
		prop.deepMerge = opts.DeepMerge
		prop.matchOn = opts.MatchOn
		prop = applyOnceOptions(prop, key, opts.Once)
	}

	return prop
}

func applyOnceOptions(prop Prop, defaultKey string, opts *OnceOptions) Prop {
	if opts == nil {
		return prop
	}

	prop.once = true
	prop.onceKey = cmp.Or(opts.Key, defaultKey)
	prop.expiresAt = opts.ExpiresAt
	prop.fresh = opts.Fresh

	return prop
}

func (p Prop) Props() []Prop { return []Prop{p} }
func (p Prop) Len() int      { return 1 }

// value returns the prop value.
func (p Prop) value(ctx context.Context) (any, error) {
	if p.valFn != nil {
		v, err := p.valFn.Value(ctx)
		if err != nil {
			return nil, err //nolint:wrapcheck
		}

		return v, nil
	}

	return p.val, nil
}

// Proper represents a collection of props that can be attached to a render context.
type Proper interface {
	// Props returns the underlying prop slice.
	Props() []Prop

	// Len returns the number of props in the collection.
	Len() int
}

// Props is a collection of props.
type Props []Prop

func (p Props) Len() int      { return len(p) }
func (p Props) Props() []Prop { return p }
