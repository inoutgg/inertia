package inertiadeferred_test

import (
	"context"
	"errors"
	"testing"

	"go.segfaultmedaddy.com/inertia"
	"go.segfaultmedaddy.com/inertia/inertiadeferred"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprotocol"
	"go.segfaultmedaddy.com/inertia/internal/inertiatest"
)

func TestDeferredProp(t *testing.T) {
	t.Parallel()

	t.Run("should exclude value from props and add to deferredProps on full request", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(inertiadeferred.New("deferred", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
				return "Deferred Content", nil
			}))).
			ExpectNoProp("deferred").
			ExpectDeferredGroup("default", "deferred").
			Run(t.Context())
	})

	t.Run("should include value in props on partial request and omit deferredProps", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewTestBuilder(t, inertiaprotocol.Request{
			URL:              "/users",
			PartialComponent: "TestComponent",
			PartialData:      []string{"deferred"},
		}).
			With(inertiadeferred.New("deferred", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
				return "Deferred Content", nil
			}))).
			ExpectProp("deferred", "Deferred Content").
			ExpectNoDeferredProps().
			Run(t.Context())
	})

	t.Run("should group props in deferredProps when WithGroup is set", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(
				inertiadeferred.New("a", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
					return "a-value", nil
				}), inertiadeferred.WithGroup("group1")),
				inertiadeferred.New("b", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
					return "b-value", nil
				}), inertiadeferred.WithGroup("group1")),
				inertiadeferred.New("c", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
					return "c-value", nil
				}), inertiadeferred.WithGroup("group2")),
			).
			ExpectDeferredGroup("group1", "a", "b").
			ExpectDeferredGroup("group2", "c").
			Run(t.Context())
	})

	t.Run("should rescue failed prop and report it in rescuedProps when WithRescue is set", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewTestBuilder(t, inertiaprotocol.Request{
			URL:              "/users",
			PartialComponent: "TestComponent",
			PartialData:      []string{"rescued"},
		}).
			With(inertiadeferred.New("rescued", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
				return nil, errors.New("deferred error")
			}), inertiadeferred.WithRescue(true))).
			ExpectPropIsNil("rescued").
			ExpectRescuedProps("rescued").
			Run(t.Context())
	})

	t.Run("should populate mergeProps when WithMerge is set", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(inertiadeferred.New("merged", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
				return map[string]string{"key": "value"}, nil
			}), inertiadeferred.WithMerge(inertia.NewMergeOpts()))).
			ExpectNoProp("merged").
			ExpectDeferredGroup("default", "merged").
			ExpectMergeProps("merged").
			Run(t.Context())
	})

	t.Run("should populate both deferredProps and onceProps when WithOnce is set", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(inertiadeferred.New("locale", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
				return "en", nil
			}), inertiadeferred.WithOnce(inertia.NewOnceOpts().Key("locale_key")))).
			ExpectNoProp("locale").
			ExpectDeferredGroup("default", "locale").
			ExpectOnceProps("locale_key", "locale").
			Run(t.Context())
	})

	t.Run("should pass through nil value when prop returns nil", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewTestBuilder(t, inertiaprotocol.Request{
			URL:              "/users",
			PartialComponent: "TestComponent",
			PartialData:      []string{"nullable"},
		}).
			With(inertiadeferred.New("nullable", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
				return nil, nil //nolint:nilnil
			}))).
			ExpectPropIsNil("nullable").
			Run(t.Context())
	})
}
