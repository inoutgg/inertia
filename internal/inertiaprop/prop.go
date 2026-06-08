package inertiaprop

import (
	"context"
	"fmt"
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

type Deferrable struct {
	Group  string
	Rescue bool
}

type Onceable struct {
	ExpiresAt *int64
	Key       string
	Fresh     bool
}

type Mergeable struct {
	AppendKeys  []string
	PrependKeys []string
	MatchOn     []string

	DeepMerge bool
	Append    bool
}

type Scrollable struct {
	PreviousPage any
	NextPage     any
	CurrentPage  any
	PageName     string
	Path         string
}
