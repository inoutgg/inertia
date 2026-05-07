package inertia

import (
	"cmp"
	"context"
)

var (
	_ Prop   = (*alwaysProp)(nil)
	_ Prop   = (*deferredProp)(nil)
	_ Prop   = (*onceProp)(nil)
	_ Prop   = (*optionalProp)(nil)
	_ Prop   = (*scrollProp)(nil)
	_ Prop   = (*standardProp)(nil)
	_ Proper = (Props)(nil)
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
type Prop interface {
	Key() string
	Value(context.Context) (any, error)
	IgnoreFirstLoad() bool
	BypassPartialFilters() bool
	Deferrable() (*deferrable, bool)
	Mergeable() (*mergeable, bool)
	Scrollable() (*scrollable, bool)
	Onceable() (*onceable, bool)
	Concurrent() bool
}

type standardProp struct {
	valFn Lazy
	val   any
	once  *onceable
	merge *mergeable
	key   string

	concurrent bool
}

func (p *standardProp) Key() string { return p.key }

func (p *standardProp) Value(ctx context.Context) (any, error) {
	if p.valFn != nil {
		return p.valFn.Value(ctx) //nolint:wrapcheck
	}

	return p.val, nil
}

func (p *standardProp) IgnoreFirstLoad() bool           { return false }
func (p *standardProp) BypassPartialFilters() bool      { return false }
func (p *standardProp) Deferrable() (*deferrable, bool) { return nil, false }
func (p *standardProp) Mergeable() (*mergeable, bool)   { return p.merge, p.merge != nil }
func (p *standardProp) Scrollable() (*scrollable, bool) { return nil, false }
func (p *standardProp) Onceable() (*onceable, bool)     { return p.once, p.once != nil }
func (p *standardProp) Concurrent() bool                { return p.concurrent }

type deferredProp struct {
	valFn    Lazy
	val      any
	once     *onceable
	deferred *deferrable
	merge    *mergeable
	key      string

	concurrent bool
}

func (p *deferredProp) Key() string { return p.key }

func (p *deferredProp) Value(ctx context.Context) (any, error) {
	if p.valFn != nil {
		return p.valFn.Value(ctx) //nolint:wrapcheck
	}

	return p.val, nil
}

func (p *deferredProp) IgnoreFirstLoad() bool      { return true }
func (p *deferredProp) BypassPartialFilters() bool { return false }
func (p *deferredProp) Deferrable() (*deferrable, bool) {
	return p.deferred, p.deferred != nil
}
func (p *deferredProp) Mergeable() (*mergeable, bool)   { return p.merge, p.merge != nil }
func (p *deferredProp) Scrollable() (*scrollable, bool) { return nil, false }
func (p *deferredProp) Onceable() (*onceable, bool)     { return p.once, p.once != nil }
func (p *deferredProp) Concurrent() bool                { return p.concurrent }

type scrollProp struct {
	valFn  Lazy
	val    any
	scroll *scrollable
	merge  *mergeable
	key    string
}

func (p *scrollProp) Key() string { return p.key }

func (p *scrollProp) Value(ctx context.Context) (any, error) {
	if p.valFn != nil {
		return p.valFn.Value(ctx) //nolint:wrapcheck
	}

	return p.val, nil
}

func (p *scrollProp) IgnoreFirstLoad() bool           { return false }
func (p *scrollProp) BypassPartialFilters() bool      { return false }
func (p *scrollProp) Deferrable() (*deferrable, bool) { return nil, false }
func (p *scrollProp) Mergeable() (*mergeable, bool)   { return p.merge, p.merge != nil }
func (p *scrollProp) Scrollable() (*scrollable, bool) {
	return p.scroll, p.scroll != nil
}
func (p *scrollProp) Onceable() (*onceable, bool) { return nil, false }
func (p *scrollProp) Concurrent() bool            { return false }

type alwaysProp struct {
	valFn Lazy
	val   any
	key   string
}

func (p *alwaysProp) Key() string { return p.key }

func (p *alwaysProp) Value(ctx context.Context) (any, error) {
	if p.valFn != nil {
		return p.valFn.Value(ctx) //nolint:wrapcheck
	}

	return p.val, nil
}

func (p *alwaysProp) IgnoreFirstLoad() bool           { return false }
func (p *alwaysProp) BypassPartialFilters() bool      { return true }
func (p *alwaysProp) Deferrable() (*deferrable, bool) { return nil, false }
func (p *alwaysProp) Mergeable() (*mergeable, bool)   { return nil, false }
func (p *alwaysProp) Scrollable() (*scrollable, bool) { return nil, false }
func (p *alwaysProp) Onceable() (*onceable, bool)     { return nil, false }
func (p *alwaysProp) Concurrent() bool                { return false }

type optionalProp struct {
	valFn Lazy
	val   any
	once  *onceable
	key   string

	concurrent bool
}

func (p *optionalProp) Key() string { return p.key }

func (p *optionalProp) Value(ctx context.Context) (any, error) {
	if p.valFn != nil {
		return p.valFn.Value(ctx) //nolint:wrapcheck
	}

	return p.val, nil
}

func (p *optionalProp) IgnoreFirstLoad() bool           { return true }
func (p *optionalProp) BypassPartialFilters() bool      { return false }
func (p *optionalProp) Deferrable() (*deferrable, bool) { return nil, false }
func (p *optionalProp) Mergeable() (*mergeable, bool)   { return nil, false }
func (p *optionalProp) Scrollable() (*scrollable, bool) { return nil, false }
func (p *optionalProp) Onceable() (*onceable, bool)     { return p.once, p.once != nil }
func (p *optionalProp) Concurrent() bool                { return p.concurrent }

type onceProp struct {
	valFn Lazy
	val   any
	once  *onceable
	key   string

	concurrent bool
}

func (p *onceProp) Key() string { return p.key }

func (p *onceProp) Value(ctx context.Context) (any, error) {
	if p.valFn != nil {
		return p.valFn.Value(ctx) //nolint:wrapcheck
	}

	return p.val, nil
}

func (p *onceProp) IgnoreFirstLoad() bool           { return false }
func (p *onceProp) BypassPartialFilters() bool      { return false }
func (p *onceProp) Deferrable() (*deferrable, bool) { return nil, false }
func (p *onceProp) Mergeable() (*mergeable, bool)   { return nil, false }
func (p *onceProp) Scrollable() (*scrollable, bool) { return nil, false }
func (p *onceProp) Onceable() (*onceable, bool)     { return p.once, p.once != nil }
func (p *onceProp) Concurrent() bool                { return p.concurrent }

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
	prop := &deferredProp{
		valFn: fn,
		val:   nil,
		key:   key,
		once:  nil,
		deferred: &deferrable{
			group: DefaultDeferredGroup,
		},
		merge:      nil,
		concurrent: false,
	}

	if opts != nil {
		opts.validate()

		prop.deferred.group = cmp.Or(opts.Group, DefaultDeferredGroup)
		prop.merge = mergeFromOptions(opts.Merge, opts.Prepend, opts.DeepMerge, opts.MatchOn)
		prop.once, prop.concurrent = onceFromOptions(key, opts.Once)
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
	return &alwaysProp{
		valFn: nil,
		val:   val,
		key:   key,
	}
}

// NewOptional creates a lazily-evaluated prop included only during partial reloads when explicitly requested.
// Useful for expensive computations that aren't needed on every render.
//
// The value function is only called when the client specifically requests this prop.
func NewOptional(key string, fn Lazy) Prop {
	return &optionalProp{
		valFn:      fn,
		val:        nil,
		key:        key,
		once:       nil,
		concurrent: false,
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

	once, concurrent := onceFromOptions(key, opts)

	return &onceProp{
		valFn:      fn,
		val:        nil,
		key:        key,
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
	Metadata *scrollable
	Wrapper  string
}

// NewScrollOptions creates type-safe infinite scroll options.
func NewScrollOptions[T ScrollPage](wrapper string, metadata ScrollMetadata[T]) *ScrollOptions {
	return &ScrollOptions{
		Wrapper: wrapper,
		Metadata: &scrollable{
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
	val := value

	var valFn Lazy

	if lazy, ok := value.(Lazy); ok {
		val = nil
		valFn = lazy
	}

	prop := &scrollProp{
		valFn: valFn,
		val:   val,
		key:   key,
		scroll: &scrollable{
			PreviousPage: nil,
			NextPage:     nil,
			CurrentPage:  nil,
			PageName:     "",
			path:         key + ".data",
		},
		merge: &mergeable{
			matchOn:   nil,
			deepMerge: false,
			prepend:   false,
		},
	}

	if opts != nil && opts.Metadata != nil {
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
	prop := &standardProp{
		valFn:      nil,
		val:        val,
		key:        key,
		once:       nil,
		merge:      nil,
		concurrent: false,
	}

	if opts != nil {
		opts.validate()

		prop.merge = mergeFromOptions(opts.Merge, opts.Prepend, opts.DeepMerge, opts.MatchOn)
		prop.once, prop.concurrent = onceFromOptions(key, opts.Once)
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

func mergeFromOptions(merge, prepend, deepMerge bool, matchOn []string) *mergeable {
	if !merge && !prepend && !deepMerge {
		return nil
	}

	return &mergeable{
		matchOn:   matchOn,
		deepMerge: deepMerge,
		prepend:   prepend,
	}
}

func onceFromOptions(defaultKey string, opts *OnceOptions) (*onceable, bool) {
	if opts == nil {
		return nil, false
	}

	return &onceable{
		key:       cmp.Or(opts.Key, defaultKey),
		expiresAt: opts.ExpiresAt,
		fresh:     opts.Fresh,
	}, opts.Concurrent
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
