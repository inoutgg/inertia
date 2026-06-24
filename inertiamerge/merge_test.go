package inertiamerge_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.segfaultmedaddy.com/inertia/inertiamerge"
	"go.segfaultmedaddy.com/inertia/inertiaprop"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprotocol"
	"go.segfaultmedaddy.com/inertia/internal/inertiatest"
)

func TestMerge(t *testing.T) {
	t.Parallel()

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

	t.Run("should populate mergeProps with NewAppendRoot for root-only append (single prop)", func(t *testing.T) {
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
						inertiamerge.NewPathsMerge().
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

	t.Run("it should suppress root-level merge entry when path-based append keys are present", func(t *testing.T) {
		t.Parallel()

		page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(
				inertiaprop.New("conversation", map[string]any{"messages": []string{"one"}},
					inertiaprop.WithMerge(
						inertiamerge.NewPathsMerge().Append(inertiamerge.At("messages")),
					),
				),
			).
			ExpectMergeProps("conversation.messages").
			ExpectNoPrependProps().
			ExpectNoDeepMergeProps().
			Run()

		assert.NotContains(t, page.MergeProps, "conversation",
			"root-level entry must be suppressed by path-based keys")
		assert.Empty(t, page.MatchPropsOn)
	})

	t.Run("it should populate deepMergeProps and matchPropsOn when NewDeepMerge is set", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(
				inertiaprop.New("settings", map[string]any{"theme": "dark"},
					inertiaprop.WithMerge(inertiamerge.NewDeepMerge("key")),
				),
			).
			ExpectDeepMergeProps("settings").
			ExpectMatchPropsOn("settings.key").
			ExpectNoMergeProps().
			ExpectNoPrependProps().
			Run()
	})

	t.Run("it should qualify matchOn keys with the prop root when matchOn paths are non-empty", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(
				inertiaprop.New("conversation", map[string]any{"messages": []string{"one"}},
					inertiaprop.WithMerge(
						inertiamerge.NewPathsMerge().
							Append(inertiamerge.At("messages").On("id")),
					),
				),
			).
			ExpectMergeProps("conversation.messages").
			ExpectMatchPropsOn("conversation.messages.id").
			Run()
	})

	t.Run("it should ignore empty append, prepend and matchOn path segments", func(t *testing.T) {
		t.Parallel()

		page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(
				inertiaprop.New("posts", map[string]any{"data": []string{"one"}, "items": []string{"two"}},
					inertiaprop.WithMerge(
						inertiamerge.NewPathsMerge().
							Append(
								inertiamerge.At("data"),
								inertiamerge.At(""),
								inertiamerge.At("items"),
							),
					),
				),
			).
			ExpectMergeProps("posts.data", "posts.items").
			ExpectNoPrependProps().
			ExpectNoDeepMergeProps().
			Run()

		assert.Empty(t, page.MatchPropsOn, "empty matchOn segments must not produce entries")
	})

	t.Run(
		"it should suppress deepMergeProps and matchPropsOn for reset keys when DeepMerge is set",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:        "/users",
				ResetProps: []string{"settings"},
			}).
				With(
					inertiaprop.New("settings", map[string]any{"theme": "dark"},
						inertiaprop.WithMerge(inertiamerge.NewDeepMerge("key")),
					),
				).
				ExpectProp("settings", map[string]any{"theme": "dark"}).
				ExpectNoDeepMergeProps().
				Run()

			assert.Empty(t, page.MatchPropsOn)
			assert.Empty(t, page.MergeProps)
		},
	)
}
