package inertiaprop_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.segfaultmedaddy.com/inertia/inertiamerge"
	"go.segfaultmedaddy.com/inertia/inertiaonce"
	"go.segfaultmedaddy.com/inertia/inertiaprop"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprotocol"
	"go.segfaultmedaddy.com/inertia/internal/inertiatest"
)

func TestProp(t *testing.T) {
	t.Parallel()

	t.Run("should include value in props on full request", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(
				inertiaprop.New("name", "Roman"),
				inertiaprop.New("age", 30),
			).
			ExpectProp("name", "Roman").
			ExpectProp("age", 30).
			Run()
	})

	t.Run("should respect partial whitelist when partial component matches", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
			URL:              "/users",
			PartialComponent: "TestComponent",
			PartialData:      []string{"name"},
		}).
			With(
				inertiaprop.New("name", "Roman"),
				inertiaprop.New("email", "test@example.com"),
			).
			ExpectProp("name", "Roman").
			ExpectNoProp("email").
			Run()
	})

	t.Run("should respect partial blacklist when partial component matches", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
			URL:              "/users",
			PartialComponent: "TestComponent",
			PartialExcept:    []string{"email"},
		}).
			With(
				inertiaprop.New("name", "Roman"),
				inertiaprop.New("email", "test@example.com"),
			).
			ExpectProp("name", "Roman").
			ExpectNoProp("email").
			Run()
	})

	t.Run("should populate mergeProps with NewAppendRoot for root-only append", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(
				inertiaprop.New("normal", "value"),
				inertiaprop.New("mergeable", map[string]string{"key": "val"},
					inertiaprop.WithMerge(inertiamerge.NewAppendRoot()),
				),
			).
			ExpectMergeProps("mergeable").
			ExpectProp("normal", "value").
			Run()
	})

	t.Run("should populate mergeProps with NewAppendRoot for root-only append", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(
				inertiaprop.New("posts", []string{"one"},
					inertiaprop.WithMerge(inertiamerge.NewAppendRoot()),
				),
			).
			ExpectMergeProps("posts").
			ExpectNoPrependProps().
			ExpectNoDeepMergeProps().
			Run()
	})

	t.Run("should populate prependProps with NewPrependRoot for root-only prepend", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(
				inertiaprop.New("notifications", []string{"one"},
					inertiaprop.WithMerge(inertiamerge.NewPrependRoot()),
				),
			).
			ExpectPrependProps("notifications").
			ExpectNoMergeProps().
			ExpectNoDeepMergeProps().
			Run()
	})

	t.Run("should populate onceProps when WithOnce is set", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(
				inertiaprop.New("regular", "value"),
				inertiaprop.New("onceable", "secret",
					inertiaprop.WithOnce(inertiaonce.NewOnceOpts().Key("once_key")),
				),
			).
			ExpectOnceProps("once_key", "onceable").
			ExpectProp("regular", "value").
			Run()
	})

	t.Run("should emit merge metadata for path and deep merge configs", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(
				inertiaprop.New(
					"posts",
					[]string{"one"},
					inertiaprop.WithMerge(
						inertiamerge.NewAppendRoot(),
					),
				),
				inertiaprop.New(
					"notifications",
					[]string{"one"},
					inertiaprop.WithMerge(
						inertiamerge.NewPrependRoot(),
					),
				),
				inertiaprop.New("conversation", map[string]any{"messages": []string{"one"}},
					inertiaprop.WithMerge(
						inertiamerge.NewPaths().
							Append(inertiamerge.At("messages").On("id")),
					),
				),
				inertiaprop.New("settings", map[string]any{"theme": "dark"},
					inertiaprop.WithMerge(
						inertiamerge.NewDeepMerge("key"),
					),
				),
			).
			ExpectMergeProps("posts", "conversation.messages").
			ExpectPrependProps("notifications").
			ExpectDeepMergeProps("settings").
			ExpectMatchPropsOn("conversation.messages.id", "settings.key").
			Run()
	})

	t.Run(
		"should suppress mergeProps and matchPropsOn for reset keys while still returning the value",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:        "/users",
				ResetProps: []string{"posts"},
			}).
				With(
					inertiaprop.New(
						"posts",
						[]string{"fresh"},
						inertiaprop.WithMerge(
							inertiamerge.NewAppendRoot(),
						),
					),
					inertiaprop.New("name", "Roman"),
				).
				ExpectProp("posts", []string{"fresh"}).
				ExpectProp("name", "Roman").
				ExpectNoMergeProps().
				ExpectNoPrependProps().
				Run()

			assert.Empty(t, page.MatchPropsOn)
		},
	)
}
