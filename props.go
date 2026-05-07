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
	value    value
	scroll   scrollable
	once     onceable
	deferred deferrable
	merge    mergeable
	partial  partial
}

type value struct {
	fn  Lazy
	val any
	key string
}

type partial struct {
	ignorable bool
	lazy      bool
}

type deferrable struct {
	group      string
	enabled    bool
	concurrent bool
}

type mergeable struct {
	matchOn   []string
	deepMerge bool
	prepend   bool
	enabled   bool
}

type scrollable struct {
	meta    scrollMetadata
	path    string
	enabled bool
}

type onceable struct {
	expiresAt *int64
	key       string
	fresh     bool
	enabled   bool
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
		deferred: deferrable{
			enabled: true, // important
			group:   DefaultDeferredGroup,
		},
		partial: partial{
			lazy:      true, // important
			ignorable: true, // important
		},
		value: value{
			key: key,
			fn:  fn,
		},
	}

	if opts != nil {
		prop.deferred.group = cmp.Or(opts.Group, DefaultDeferredGroup)
		prop.merge.enabled = opts.Merge
		prop.merge.prepend = opts.Prepend
		prop.merge.deepMerge = opts.DeepMerge
		prop.merge.matchOn = opts.MatchOn
		prop = applyOnceOptions(prop, key, opts.Once)
		prop.deferred.concurrent = opts.Concurrent
	}

	return prop
}

// NewAlways creates a prop that is always included in responses.
// Unlike regular props, it ignores partial reload filters (X-Inertia-Partial-Data/Except headers).
//
// It is particularly useful to enforce load of critical data that must always be present,
// such as authentication state or global config.
func NewAlways(key string, val any) Prop {
	//nolint:exhaustruct
	return Prop{
		partial: partial{
			ignorable: false, // important
		},
		value: value{
			key: key,
			val: val,
		},
	}
}

// NewOptional creates a lazily-evaluated prop included only during partial reloads when explicitly requested.
// Useful for expensive computations that aren't needed on every render.
//
// The value function is only called when the client specifically requests this prop.
func NewOptional(key string, fn Lazy) Prop {
	//nolint:exhaustruct
	return Prop{
		partial: partial{
			ignorable: true, // important
			lazy:      true, // important
		},
		value: value{
			key: key,
			fn:  fn,
		},
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
		partial: partial{
			ignorable: true, // important
		},
		value: value{
			key: key,
			fn:  fn,
		},
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
		prop.value.val = nil
		prop.value.fn = lazy
	}

	prop.scroll.enabled = true
	prop.scroll.path = key + ".data"

	if opts != nil {
		prop.scroll.path = qualifyPropPath(key, cmp.Or(opts.Wrapper, "data"))
		prop.scroll.meta = opts.Metadata
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
		partial: partial{
			ignorable: true, // important
		},
		value: value{
			key: key,
			val: val,
		},
	}

	if opts != nil {
		prop.merge.enabled = opts.Merge
		prop.merge.prepend = opts.Prepend
		prop.merge.deepMerge = opts.DeepMerge
		prop.merge.matchOn = opts.MatchOn
		prop = applyOnceOptions(prop, key, opts.Once)
	}

	return prop
}

func applyOnceOptions(prop Prop, defaultKey string, opts *OnceOptions) Prop {
	if opts == nil {
		return prop
	}

	prop.once.enabled = true
	prop.once.key = cmp.Or(opts.Key, defaultKey)
	prop.once.expiresAt = opts.ExpiresAt
	prop.once.fresh = opts.Fresh

	return prop
}

func (p Prop) Props() []Prop { return []Prop{p} }
func (p Prop) Len() int      { return 1 }

func (p Prop) key() string { return p.value.key }

func (p Prop) includeOnInitial() bool { return !p.partial.lazy }

func (p Prop) ignorePartialFilters() bool { return !p.partial.ignorable }

func (p Prop) deferrable() (deferrable, bool) {
	return p.deferred, p.deferred.enabled
}

func (p Prop) mergeable() (mergeable, bool) {
	return p.merge, p.merge.enabled
}

func (p Prop) scrollable() (scrollable, bool) {
	return p.scroll, p.scroll.enabled
}

func (p Prop) onceable() (onceable, bool) {
	return p.once, p.once.enabled
}

func (p Prop) resolveConcurrently() bool { return p.deferred.concurrent }

// resolveValue returns the prop value.
func (p Prop) resolveValue(ctx context.Context) (any, error) {
	if p.value.fn != nil {
		v, err := p.value.fn.Value(ctx)
		if err != nil {
			return nil, err //nolint:wrapcheck
		}

		return v, nil
	}

	return p.value.val, nil
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
