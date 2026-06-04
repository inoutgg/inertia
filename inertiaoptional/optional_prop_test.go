package inertiaoptional_test

import (
	"context"
	"errors"
	"testing"

	"go.segfaultmedaddy.com/inertia/inertiaoptional"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprotocol"
	"go.segfaultmedaddy.com/inertia/internal/inertiatest"
)

func TestOptionalProp(t *testing.T) {
	t.Parallel()

	const expensiveValue = "computed"

	t.Run("should exclude value from props on full request", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(inertiaoptional.New("expensive", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
				return expensiveValue, nil
			}))).
			ExpectNoProp("expensive").
			Run()
	})

	t.Run("should compute value on demand when prop is requested by client", func(t *testing.T) {
		t.Parallel()

		lazy := inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
			return expensiveValue, nil
		})

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
			URL:              "/users",
			PartialComponent: "TestComponent",
			PartialData:      []string{"expensive"},
		}).
			With(inertiaoptional.New("expensive", lazy)).
			ExpectProp("expensive", expensiveValue).
			Run()

		lazy.ExpectCalledOnce(t)
	})

	t.Run("should exclude value from props when partial request does not whitelist it", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
			URL:              "/users",
			PartialComponent: "TestComponent",
			PartialData:      []string{"other"},
		}).
			With(inertiaoptional.New("expensive", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
				return expensiveValue, nil
			}))).
			ExpectNoProp("expensive").
			Run()
	})

	t.Run("should compute multiple props in parallel when prop is marked concurrent", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
			URL:              "/users",
			PartialComponent: "TestComponent",
			PartialData:      []string{"a", "b"},
		}).
			With(
				inertiaoptional.New("a", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
					return "val-a", nil
				}), inertiaoptional.WithConcurrent),
				inertiaoptional.New("b", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
					return "val-b", nil
				}), inertiaoptional.WithConcurrent),
			).
			ExpectProp("a", "val-a").
			ExpectProp("b", "val-b").
			Run()
	})

	t.Run("should skip value computation on initial page load", func(t *testing.T) {
		t.Parallel()

		lazy := inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
			return "lazy-value", nil
		})

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(inertiaoptional.New("lazy", lazy)).
			ExpectNoProp("lazy").
			Run()

		lazy.ExpectNotCalled(t)
	})

	t.Run("should return error when value resolution fails", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
			URL:              "/users",
			PartialComponent: "TestComponent",
			PartialData:      []string{"failing"},
		}).
			With(inertiaoptional.New("failing", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
				return nil, errLazyFailure
			}))).
			ExpectError(errLazyFailure).
			Run()
	})

	t.Run("should pass through false value when prop returns false", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
			URL:              "/users",
			PartialComponent: "TestComponent",
			PartialData:      []string{"flag"},
		}).
			With(inertiaoptional.New("flag", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
				return false, nil
			}))).
			ExpectProp("flag", false).
			Run()
	})

	t.Run("should pass through nil value when prop returns nil", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
			URL:              "/users",
			PartialComponent: "TestComponent",
			PartialData:      []string{"nullable"},
		}).
			With(inertiaoptional.New("nullable", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
				return nil, nil //nolint:nilnil
			}))).
			ExpectPropIsNil("nullable").
			Run()
	})
}

var errLazyFailure = errors.New("lazy error")
