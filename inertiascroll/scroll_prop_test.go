package inertiascroll_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.segfaultmedaddy.com/inertia"
	"go.segfaultmedaddy.com/inertia/inertiascroll"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprotocol"
	"go.segfaultmedaddy.com/inertia/internal/inertiatest"
)

func TestScrollProp(t *testing.T) {
	t.Parallel()

	t.Run("should populate scrollProps with pagination metadata on full request", func(t *testing.T) {
		t.Parallel()

		nextPage := 2
		currentPage := 1

		page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
			URL:               "/users",
			ScrollMergeIntent: inertiaprop.ScrollMergeIntentAppend,
		}).
			With(inertiascroll.New(
				"users",
				map[string]any{"data": []string{"one"}},
				inertiascroll.WithPagination(nil, &nextPage, &currentPage),
				inertiascroll.WithPageName("page"),
			)).
			ExpectProp("users", map[string]any{"data": []string{"one"}}).
			Run()

		scrollProps, ok := page.ScrollProps["users"]
		require.True(t, ok)
		assert.Equal(t, "page", scrollProps.PageName)
		assert.Nil(t, scrollProps.PreviousPage)

		next, ok := scrollProps.NextPage.(*int)
		require.True(t, ok)
		assert.Equal(t, nextPage, *next)

		current, ok := scrollProps.CurrentPage.(*int)
		require.True(t, ok)
		assert.Equal(t, currentPage, *current)
	})

	t.Run("should add to mergeProps when scroll merge intent is append", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
			URL:               "/users",
			ScrollMergeIntent: inertiaprop.ScrollMergeIntentAppend,
		}).
			With(inertiascroll.New("users", map[string]any{"data": []string{"one"}})).
			ExpectMergeProps("users.data").
			ExpectNoPrependProps().
			Run()
	})

	t.Run("should add to prependProps when scroll merge intent is prepend", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
			URL:               "/users",
			ScrollMergeIntent: inertiaprop.ScrollMergeIntentPrepend,
		}).
			With(inertiascroll.New("users", map[string]any{"data": []string{"one"}})).
			ExpectPrependProps("users.data").
			ExpectNoMergeProps().
			Run()
	})

	t.Run("should return error when scroll merge intent is invalid", func(t *testing.T) {
		t.Parallel()

		_, err := inertiaprotocol.Render(t.Context(), inertiaprotocol.Request{
			URL:               "/users",
			ScrollMergeIntent: "sideways",
		}, inertiaprotocol.Context{
			Component: inertiatest.DefaultComponent,
			Version:   inertiatest.DefaultVersion,
			Props: []inertia.Prop{
				inertiascroll.New("users", map[string]any{"data": []string{"one"}}),
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid scroll merge intent")
	})

	t.Run("should resolve value from lazy source when prop value is lazy", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
			URL:               "/users",
			ScrollMergeIntent: inertiaprop.ScrollMergeIntentAppend,
		}).
			With(inertiascroll.New(
				"users",
				inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
					return map[string]any{"data": []string{"lazy-one"}}, nil
				}),
			)).
			ExpectProp("users", map[string]any{"data": []string{"lazy-one"}}).
			ExpectMergeProps("users.data").
			Run()
	})

	t.Run("should use custom wrapper path when WithWrapper is set", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
			URL:               "/users",
			ScrollMergeIntent: inertiaprop.ScrollMergeIntentAppend,
		}).
			With(inertiascroll.New(
				"users",
				map[string]any{"items": []string{"one"}, "meta": map[string]int{"total": 1}},
				inertiascroll.WithWrapper("items"),
			)).
			ExpectProp("users", map[string]any{
				"items": []string{"one"},
				"meta":  map[string]int{"total": 1},
			}).
			ExpectMergeProps("users.items").
			Run()
	})
}
