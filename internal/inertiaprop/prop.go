package inertiaprop

import (
	"context"
	"slices"
)

// Prop represents a single property passed to an Inertia page component.
type Prop interface {
	Key() string
	Value(context.Context) (any, error)
	IsFirstLoadIgnorable() bool
	BypassPartialFilters() bool
	Deferrable() (*Deferrable, bool)
	Mergeable() (*Mergeable, bool)
	Scrollable() (*Scrollable, bool)
	Onceable() (*Onceable, bool)
	Concurrent() bool
}

// Lazy represents a prop value that is resolved on-demand rather than eagerly.
type Lazy interface {
	// Value resolves and returns the prop's value.
	//
	// The returned value must be JSON-serializable.
	Value(context.Context) (any, error)
}

// LazyFunc is a function adapter that implements the Lazy interface.
//
// It allows using ordinary functions as lazy prop values.
// The returned value must be JSON-serializable.
type LazyFunc func(context.Context) (any, error)

// Value calls `fn()`.
func (fn LazyFunc) Value(ctx context.Context) (any, error) { return fn(ctx) }

type Deferrable struct {
	Group  string
	Rescue bool
}

type MergeKey struct {
	Key     string
	MatchOn string
}

// MergeOpts configures merge prop behavior.
type MergeOpts struct {
	appendKeys  []MergeKey
	prependKeys []MergeKey

	append  bool
	prepend bool
}

// NewMergeOpts creates a default MergeOpts instance with append behavior enabled.
func NewMergeOpts() *MergeOpts {
	//nolint:exhaustruct
	return &MergeOpts{append: true}
}

// Append configures the merge prop to append keys to the existing prop value.
func (o *MergeOpts) Append(keys ...MergeKey) *MergeOpts {
	if len(keys) > 0 {
		o.appendKeys = append(o.appendKeys, keys...)
		o.append = false

		return o
	}

	o.append = true
	o.prepend = false

	return o
}

// Prepend configures the merge prop to prepend keys to the existing prop value.
func (o *MergeOpts) Prepend(keys ...MergeKey) *MergeOpts {
	o.append = false

	if len(keys) > 0 {
		o.prependKeys = append(o.prependKeys, keys...)

		return o
	}

	o.prepend = true

	return o
}

func (o *MergeOpts) Mergeable() *Mergeable {
	if o == nil {
		return nil
	}

	return &Mergeable{
		Append:      o.append,
		Prepend:     o.prepend,
		AppendKeys:  slices.Clone(o.appendKeys),
		PrependKeys: slices.Clone(o.prependKeys),
	}
}

type Mergeable struct {
	AppendKeys  []MergeKey
	PrependKeys []MergeKey

	Append  bool
	Prepend bool
}

type Scrollable struct {
	PreviousPage any
	NextPage     any
	CurrentPage  any
	PageName     string
	Path         string
}

type Onceable struct {
	ExpiresAt *int64
	Key       string
	Fresh     bool
}

// OnceOpts configures once prop behavior.
type OnceOpts struct {
	expiresAt *int64
	key       string
	fresh     bool
}

func (o *OnceOpts) Key(key string) *OnceOpts {
	o.key = key
	return o
}

func (o *OnceOpts) ExpiresAt(expiresAt *int64) *OnceOpts {
	o.expiresAt = expiresAt
	return o
}

func (o *OnceOpts) Fresh(fresh bool) *OnceOpts {
	o.fresh = fresh
	return o
}

func (o *OnceOpts) Onceable() *Onceable {
	if o == nil {
		return nil
	}

	return &Onceable{
		ExpiresAt: o.expiresAt,
		Key:       o.key,
		Fresh:     o.fresh,
	}
}
