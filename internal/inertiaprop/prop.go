package inertiaprop

import (
	"context"
	"fmt"
	"slices"
)

const (
	ScrollMergeIntentAppend  = "append"
	ScrollMergeIntentPrepend = "prepend"
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

// RescueError is returned by a deferred prop's Value method when the prop
// is configured with rescue and fails to resolve.
type RescueError struct {
	Err error
	Key string
}

func (e *RescueError) Error() string {
	return fmt.Sprintf("inertia: rescued deferred prop %s: %v", e.Key, e.Err)
}

func (e *RescueError) Unwrap() error {
	return e.Err
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
	} else {
		o.append = true
		o.prepend = false
	}

	return o
}

// Prepend configures the merge prop to prepend keys to the existing prop value.
func (o *MergeOpts) Prepend(keys ...MergeKey) *MergeOpts {
	if len(keys) > 0 {
		o.prependKeys = append(o.prependKeys, keys...)
	} else {
		o.prepend = true
		o.append = false
	}

	return o
}

// OnceOpts configures once prop behavior.
type OnceOpts struct {
	expiresAt *int64
	key       string
	fresh     bool
}

func NewOnceOpts() *OnceOpts {
	return &OnceOpts{} //nolint:exhaustruct
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

type Deferrable struct {
	Group  string
	Rescue bool
}

type Onceable struct {
	ExpiresAt *int64
	Key       string
	Fresh     bool
}

// ToOnceable converts OnceOpts to Onceable.
func ToOnceable(opts *OnceOpts) *Onceable {
	if opts == nil {
		return nil
	}

	return &Onceable{
		ExpiresAt: opts.expiresAt,
		Key:       opts.key,
		Fresh:     opts.fresh,
	}
}

type Mergeable struct {
	AppendKeys  []MergeKey
	PrependKeys []MergeKey

	Append  bool
	Prepend bool
}

// ToMergeable converts MergeOpts to Mergeable.
func ToMergeable(opts *MergeOpts) *Mergeable {
	if opts == nil {
		return nil
	}

	return &Mergeable{
		Append:      opts.append,
		Prepend:     opts.prepend,
		AppendKeys:  slices.Clone(opts.appendKeys),
		PrependKeys: slices.Clone(opts.prependKeys),
	}
}

type Scrollable struct {
	PreviousPage any
	NextPage     any
	CurrentPage  any
	PageName     string
	Path         string
}
