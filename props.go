package inertia

import (
	"cmp"
	"context"
)

var _ Proper = (Props)(nil)

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
type Prop interface {
	Key() string
	Value(context.Context) (any, error)
}

type (
	firstLoadIgnorable interface {
		IgnoreFirstLoad() bool
	}

	partialFilterBypasser interface {
		BypassPartialFilters() bool
	}

	deferrableProp interface {
		Deferrable() (deferrable, bool)
	}

	mergeableProp interface {
		Mergeable() (mergeable, bool)
	}

	scrollableProp interface {
		Scrollable() (scrollable, bool)
	}

	onceableProp interface {
		Onceable() (onceable, bool)
	}

	concurrentProp interface {
		Concurrent() bool
	}
)

type baseProp struct {
	valFn Lazy
	val   any
	key   string
}

func (p baseProp) Key() string { return p.key }
func (p baseProp) Value(ctx context.Context) (any, error) {
	if p.valFn != nil {
		return p.valFn.Value(ctx) //nolint:wrapcheck
	}

	return p.val, nil
}

type standardProp struct {
	baseProp

	once       onceable
	merge      mergeable
	concurrent bool
	mergeable  bool
	onceable   bool
}

func (p standardProp) Mergeable() (mergeable, bool) { return p.merge, p.mergeable }
func (p standardProp) Onceable() (onceable, bool)   { return p.once, p.onceable }
func (p standardProp) Concurrent() bool             { return p.concurrent }

type deferredProp struct {
	baseProp

	once       onceable
	deferred   deferrable
	merge      mergeable
	concurrent bool
	mergeable  bool
	onceable   bool
}

func (p deferredProp) IgnoreFirstLoad() bool          { return true }
func (p deferredProp) Deferrable() (deferrable, bool) { return p.deferred, true }
func (p deferredProp) Mergeable() (mergeable, bool)   { return p.merge, p.mergeable }
func (p deferredProp) Onceable() (onceable, bool)     { return p.once, p.onceable }
func (p deferredProp) Concurrent() bool               { return p.concurrent }

type scrollProp struct {
	baseProp

	scroll scrollable
	merge  mergeable
}

func (p scrollProp) Scrollable() (scrollable, bool) { return p.scroll, true }
func (p scrollProp) Mergeable() (mergeable, bool)   { return p.merge, true }

type alwaysProp struct {
	baseProp
}

func (p alwaysProp) BypassPartialFilters() bool { return true }

type optionalProp struct {
	baseProp

	once       onceable
	concurrent bool
	onceable   bool
}

func (p optionalProp) IgnoreFirstLoad() bool      { return true }
func (p optionalProp) Onceable() (onceable, bool) { return p.once, p.onceable }
func (p optionalProp) Concurrent() bool           { return p.concurrent }

type onceProp struct {
	baseProp

	once       onceable
	concurrent bool
}

func (p onceProp) Onceable() (onceable, bool) { return p.once, true }
func (p onceProp) Concurrent() bool           { return p.concurrent }

type deferrable struct {
	group string
}

type mergeable struct {
	matchOn   []string
	deepMerge bool
	prepend   bool
}

type scrollable struct {
	PreviousPage any
	NextPage     any
	CurrentPage  any
	PageName     string
	path         string
}

type onceable struct {
	expiresAt *int64
	key       string
	fresh     bool
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
	prop := deferredProp{
		baseProp: baseProp{
			key:   key,
			valFn: fn,
			val:   nil,
		},
		deferred: deferrable{
			group: DefaultDeferredGroup,
		},
		once: onceable{
			expiresAt: nil,
			key:       "",
			fresh:     false,
		},
		merge: mergeable{
			matchOn:   nil,
			deepMerge: false,
			prepend:   false,
		},
		concurrent: false,
		mergeable:  false,
		onceable:   false,
	}

	if opts != nil {
		opts.validate()

		prop.deferred.group = cmp.Or(opts.Group, DefaultDeferredGroup)
		prop.mergeable = opts.Merge || opts.Prepend || opts.DeepMerge
		prop.merge.prepend = opts.Prepend
		prop.merge.deepMerge = opts.DeepMerge
		prop.merge.matchOn = opts.MatchOn
		prop.once, prop.onceable, prop.concurrent = onceFromOptions(key, opts.Once)
		prop.concurrent = opts.Concurrent || prop.concurrent
	}

	return prop
}

// NewAlways creates a prop that is always included in responses.
// Unlike regular props, it ignores partial reload filters (X-Inertia-Partial-Data/Except headers).
//
// It is particularly useful to enforce load of critical data that must always be present,
// such as authentication state or global config.
func NewAlways(key string, val any) Prop {
	return alwaysProp{
		baseProp: baseProp{
			key:   key,
			val:   val,
			valFn: nil,
		},
	}
}

// NewOptional creates a lazily-evaluated prop included only during partial reloads when explicitly requested.
// Useful for expensive computations that aren't needed on every render.
//
// The value function is only called when the client specifically requests this prop.
func NewOptional(key string, fn Lazy) Prop {
	return optionalProp{
		baseProp: baseProp{
			key:   key,
			valFn: fn,
			val:   nil,
		},
		once: onceable{
			expiresAt: nil,
			key:       "",
			fresh:     false,
		},
		concurrent: false,
		onceable:   false,
	}
}

// OnceOptions configures once prop behavior.
type OnceOptions struct {
	ExpiresAt  *int64
	Key        string
	Fresh      bool
	Concurrent bool
}

// NewOnce creates a prop remembered by the client and skipped on subsequent visits.
func NewOnce(key string, fn Lazy, opts *OnceOptions) Prop {
	if opts == nil {
		opts = &OnceOptions{} //nolint:exhaustruct
	}

	once, _, concurrent := onceFromOptions(key, opts)

	return onceProp{
		baseProp: baseProp{
			key:   key,
			valFn: fn,
			val:   nil,
		},
		once:       once,
		concurrent: concurrent,
	}
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

// ScrollOptions configures infinite scroll prop behavior.
type ScrollOptions struct {
	Wrapper  string
	Metadata scrollable
}

// NewScrollOptions creates type-safe infinite scroll options.
func NewScrollOptions[T ScrollPage](wrapper string, metadata ScrollMetadata[T]) *ScrollOptions {
	return &ScrollOptions{
		Wrapper: wrapper,
		Metadata: scrollable{
			PageName:     metadata.PageName,
			PreviousPage: metadata.PreviousPage,
			NextPage:     metadata.NextPage,
			CurrentPage:  metadata.CurrentPage,
			path:         "",
		},
	}
}

// NewScroll creates an infinite scroll prop with v3 scroll metadata.
func NewScroll(key string, value any, opts *ScrollOptions) Prop {
	base := baseProp{
		key:   key,
		val:   value,
		valFn: nil,
	}
	if lazy, ok := value.(Lazy); ok {
		base.val = nil
		base.valFn = lazy
	}

	prop := scrollProp{
		baseProp: base,
		scroll: scrollable{
			PreviousPage: nil,
			NextPage:     nil,
			CurrentPage:  nil,
			PageName:     "",
			path:         key + ".data",
		},
		merge: mergeable{
			matchOn:   nil,
			deepMerge: false,
			prepend:   false,
		},
	}

	if opts != nil {
		prop.scroll.PageName = opts.Metadata.PageName
		prop.scroll.PreviousPage = opts.Metadata.PreviousPage
		prop.scroll.NextPage = opts.Metadata.NextPage
		prop.scroll.CurrentPage = opts.Metadata.CurrentPage
		prop.scroll.path = qualifyPropPath(key, cmp.Or(opts.Wrapper, "data"))
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

func (opts PropOptions) validate() {
	modes := 0
	if opts.Merge {
		modes++
	}

	if opts.Prepend {
		modes++
	}

	if opts.DeepMerge {
		modes++
	}

	if modes > 1 {
		panic("inertia: merge, prepend, and deep merge are mutually exclusive")
	}
}

// NewProp creates a standard prop included on initial page load and partial reloads.
//
// If opts is nil, default options are used (no merging).
func NewProp(key string, val any, opts *PropOptions) Prop {
	prop := standardProp{
		baseProp: baseProp{
			key:   key,
			val:   val,
			valFn: nil,
		},
		once: onceable{
			expiresAt: nil,
			key:       "",
			fresh:     false,
		},
		merge: mergeable{
			matchOn:   nil,
			deepMerge: false,
			prepend:   false,
		},
		concurrent: false,
		mergeable:  false,
		onceable:   false,
	}

	if opts != nil {
		opts.validate()

		prop.mergeable = opts.Merge || opts.Prepend || opts.DeepMerge
		prop.merge.prepend = opts.Prepend
		prop.merge.deepMerge = opts.DeepMerge
		prop.merge.matchOn = opts.MatchOn
		prop.once, prop.onceable, prop.concurrent = onceFromOptions(key, opts.Once)
	}

	return prop
}

func (opts DeferredOptions) validate() {
	modes := 0
	if opts.Merge {
		modes++
	}

	if opts.Prepend {
		modes++
	}

	if opts.DeepMerge {
		modes++
	}

	if modes > 1 {
		panic("inertia: merge, prepend, and deep merge are mutually exclusive")
	}
}

func onceFromOptions(defaultKey string, opts *OnceOptions) (onceable, bool, bool) {
	if opts == nil {
		return onceable{
			expiresAt: nil,
			key:       "",
			fresh:     false,
		}, false, false
	}

	return onceable{
		key:       cmp.Or(opts.Key, defaultKey),
		expiresAt: opts.ExpiresAt,
		fresh:     opts.Fresh,
	}, true, opts.Concurrent
}

func shouldIgnoreFirstLoad(prop Prop) bool {
	ignorable, ok := prop.(firstLoadIgnorable)
	return ok && ignorable.IgnoreFirstLoad()
}

func shouldBypassPartialFilters(prop Prop) bool {
	bypasser, ok := prop.(partialFilterBypasser)
	return ok && bypasser.BypassPartialFilters()
}

func isConcurrent(prop Prop) bool {
	concurrent, ok := prop.(concurrentProp)
	return ok && concurrent.Concurrent()
}

func getDeferrable(prop Prop) (deferrable, bool) {
	deferrableProp, ok := prop.(deferrableProp)
	if !ok {
		return deferrable{group: ""}, false
	}

	return deferrableProp.Deferrable()
}

func getMergeable(prop Prop) (mergeable, bool) {
	mergeableProp, ok := prop.(mergeableProp)
	if !ok {
		return mergeable{
			matchOn:   nil,
			deepMerge: false,
			prepend:   false,
		}, false
	}

	return mergeableProp.Mergeable()
}

func getScrollable(prop Prop) (scrollable, bool) {
	scrollableProp, ok := prop.(scrollableProp)
	if !ok {
		return scrollable{
			PreviousPage: nil,
			NextPage:     nil,
			CurrentPage:  nil,
			PageName:     "",
			path:         "",
		}, false
	}

	return scrollableProp.Scrollable()
}

func getOnceable(prop Prop) (onceable, bool) {
	onceableProp, ok := prop.(onceableProp)
	if !ok {
		return onceable{
			expiresAt: nil,
			key:       "",
			fresh:     false,
		}, false
	}

	return onceableProp.Onceable()
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
