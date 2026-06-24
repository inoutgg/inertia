package inertiaprotocol_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.segfaultmedaddy.com/inertia/inertiaalways"
	"go.segfaultmedaddy.com/inertia/inertiadeferred"
	"go.segfaultmedaddy.com/inertia/inertiamerge"
	"go.segfaultmedaddy.com/inertia/inertiaonce"
	"go.segfaultmedaddy.com/inertia/inertiaoptional"
	"go.segfaultmedaddy.com/inertia/inertiaprop"
	"go.segfaultmedaddy.com/inertia/inertiascroll"
	inertiainternalprop "go.segfaultmedaddy.com/inertia/internal/inertiaprop"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprotocol"
	"go.segfaultmedaddy.com/inertia/internal/inertiatest"
)

var (
	errRenderSentinelA = errors.New("sentinel failure a")
	errRenderSentinelB = errors.New("sentinel failure b")
)

func TestRenderer_PageMetadata(t *testing.T) {
	t.Parallel()

	t.Run("it should populate component, url and version from the context and request",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users/123"}).
				With(inertiaprop.New("name", "Roman")).
				ExpectProp("name", "Roman").
				Run()

			assert.Equal(t, inertiatest.DefaultComponent, page.Component)
			assert.Equal(t, "/users/123", page.URL)
			assert.Equal(t, inertiatest.DefaultVersion, page.Version)
		},
	)

	t.Run("it should default preserveFragment, clearHistory and encryptHistory to false when unset",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(inertiaprop.New("name", "Roman")).
				Run()

			assert.False(t, page.PreserveFragment)
			assert.False(t, page.ClearHistory)
			assert.False(t, page.EncryptHistory)
		},
	)

	t.Run("it should propagate preserveFragment, clearHistory and encryptHistory when set on the context",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(inertiaprop.New("name", "Roman")).
				Run(
					inertiatest.WithPreserveFragment(true),
					inertiatest.WithClearHistory(true),
					inertiatest.WithEncryptHistory(true),
				)

			assert.True(t, page.PreserveFragment)
			assert.True(t, page.ClearHistory)
			assert.True(t, page.EncryptHistory)
		},
	)
}

func TestRenderer_SharedProps(t *testing.T) {
	t.Parallel()

	t.Run("it should list shared prop keys in sharedProps when shared props are provided",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(inertiaprop.New("name", "Roman")).
				Run(inertiatest.WithSharedProps(
					inertiaprop.New("auth", map[string]string{"user": "Roman"}),
					inertiaprop.New("errors", map[string]string{"name": "required"}),
				))

			assert.ElementsMatch(t,
				[]string{"auth", "errors"},
				page.SharedProps,
			)
		},
	)

	t.Run("it should leave sharedProps empty when no shared props are provided",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(inertiaprop.New("name", "Roman")).
				Run()

			assert.Empty(t, page.SharedProps)
		},
	)

	t.Run("it should not resolve shared prop values into Page props when they are only declared as shared",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(inertiaprop.New("name", "Roman")).
				Run(inertiatest.WithSharedProps(
					inertiaprop.New("auth", map[string]string{"user": "Roman"}),
				))

			assert.NotContains(t, page.Props, "auth")
			assert.Contains(t, page.SharedProps, "auth")
		},
	)

	t.Run("it should keep shared props independent of partial whitelist and blacklist",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: inertiatest.DefaultComponent,
				PartialData:      []string{"name"},
			}).
				With(inertiaprop.New("name", "Roman")).
				Run(inertiatest.WithSharedProps(
					inertiaprop.New("auth", map[string]string{"user": "Roman"}),
				))

			assert.Contains(t, page.Props, "name")
			assert.NotContains(t, page.Props, "auth")
			assert.Contains(t, page.SharedProps, "auth")
		},
	)
}

func TestRenderer_PartialComponentMismatch(t *testing.T) {
	t.Parallel()

	t.Run("it should behave like a full render when partialComponent does not match the rendered component",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: "OtherComponent",
				PartialData:      []string{"name"},
			}).
				With(
					inertiaprop.New("name", "Roman"),
					inertiaprop.New("email", "test@example.com"),
					inertiadeferred.New("deferred", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return "deferred-value", nil
					})),
				).
				ExpectProp("name", "Roman").
				ExpectProp("email", "test@example.com").
				ExpectNoProp("deferred").
				Run()

			assert.NotNil(
				t,
				page.DeferredProps,
				"deferredProps must still be emitted when partialComponent does not match",
			)
		},
	)

	t.Run("it should ignore partialData whitelist when partialComponent does not match",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: "OtherComponent",
				PartialData:      []string{"name"},
			}).
				With(
					inertiaprop.New("name", "Roman"),
					inertiaprop.New("email", "test@example.com"),
				).
				ExpectProp("name", "Roman").
				ExpectProp("email", "test@example.com").
				Run()

			assert.Len(t, page.Props, 2)
		},
	)

	t.Run("it should ignore partialExcept blacklist when partialComponent does not match",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: "OtherComponent",
				PartialExcept:    []string{"email"},
			}).
				With(
					inertiaprop.New("name", "Roman"),
					inertiaprop.New("email", "test@example.com"),
				).
				ExpectProp("name", "Roman").
				ExpectProp("email", "test@example.com").
				Run()
		},
	)
}

func TestRenderer_PartialResolution(t *testing.T) {
	t.Parallel()

	t.Run("it should exclude blacklisted props when partialExcept is non-empty",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: inertiatest.DefaultComponent,
				PartialExcept:    []string{"email"},
			}).
				With(
					inertiaprop.New("name", "Roman"),
					inertiaprop.New("email", "test@example.com"),
				).
				ExpectProp("name", "Roman").
				ExpectNoProp("email").
				Run()
		},
	)

	t.Run("it should combine whitelist and blacklist when both are present",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: inertiatest.DefaultComponent,
				PartialData:      []string{"name", "email"},
				PartialExcept:    []string{"email"},
			}).
				With(
					inertiaprop.New("name", "Roman"),
					inertiaprop.New("email", "test@example.com"),
					inertiaprop.New("age", 30),
				).
				ExpectProp("name", "Roman").
				ExpectNoProp("email").
				ExpectNoProp("age").
				Run()
		},
	)

	t.Run("it should rescue multiple failed props aggregating each key into rescuedProps",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: inertiatest.DefaultComponent,
				PartialData:      []string{"a", "b"},
			}).
				With(
					inertiadeferred.New("a", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return nil, errRenderSentinelA
					}), inertiadeferred.WithRescue(true)),
					inertiadeferred.New("b", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return nil, errRenderSentinelB
					}), inertiadeferred.WithRescue(true)),
				).
				ExpectNoProp("a", "b").
				ExpectRescuedProps("a", "b").
				Run()
		},
	)

	t.Run("it should return a wrapped error when a non-rescued prop fails resolution on partial request",
		func(t *testing.T) {
			t.Parallel()

			page, err := inertiatest.TestRender(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: inertiatest.DefaultComponent,
				PartialData:      []string{"failing"},
			}, inertiatest.WithProps(
				inertiaoptional.New(
					"failing",
					inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return nil, errRenderSentinelA
					}),
				),
			),
			)

			require.Error(t, err)
			require.ErrorIs(t, err, errRenderSentinelA)
			require.ErrorContains(t, err, "failed to resolve prop failing")
			assert.Nil(t, page)
		},
	)
}

func TestRenderer_ConcurrentResolution(t *testing.T) {
	t.Parallel()

	t.Run("it should aggregate rescued keys and not fail when concurrent props fail with RescueError",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: inertiatest.DefaultComponent,
				PartialData:      []string{"a", "b"},
			}).
				With(
					inertiadeferred.New("a", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return nil, errRenderSentinelA
					}), inertiadeferred.WithRescue(true), inertiadeferred.WithConcurrent),
					inertiadeferred.New("b", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return nil, errRenderSentinelB
					}), inertiadeferred.WithRescue(true), inertiadeferred.WithConcurrent),
				).
				ExpectNoProp("a", "b").
				ExpectRescuedProps("a", "b").
				Run()
		},
	)

	t.Run("it should return a wrapped error when a concurrent prop fails with a non-rescue error",
		func(t *testing.T) {
			t.Parallel()

			page, err := inertiatest.TestRender(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: inertiatest.DefaultComponent,
				PartialData:      []string{"a", "b"},
			}, inertiatest.WithProps(
				inertiaoptional.New(
					"a",
					inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return "a-value", nil
					}),
					inertiaoptional.WithConcurrent,
				),
				inertiaoptional.New(
					"b",
					inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return nil, errRenderSentinelB
					}),
					inertiaoptional.WithConcurrent,
				),
			),
			)

			require.Error(t, err)
			require.ErrorIs(t, err, errRenderSentinelB)
			require.ErrorContains(t, err, "failed to resolve prop b")
			assert.Nil(t, page)
		},
	)

	t.Run("it should mix concurrent and sequential props in the same partial request preserving all values",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: inertiatest.DefaultComponent,
				PartialData:      []string{"seq", "con1", "con2"},
			}).
				With(
					inertiaprop.New("seq", "seq-value"),
					inertiaoptional.New("con1", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return "con1-value", nil
					}), inertiaoptional.WithConcurrent),
					inertiaoptional.New("con2", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return "con2-value", nil
					}), inertiaoptional.WithConcurrent),
				).
				ExpectProp("seq", "seq-value").
				ExpectProp("con1", "con1-value").
				ExpectProp("con2", "con2-value").
				Run()
		},
	)
}

func TestRenderer_FirstLoadIgnorables(t *testing.T) {
	t.Parallel()

	t.Run("it should skip optional and deferred props on full request without invoking their lazy functions",
		func(t *testing.T) {
			t.Parallel()

			optionalLazy := inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
				return "optional-value", nil
			})
			deferredLazy := inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
				return "deferred-value", nil
			})

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(
					inertiaoptional.New("opt", optionalLazy),
					inertiadeferred.New("def", deferredLazy),
				).
				ExpectNoProp("opt").
				ExpectNoProp("def").
				Run()

			optionalLazy.ExpectNotCalled(t)
			deferredLazy.ExpectNotCalled(t)
		},
	)

	t.Run("it should still include always and standard props alongside ignorable props on full request",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(
					inertiaalways.New("auth", map[string]string{"user": "Roman"}),
					inertiaprop.New("name", "Roman"),
					inertiaoptional.New("opt", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return "opt-value", nil
					})),
					inertiadeferred.New("def", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return "def-value", nil
					})),
				).
				ExpectProp("auth", map[string]string{"user": "Roman"}).
				ExpectProp("name", "Roman").
				ExpectNoProp("opt").
				ExpectNoProp("def").
				Run()
		},
	)
}

func TestRenderer_PartialOnceInteraction(t *testing.T) {
	t.Parallel()

	t.Run(
		"it should keep once prop on partial request when its key is whitelisted despite exceptOnceProps listing it",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: inertiatest.DefaultComponent,
				PartialData:      []string{"locale"},
				ExceptOnceProps:  []string{"locale_key"},
			}).
				With(
					inertiaprop.NewLazy("locale", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return "lazy-locale", nil
					}), inertiaprop.WithOnce(inertiaonce.NewOnceOpts().Key("locale_key"))),
				).
				ExpectProp("locale", "lazy-locale").
				ExpectOnceProps("locale_key", "locale").
				Run()
		},
	)
}

func TestRenderer_ResetHeader(t *testing.T) {
	t.Parallel()

	t.Run("it should not affect non-reset props when some props are listed in resetProps",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:        "/users",
				ResetProps: []string{"reset_one"},
			}).
				With(
					inertiaprop.New("reset_one", []string{"a"},
						inertiaprop.WithMerge(inertiamerge.NewAppendRoot()),
					),
					inertiaprop.New("keep", []string{"b"},
						inertiaprop.WithMerge(inertiamerge.NewAppendRoot()),
					),
				).
				ExpectProp("reset_one", []string{"a"}).
				ExpectProp("keep", []string{"b"}).
				ExpectMergeProps("keep").
				ExpectNoPrependProps().
				Run()
		},
	)
}

func TestRenderer_MixedPropTypes(t *testing.T) {
	t.Parallel()

	t.Run(
		"it should produce all metadata fields together when standard, deferred, optional, once, merge and scroll props are combined on a full request",
		func(t *testing.T) {
			t.Parallel()

			next, current := 2, 1

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:               "/users",
				ScrollMergeIntent: inertiainternalprop.ScrollMergeIntentAppend,
			}).
				With(
					inertiaprop.New("stats", "visible"),
					inertiaprop.New("feed", []map[string]int{{"id": 1}},
						inertiaprop.WithMerge(inertiamerge.NewAppendRoot()),
					),
					inertiadeferred.New("notifications", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return []string{"msg"}, nil
					})),
					inertiaoptional.New("settings", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return map[string]string{"theme": "dark"}, nil
					})),
					inertiaprop.New("locale", "en",
						inertiaprop.WithOnce(inertiaonce.NewOnceOpts().Key("locale_key")),
					),
					inertiascroll.New("users", map[string]any{"data": []string{"one"}},
						inertiascroll.WithPagination(nil, &next, &current),
					),
				).
				ExpectProp("stats", "visible").
				ExpectProp("feed", []map[string]int{{"id": 1}}).
				ExpectProp("locale", "en").
				ExpectNoProp("notifications").
				ExpectNoProp("settings").
				ExpectMergeProps("feed", "users.data").
				ExpectDeferredGroup("default", "notifications").
				ExpectOnceProps("locale_key", "locale").
				Run()

			assert.Contains(t, page.ScrollProps, "users")
		},
	)

	t.Run(
		"it should resolve every prop on a partial request that whitelists the parent key while omitting deferredProps metadata",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:               "/users",
				PartialComponent:  inertiatest.DefaultComponent,
				PartialData:       []string{"stats", "feed", "notifications", "locale", "users"},
				ScrollMergeIntent: inertiainternalprop.ScrollMergeIntentAppend,
			}).
				With(
					inertiaprop.New("stats", "visible"),
					inertiaprop.New("feed", []map[string]int{{"id": 1}},
						inertiaprop.WithMerge(inertiamerge.NewAppendRoot()),
					),
					inertiadeferred.New("notifications", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return []string{"msg"}, nil
					})),
					inertiaprop.New("locale", "en",
						inertiaprop.WithOnce(inertiaonce.NewOnceOpts().Key("locale_key")),
					),
					inertiascroll.New("users", map[string]any{"data": []string{"one"}}),
				).
				ExpectProp("stats", "visible").
				ExpectProp("feed", []map[string]int{{"id": 1}}).
				ExpectProp("notifications", []string{"msg"}).
				ExpectProp("locale", "en").
				ExpectProp("users", map[string]any{"data": []string{"one"}}).
				ExpectMergeProps("feed", "users.data").
				ExpectOnceProps("locale_key", "locale").
				ExpectNoDeferredProps().
				Run()
		},
	)

	t.Run(
		"it should keep rescuedProps, scrollProps and mergeProps consistent when a rescued deferred prop coexists with a scroll prop and a merge prop on the same partial request",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:               "/users",
				PartialComponent:  inertiatest.DefaultComponent,
				PartialData:       []string{"rescued", "keep", "users"},
				ScrollMergeIntent: inertiainternalprop.ScrollMergeIntentAppend,
			}).
				With(
					inertiadeferred.New("rescued", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return nil, errRenderSentinelA
					}), inertiadeferred.WithRescue(true)),
					inertiaprop.New("keep", []string{"a"},
						inertiaprop.WithMerge(inertiamerge.NewAppendRoot()),
					),
					inertiascroll.New("users", map[string]any{"data": []string{"one"}}),
				).
				ExpectNoProp("rescued").
				ExpectProp("keep", []string{"a"}).
				ExpectProp("users", map[string]any{"data": []string{"one"}}).
				ExpectRescuedProps("rescued").
				ExpectMergeProps("keep", "users.data").
				ExpectNoDeferredProps().
				Run()

			assert.Contains(t, page.ScrollProps, "users")
		},
	)

	t.Run("it should omit deferredProps when no deferred props are present",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(inertiaprop.New("name", "Roman")).
				Run()

			assert.Nil(t, page.DeferredProps)
		},
	)
}

func TestRenderer_ErrorPropagation(t *testing.T) {
	t.Parallel()

	t.Run("it should wrap prop resolution errors with the prop key on full render",
		func(t *testing.T) {
			t.Parallel()

			page, err := inertiatest.TestRender(
				t,
				inertiaprotocol.Request{URL: "/users"},
				inertiatest.WithProps(
					inertiaprop.NewLazy(
						"failing",
						inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
							return nil, errRenderSentinelA
						}),
					),
				),
			)

			require.Error(t, err)
			require.ErrorIs(t, err, errRenderSentinelA)
			require.ErrorContains(t, err, "failed to resolve prop failing")
			assert.Nil(t, page)
		},
	)

	t.Run("it should wrap prop resolution errors with the prop key on partial render",
		func(t *testing.T) {
			t.Parallel()

			page, err := inertiatest.TestRender(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: inertiatest.DefaultComponent,
				PartialData:      []string{"failing"},
			}, inertiatest.WithProps(
				inertiaoptional.New(
					"failing",
					inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return nil, errRenderSentinelA
					}),
				),
			),
			)

			require.Error(t, err)
			require.ErrorIs(t, err, errRenderSentinelA)
			require.ErrorContains(t, err, "failed to resolve prop failing")
			assert.Nil(t, page)
		},
	)

	t.Run("it should not wrap rescue errors but instead surface them via rescuedProps",
		func(t *testing.T) {
			t.Parallel()

			page, err := inertiatest.TestRender(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: inertiatest.DefaultComponent,
				PartialData:      []string{"rescued"},
			}, inertiatest.WithProps(
				inertiadeferred.New(
					"rescued",
					inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return nil, errRenderSentinelA
					}),
					inertiadeferred.WithRescue(true),
				),
			))

			require.NoError(t, err, "rescued props must not propagate as errors")
			require.NotNil(t, page)
			assert.Contains(t, page.RescuedProps, "rescued")
			assert.NotContains(t, page.Props, "rescued",
				"rescued prop must be omitted from props")
		},
	)
}
