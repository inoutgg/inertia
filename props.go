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
	val        any
	valFn      Lazy // optional, deferred
	key        string
	group      string // deferred
	mergeable  bool
	prepend    bool
	deepMerge  bool
	matchOn    []string
	once       bool
	onceKey    string
	expiresAt  *int64
	fresh      bool
	scroll     bool
	scrollPath string
	scrollMeta scrollMetadata
	deferred   bool
	lazy       bool // optional, deferred
	ignorable  bool // false if always prop
	concurrent bool // deferred
}

// DeferredOptions configures the behavior of deferred props.
type DeferredOptions struct {
	// Group assigns this prop to a named deferred group.
	//
	// Props in the same group are resolved together when requested by the client.
	// Defaults to DefaultDeferredGroup if not specified.
	Group string

	// Merge determines how updates are handled on partial reloads.
	//
	// If true, the prop value is merged with the existing client-side value.
	// If false, the value is replaced entirely. Defaults to false.
	Merge bool

	// Prepend marks the prop for prepend merging instead of append merging.
	Prepend bool

	// DeepMerge marks the prop for deep merging.
	DeepMerge bool

	// MatchOn configures prop-relative paths used to match items while merging.
	MatchOn []string

	// Once configures the prop to be remembered and reused by the client.
	Once *OnceOptions

	// Concurrent enables parallel resolution for this prop.
	//
	// When true, this prop can be resolved concurrently with other concurrent props
	// within the same request, up to the configured concurrency limit.
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
		prop.applyOnceOptions(key, opts.Once)
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
	// Key is the client-side remembered key. Defaults to the prop key.
	Key string

	// ExpiresAt is emitted as a Unix millisecond expiration timestamp.
	ExpiresAt *int64

	// Fresh forces the prop to resolve even if the client has already loaded it.
	Fresh bool
}

// NewOnce creates a prop remembered by the client and skipped on subsequent visits.
func NewOnce(key string, fn Lazy, opts *OnceOptions) Prop {
	prop := Prop{
		ignorable: true, // important
		key:       key,
		valFn:     fn,
	}

	if opts == nil {
		opts = &OnceOptions{}
	}

	prop.applyOnceOptions(key, opts)

	return prop
}

// ScrollPage is a page number or cursor supported by Inertia infinite scroll metadata.
type ScrollPage interface {
	~int | ~int64 | ~string
}

// ScrollMetadata configures pagination metadata for infinite scroll props.
type ScrollMetadata[T ScrollPage] struct {
	PageName     string
	PreviousPage *T
	NextPage     *T
	CurrentPage  *T
}

type scrollMetadata struct {
	PageName     string
	PreviousPage any
	NextPage     any
	CurrentPage  any
}

// ScrollOptions configures infinite scroll prop behavior.
type ScrollOptions struct {
	// Wrapper is the nested data path to merge. Defaults to "data".
	Wrapper string

	// Metadata is emitted as scrollProps for the client component.
	Metadata scrollMetadata
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
	prop := NewProp(key, value, &PropOptions{Merge: true})
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
	// Merge determines whether this prop's value is merged or replaced during partial reloads.
	Merge bool

	// Prepend marks the prop for prepend merging instead of append merging.
	Prepend bool

	// DeepMerge marks the prop for deep merging.
	DeepMerge bool

	// MatchOn configures prop-relative paths used to match items while merging.
	MatchOn []string

	// Once configures the prop to be remembered and reused by the client.
	Once *OnceOptions
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
		prop.applyOnceOptions(key, opts.Once)
	}

	return prop
}

func (p *Prop) applyOnceOptions(defaultKey string, opts *OnceOptions) {
	if opts == nil {
		return
	}

	p.once = true
	p.onceKey = cmp.Or(opts.Key, defaultKey)
	p.expiresAt = opts.ExpiresAt
	p.fresh = opts.Fresh
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
