package inertia

import (
	"cmp"
	"context"
)

var (
	_ Prop   = (*AlwaysProp)(nil)
	_ Prop   = (*DeferredProp)(nil)
	_ Prop   = (*OnceProp)(nil)
	_ Prop   = (*OptionalProp)(nil)
	_ Prop   = (*ScrollProp)(nil)
	_ Prop   = (*StandardProp)(nil)
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
	IsFirstLoadIgnorable() bool
	BypassPartialFilters() bool
	Deferrable() (*deferrable, bool)
	Mergeable() (*mergeable, bool)
	Scrollable() (*scrollable, bool)
	Onceable() (*onceable, bool)
	Concurrent() bool
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

// PropOpts configures standard prop behavior.
type PropOpts struct {
	Once  *OnceOpts
	Merge *MergeOpts
}

func (opts *PropOpts) validate() {
	// TODO: validate that merge, prepend, and deep merge are mutually exclusive
}

// OnceOpts configures once prop behavior.
type OnceOpts struct {
	ExpiresAt *int64
	Fresh     bool
}

// DeferredOpts configures the behavior of deferred props.
type DeferredOpts struct {
	Once       *OnceOpts
	Group      string
	Merge      *MergeOpts
	Concurrent bool
}

func (opts DeferredOpts) validate() {
	// TODO: validate that merge, prepend, and deep merge are mutually exclusive
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

// ScrollOpts configures infinite scroll prop behavior.
type ScrollOpts struct {
	Metadata *scrollable
	Wrapper  string
}

type StandardProp struct {
	val   any
	once  *onceable
	merge *mergeable
	key   string

	concurrent bool
}

// NewProp creates a standard prop included on initial page load and partial reloads.
//
// If opts is nil, default options are used (no merging).
func NewProp(key string, val any, opts *PropOpts) *StandardProp {
	prop := &StandardProp{
		val:        val,
		once:       nil,
		merge:      nil,
		key:        key,
		concurrent: false,
	}

	if opts != nil {
		opts.validate()
	}

	return prop
}

func (p *StandardProp) Key() string                        { return p.key }
func (p *StandardProp) Value(context.Context) (any, error) { return p.val, nil }

func (p *StandardProp) IsFirstLoadIgnorable() bool      { return false }
func (p *StandardProp) BypassPartialFilters() bool      { return false }
func (p *StandardProp) Deferrable() (*deferrable, bool) { return nil, false }
func (p *StandardProp) Mergeable() (*mergeable, bool)   { return p.merge, p.merge != nil }
func (p *StandardProp) Scrollable() (*scrollable, bool) { return nil, false }
func (p *StandardProp) Onceable() (*onceable, bool)     { return p.once, p.once != nil }
func (p *StandardProp) Concurrent() bool                { return p.concurrent }

type DeferredProp struct {
	val      Lazy
	once     *onceable
	deferred *deferrable
	merge    *mergeable
	key      string

	concurrent bool
}

// NewDeferred creates a deferred prop that is lazy-loaded by the client after initial render.
// Deferred props reduce initial page load time by deferring expensive computations.
//
// If opts is nil, default options are used (default group, no merging, sequential resolution).
func NewDeferred(key string, val Lazy, opts *DeferredOpts) *DeferredProp {
	prop := &DeferredProp{
		val:      val,
		once:     nil,
		deferred: &deferrable{group: DefaultDeferredGroup},
		merge:    nil,
		key:      key,
	}

	if opts != nil {
		opts.validate()

		prop.deferred.group = cmp.Or(opts.Group, DefaultDeferredGroup)
		prop.concurrent = opts.Concurrent
	}

	return prop
}

func (p *DeferredProp) Key() string                            { return p.key }
func (p *DeferredProp) Value(ctx context.Context) (any, error) { return p.val.Value(ctx) } //nolint:wrapcheck

func (p *DeferredProp) IsFirstLoadIgnorable() bool      { return true }
func (p *DeferredProp) BypassPartialFilters() bool      { return false }
func (p *DeferredProp) Deferrable() (*deferrable, bool) { return p.deferred, p.deferred != nil }
func (p *DeferredProp) Mergeable() (*mergeable, bool)   { return p.merge, p.merge != nil }
func (p *DeferredProp) Scrollable() (*scrollable, bool) { return nil, false }
func (p *DeferredProp) Onceable() (*onceable, bool)     { return p.once, p.once != nil }
func (p *DeferredProp) Concurrent() bool                { return p.concurrent }

type ScrollProp struct {
	valFn  Lazy
	val    any
	scroll *scrollable
	merge  *mergeable
	key    string
}

// NewScrollOptions creates type-safe infinite scroll options.
func NewScrollOptions[T ScrollPage](wrapper string, metadata ScrollMetadata[T]) *ScrollOpts {
	return &ScrollOpts{
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
func NewScroll(key string, value any, opts *ScrollOpts) ScrollProp {
	val := value

	var valFn Lazy

	if lazy, ok := value.(Lazy); ok {
		val = nil
		valFn = lazy
	}

	prop := ScrollProp{
		valFn: valFn,
		val:   val,
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
		key: key,
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

func (p *ScrollProp) Key() string { return p.key }

func (p *ScrollProp) Value(ctx context.Context) (any, error) {
	if p.valFn != nil {
		return p.valFn.Value(ctx) //nolint:wrapcheck
	}

	return p.val, nil
}

func (p *ScrollProp) IsFirstLoadIgnorable() bool      { return false }
func (p *ScrollProp) BypassPartialFilters() bool      { return false }
func (p *ScrollProp) Deferrable() (*deferrable, bool) { return nil, false }
func (p *ScrollProp) Mergeable() (*mergeable, bool)   { return p.merge, p.merge != nil }
func (p *ScrollProp) Scrollable() (*scrollable, bool) { return p.scroll, p.scroll != nil }
func (p *ScrollProp) Onceable() (*onceable, bool)     { return nil, false }
func (p *ScrollProp) Concurrent() bool                { return false }

// AlwaysProp is a prop that is always included in responses, regardless of partial reload filters.
type AlwaysProp struct {
	valFn Lazy
	val   any
	key   string
}

// NewAlways creates a prop that is always included in responses.
// Unlike regular props, it ignores partial reload filters (X-Inertia-Partial-Data/Except headers).
//
// It is particularly useful to enforce load of critical data that must always be present,
// such as authentication state or global config.
func NewAlways(key string, val any) *AlwaysProp {
	if lazy, ok := val.(Lazy); ok {
		return &AlwaysProp{
			valFn: lazy,
			key:   key,
		}
	}

	return &AlwaysProp{
		val: val,
		key: key,
	}
}

func (p *AlwaysProp) Key() string { return p.key }

func (p *AlwaysProp) Value(ctx context.Context) (any, error) {
	if p.valFn != nil {
		//nolint:wrapcheck
		return p.valFn.Value(ctx)
	}

	return p.val, nil
}

func (p *AlwaysProp) IsFirstLoadIgnorable() bool      { return false }
func (p *AlwaysProp) BypassPartialFilters() bool      { return true }
func (p *AlwaysProp) Deferrable() (*deferrable, bool) { return nil, false }
func (p *AlwaysProp) Mergeable() (*mergeable, bool)   { return nil, false }
func (p *AlwaysProp) Scrollable() (*scrollable, bool) { return nil, false }
func (p *AlwaysProp) Onceable() (*onceable, bool)     { return nil, false }
func (p *AlwaysProp) Concurrent() bool                { return false }

// OptionalProp is a lazily-evaluated prop included only during partial reloads when explicitly requested.
type OptionalProp struct {
	val        Lazy
	key        string
	concurrent bool
}

// NewOptional creates a lazily-evaluated prop included only during partial reloads when explicitly requested.
//
// It is useful for expensive computations that aren't needed on every render.
//
// The value function is only called when the client specifically requests this prop.
func NewOptional(key string, val Lazy) *OptionalProp {
	return &OptionalProp{
		val:        val,
		key:        key,
		concurrent: false,
	}
}

func (p *OptionalProp) Key() string                            { return p.key }
func (p *OptionalProp) Value(ctx context.Context) (any, error) { return p.val.Value(ctx) } //nolint:wrapcheck

func (p *OptionalProp) IsFirstLoadIgnorable() bool      { return true }
func (p *OptionalProp) BypassPartialFilters() bool      { return false }
func (p *OptionalProp) Deferrable() (*deferrable, bool) { return nil, false }
func (p *OptionalProp) Mergeable() (*mergeable, bool)   { return nil, false }
func (p *OptionalProp) Scrollable() (*scrollable, bool) { return nil, false }
func (p *OptionalProp) Onceable() (*onceable, bool)     { return nil, false }
func (p *OptionalProp) Concurrent() bool                { return p.concurrent }

type OnceProp struct {
	val  Lazy
	once *onceable
	key  string

	concurrent bool
}

// NewOnce creates a prop remembered by the client and skipped on subsequent visits.
func NewOnce(key string, val Lazy, opts *OnceOpts) *OnceProp {
	if opts == nil {
		opts = &OnceOpts{} //nolint:exhaustruct
	}

	return &OnceProp{
		val: val,
		key: key,
	}
}

func (p *OnceProp) Key() string                            { return p.key }
func (p *OnceProp) Value(ctx context.Context) (any, error) { return p.val.Value(ctx) } //nolint:wrapcheck

func (p *OnceProp) IsFirstLoadIgnorable() bool      { return false }
func (p *OnceProp) BypassPartialFilters() bool      { return false }
func (p *OnceProp) Deferrable() (*deferrable, bool) { return nil, false }
func (p *OnceProp) Mergeable() (*mergeable, bool)   { return nil, false }
func (p *OnceProp) Scrollable() (*scrollable, bool) { return nil, false }
func (p *OnceProp) Onceable() (*onceable, bool)     { return p.once, p.once != nil }
func (p *OnceProp) Concurrent() bool                { return p.concurrent }

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
