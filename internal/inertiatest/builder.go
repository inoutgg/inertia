package inertiatest

import (
	"testing"

	"github.com/alitto/pond/v2"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprotocol"
)

// DefaultComponent is the default component name used in test renders.
const DefaultComponent = "TestComponent"

// DefaultVersion is the default version used in test renders.
const DefaultVersion = "1.0.0"

// PropAssert is a function that asserts expectations on a rendered page.
type PropAssert func(t *testing.T, page *inertiaprotocol.Page)

// ContextOption is a function modifying the inertia protocol request context
// during PropTestBuilder run.
type ContextOption func(*inertiaprotocol.Context)

// PropTestBuilder is a fluent test builder that encapsulates rendering, assertion, and snapshotting.
//
// Create a builder with New, add props with With, add expectations with Expect* methods,
// then call Run to execute the test.
type PropTestBuilder struct {
	expectErr  error
	t          *testing.T
	request    inertiaprotocol.Request
	props      []inertiaprop.Prop
	assertions []PropAssert
}

// NewPropTestBuilder creates a new test builder for the given request.
func NewPropTestBuilder(t *testing.T, request inertiaprotocol.Request) *PropTestBuilder {
	t.Helper()

	return &PropTestBuilder{ //nolint:exhaustruct
		t:       t,
		request: request,
	}
}

// With adds props to the test builder.
func (b *PropTestBuilder) With(props ...inertiaprop.Prop) *PropTestBuilder {
	b.props = append(b.props, props...)

	return b
}

// ExpectError sets an expected error from the render call.
// When set, Run will assert that rendering returns this error and skip page assertions.
func (b *PropTestBuilder) ExpectError(err error) *PropTestBuilder {
	b.expectErr = err

	return b
}

// ExpectProp asserts that the prop with the given key has the expected value.
func (b *PropTestBuilder) ExpectProp(key string, expected any) *PropTestBuilder {
	b.assertions = append(b.assertions, func(t *testing.T, page *inertiaprotocol.Page) {
		t.Helper()

		assert.Equal(t, expected, page.Props[key])
	})

	return b
}

// ExpectNoProp asserts that the given keys are not present in the page props.
func (b *PropTestBuilder) ExpectNoProp(keys ...string) *PropTestBuilder {
	b.assertions = append(b.assertions, func(t *testing.T, page *inertiaprotocol.Page) {
		t.Helper()

		for _, k := range keys {
			assert.NotContains(t, page.Props, k, "expected prop %q to be absent", k)
		}
	})

	return b
}

// ExpectPropIsNil asserts that the prop with the given key is nil in the page props.
func (b *PropTestBuilder) ExpectPropIsNil(key string) *PropTestBuilder {
	b.assertions = append(b.assertions, func(t *testing.T, page *inertiaprotocol.Page) {
		t.Helper()

		assert.Nil(t, page.Props[key], "expected prop %q to be nil", key)
	})

	return b
}

// ExpectMergeProps asserts that the given keys are present in page.MergeProps.
func (b *PropTestBuilder) ExpectMergeProps(keys ...string) *PropTestBuilder {
	b.assertions = append(b.assertions, func(t *testing.T, page *inertiaprotocol.Page) {
		t.Helper()

		for _, k := range keys {
			assert.Contains(t, page.MergeProps, k, "expected %q in mergeProps", k)
		}
	})

	return b
}

// ExpectNoMergeProps asserts that page.MergeProps is empty.
func (b *PropTestBuilder) ExpectNoMergeProps() *PropTestBuilder {
	b.assertions = append(b.assertions, func(t *testing.T, page *inertiaprotocol.Page) {
		t.Helper()

		assert.Empty(t, page.MergeProps)
	})

	return b
}

// ExpectPrependProps asserts that the given keys are present in page.PrependProps.
func (b *PropTestBuilder) ExpectPrependProps(keys ...string) *PropTestBuilder {
	b.assertions = append(b.assertions, func(t *testing.T, page *inertiaprotocol.Page) {
		t.Helper()

		for _, k := range keys {
			assert.Contains(t, page.PrependProps, k, "expected %q in prependProps", k)
		}
	})

	return b
}

// ExpectDeepMergeProps asserts that the given keys are present in page.DeepMergeProps.
func (b *PropTestBuilder) ExpectDeepMergeProps(keys ...string) *PropTestBuilder {
	b.assertions = append(b.assertions, func(t *testing.T, page *inertiaprotocol.Page) {
		t.Helper()

		for _, k := range keys {
			assert.Contains(t, page.DeepMergeProps, k, "expected %q in deepMergeProps", k)
		}
	})

	return b
}

// ExpectNoDeepMergeProps asserts that page.DeepMergeProps is empty.
func (b *PropTestBuilder) ExpectNoDeepMergeProps() *PropTestBuilder {
	b.assertions = append(b.assertions, func(t *testing.T, page *inertiaprotocol.Page) {
		t.Helper()

		assert.Empty(t, page.DeepMergeProps)
	})

	return b
}

// ExpectNoPrependProps asserts that page.PrependProps is empty.
func (b *PropTestBuilder) ExpectNoPrependProps() *PropTestBuilder {
	b.assertions = append(b.assertions, func(t *testing.T, page *inertiaprotocol.Page) {
		t.Helper()

		assert.Empty(t, page.PrependProps)
	})

	return b
}

// ExpectOnceProps asserts that the given key is present in page.OnceProps with the expected prop name.
func (b *PropTestBuilder) ExpectOnceProps(key, propName string) *PropTestBuilder {
	b.assertions = append(b.assertions, func(t *testing.T, page *inertiaprotocol.Page) {
		t.Helper()

		assert.Contains(t, page.OnceProps, key, "expected %q in onceProps", key)
		assert.Equal(t, propName, page.OnceProps[key].Prop)
	})

	return b
}

// ExpectMatchPropsOn asserts that page.MatchPropsOn contains exactly the given keys.
func (b *PropTestBuilder) ExpectMatchPropsOn(keys ...string) *PropTestBuilder {
	b.assertions = append(b.assertions, func(t *testing.T, page *inertiaprotocol.Page) {
		t.Helper()

		assert.ElementsMatch(t, keys, page.MatchPropsOn)
	})

	return b
}

// ExpectNoDeferredProps asserts that page.DeferredProps is nil.
func (b *PropTestBuilder) ExpectNoDeferredProps() *PropTestBuilder {
	b.assertions = append(b.assertions, func(t *testing.T, page *inertiaprotocol.Page) {
		t.Helper()

		assert.Nil(t, page.DeferredProps)
	})

	return b
}

// ExpectDeferredGroup asserts that the given keys are present in the specified group of page.DeferredProps.
func (b *PropTestBuilder) ExpectDeferredGroup(group string, keys ...string) *PropTestBuilder {
	b.assertions = append(b.assertions, func(t *testing.T, page *inertiaprotocol.Page) {
		t.Helper()

		for _, k := range keys {
			assert.Contains(t, page.DeferredProps[group], k,
				"expected %q in deferredProps[%q]", k, group)
		}
	})

	return b
}

// ExpectRescuedProps asserts that the given keys are present in page.RescuedProps.
func (b *PropTestBuilder) ExpectRescuedProps(keys ...string) *PropTestBuilder {
	b.assertions = append(b.assertions, func(t *testing.T, page *inertiaprotocol.Page) {
		t.Helper()

		for _, k := range keys {
			assert.Contains(t, page.RescuedProps, k, "expected %q in rescuedProps", k)
		}
	})

	return b
}

// Run executes the test: renders the page, runs all expectations, and creates a snapshot.
//
// If ExpectError was called, Run asserts that rendering returns the expected error,
// skips page assertions and snapshotting, and returns nil.
//
// Otherwise, Run returns the rendered page so callers can perform additional assertions.
func (b *PropTestBuilder) Run(opts ...ContextOption) *inertiaprotocol.Page {
	b.t.Helper()

	ctx := inertiaprotocol.Context{ //nolint:exhaustruct
		Component:  DefaultComponent,
		Version:    DefaultVersion,
		Props:      b.props,
		ResultPool: pond.NewResultPool[inertiaprotocol.Result](0),
	}
	for _, opt := range opts {
		opt(&ctx)
	}

	page, err := inertiaprotocol.Render(b.t.Context(), b.request, ctx)
	if b.expectErr != nil {
		require.Error(b.t, err)
		assert.ErrorIs(b.t, err, b.expectErr)

		return nil
	}

	require.NoError(b.t, err)

	for _, assertFn := range b.assertions {
		assertFn(b.t, page)
	}

	snaps.MatchJSON(b.t, page)

	return page
}
