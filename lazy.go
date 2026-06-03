package inertia

import (
	"context"

	"go.segfaultmedaddy.com/inertia/internal/inertiatest"
)

// TestLazyFunc is a lazy function wrapper that tracks how many times it has been called.
//
// It implements the Lazy interface and can be passed directly to prop constructors.
type TestLazyFunc = inertiatest.TestLazyFunc

// NewTestLazyFunc wraps the given function with call-count tracking for use in tests.
//
// The returned TestLazyFunc implements the Lazy interface and can be passed directly
// to prop constructors. Use ExpectCalledOnce(t) and ExpectNotCalled(t) on it to assert
// how many times the function was invoked during the render.
func NewTestLazyFunc(fn func(context.Context) (any, error)) *TestLazyFunc {
	return inertiatest.NewTestLazyFunc(fn)
}
