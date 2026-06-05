package inertiaalways_test

import (
	"context"
	"testing"

	"go.segfaultmedaddy.com/inertia/inertiaalways"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprotocol"
	"go.segfaultmedaddy.com/inertia/internal/inertiatest"
)

func TestAlwaysProp(t *testing.T) {
	t.Parallel()

	t.Run("should include value in props on full request", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(inertiaalways.New("auth", map[string]string{"user": "Roman"})).
			ExpectProp("auth", map[string]string{"user": "Roman"}).
			Run()
	})

	t.Run("should bypass partial filters when included in partial request", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name    string
			request inertiaprotocol.Request
		}{
			{
				name: "whitelist does not include it",
				request: inertiaprotocol.Request{
					URL:              "/users",
					PartialComponent: "TestComponent",
					PartialData:      []string{"other"},
				},
			},
			{
				name: "blacklist excludes it",
				request: inertiaprotocol.Request{
					URL:              "/users",
					PartialComponent: "TestComponent",
					PartialExcept:    []string{"auth"},
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				inertiatest.NewPropTestBuilder(t, tc.request).
					With(inertiaalways.New("auth", map[string]string{"user": "Roman"})).
					ExpectProp("auth", map[string]string{"user": "Roman"}).
					Run()
			})
		}
	})

	t.Run("should resolve value from lazy source when prop value is lazy", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(inertiaalways.NewLazy("auth", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
				return map[string]string{"user": "LazyRoman"}, nil
			}))).
			ExpectProp("auth", map[string]string{"user": "LazyRoman"}).
			Run()
	})
}
