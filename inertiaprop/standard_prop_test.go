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

	t.Run("should emit merge match-on metadata for root and path configs", func(t *testing.T) {
		t.Parallel()

		inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
			With(
				inertiaprop.New(
					"posts",
					[]string{"one"},
					inertiaprop.WithMerge(
						inertiamerge.NewAppendRoot("id"),
					),
				),
				inertiaprop.New(
					"notifications",
					[]string{"one"},
					inertiaprop.WithMerge(
						inertiamerge.NewPrependRoot("uuid"),
					),
				),
				inertiaprop.New("conversation", map[string]any{"messages": []string{"one"}},
					inertiaprop.WithMerge(
						inertiamerge.NewPaths().
							Append(inertiamerge.At("messages").On("id")),
					),
				),
			).
			ExpectMergeProps("posts", "conversation.messages").
			ExpectPrependProps("notifications").
			ExpectMatchPropsOn("posts.id", "notifications.uuid", "conversation.messages.id").
			Run()
	})

	t.Run(
		"should treat matchOn-only as root-level append with NewAppendRoot",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(
					inertiaprop.New(
						"posts",
						[]string{"one"},
						inertiaprop.WithMerge(
							inertiamerge.NewAppendRoot("id"),
						),
					),
				).
				ExpectMergeProps("posts").
				ExpectNoPrependProps().
				ExpectMatchPropsOn("posts.id").
				Run()
		},
	)

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
							inertiamerge.NewAppendRoot("id"),
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
