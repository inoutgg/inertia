package inertiatest

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLazyFunc wraps a lazy function and tracks how many times it has been called.
//
// It implements the inertia.Lazy interface and can be passed directly to prop constructors.
//
// The wrapped function is not invoked at construction; it is invoked only when the
// prop's Value method is called during render.
type TestLazyFunc struct {
	fn    func(context.Context) (any, error)
	calls atomic.Int64
}

// NewTestLazyFunc creates a new TestLazyFunc from the given function.
func NewTestLazyFunc(fn func(context.Context) (any, error)) *TestLazyFunc {
	return &TestLazyFunc{fn: fn} //nolint:exhaustruct
}

// Value invokes the wrapped function and increments the call counter.
//
// It satisfies the inertia.Lazy interface.
func (l *TestLazyFunc) Value(ctx context.Context) (any, error) {
	l.calls.Add(1)

	return l.fn(ctx)
}

// Calls returns the number of times the wrapped function has been invoked.
func (l *TestLazyFunc) Calls() int64 {
	return l.calls.Load()
}

// ExpectCalledOnce asserts that the wrapped function was called exactly once.
func (l *TestLazyFunc) ExpectCalledOnce(t *testing.T) {
	t.Helper()

	assert.Equal(t, int64(1), l.calls.Load(), "expected lazy function to be called once")
}

// ExpectNotCalled asserts that the wrapped function was never called.
func (l *TestLazyFunc) ExpectNotCalled(t *testing.T) {
	t.Helper()

	assert.Zero(t, l.calls.Load(), "expected lazy function to not be called")
}
