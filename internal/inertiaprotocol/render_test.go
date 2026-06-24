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

	t.Run("it should resolve only whitelisted props when partialData is non-empty",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: inertiatest.DefaultComponent,
				PartialData:      []string{"name"},
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

	t.Run("it should include bypass-partial-filters props (always) regardless of whitelist and blacklist",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: inertiatest.DefaultComponent,
				PartialData:      []string{"other"},
				PartialExcept:    []string{"auth"},
			}).
				With(
					inertiaalways.New("auth", map[string]string{"user": "Roman"}),
					inertiaprop.New("other", "value"),
				).
				ExpectProp("auth", map[string]string{"user": "Roman"}).
				ExpectProp("other", "value").
				Run()
		},
	)

	t.Run("it should omit deferredProps metadata on partial request regardless of how many deferred props exist",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:              "/users",
				PartialComponent: inertiatest.DefaultComponent,
				PartialData:      []string{"a", "b"},
			}).
				With(
					inertiadeferred.New("a", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return "a-value", nil
					}), inertiadeferred.WithGroup("g1")),
					inertiadeferred.New("b", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return "b-value", nil
					}), inertiadeferred.WithGroup("g2")),
				).
				ExpectProp("a", "a-value").
				ExpectProp("b", "b-value").
				ExpectNoDeferredProps().
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

	t.Run("it should emit deferredProps metadata for deferred props skipped on full request",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(
					inertiadeferred.New("def", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return "def-value", nil
					})),
				).
				ExpectNoProp("def").
				ExpectDeferredGroup("default", "def").
				Run()
		},
	)
}

func TestRenderer_OncePropsResolution(t *testing.T) {
	t.Parallel()

	t.Run("it should resolve once prop value on first load and emit onceProps metadata",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(
					inertiaprop.New("locale", "en",
						inertiaprop.WithOnce(inertiaonce.NewOnceOpts().Key("locale_key")),
					),
				).
				ExpectProp("locale", "en").
				ExpectOnceProps("locale_key", "locale").
				Run()
		},
	)

	t.Run("it should skip once prop value resolution when its key appears in exceptOnceProps",
		func(t *testing.T) {
			t.Parallel()

			lazy := inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
				return "lazy-locale", nil
			})

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:             "/users",
				ExceptOnceProps: []string{"locale_key"},
			}).
				With(
					inertiaprop.NewLazy("locale", lazy,
						inertiaprop.WithOnce(inertiaonce.NewOnceOpts().Key("locale_key")),
					),
					inertiaprop.New("timezone", "UTC"),
				).
				ExpectNoProp("locale").
				ExpectProp("timezone", "UTC").
				ExpectOnceProps("locale_key", "locale").
				Run()

			lazy.ExpectNotCalled(t)
		},
	)

	t.Run("it should not skip once prop when onceable.Fresh is true even if key is in exceptOnceProps",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:             "/users",
				ExceptOnceProps: []string{"locale_key"},
			}).
				With(
					inertiaprop.New("locale", "en",
						inertiaprop.WithOnce(
							inertiaonce.NewOnceOpts().Key("locale_key").Fresh(true),
						),
					),
				).
				ExpectProp("locale", "en").
				ExpectOnceProps("locale_key", "locale").
				Run()
		},
	)

	t.Run("it should preserve custom once key in onceProps map when WithOnce key differs from prop key",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(
					inertiaprop.New("locale", "en",
						inertiaprop.WithOnce(inertiaonce.NewOnceOpts().Key("app-locale")),
					),
				).
				ExpectProp("locale", "en").
				ExpectOnceProps("app-locale", "locale").
				Run()
		},
	)

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

	t.Run(
		"it should suppress deferredProps for a deferred+once prop when its key is in exceptOnceProps",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:             "/users",
				ExceptOnceProps: []string{"posts"},
			}).
				With(
					inertiadeferred.New("posts", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return []string{"p"}, nil
					}),
						inertiadeferred.WithOnce(inertiaonce.NewOnceOpts().Key("posts")),
					),
				).
				ExpectNoProp("posts").
				ExpectOnceProps("posts", "posts").
				Run()

			assert.Nil(
				t,
				page.DeferredProps,
				"deferredProps must be suppressed when once key is already loaded",
			)
		},
	)
}

func TestRenderer_DeferredPropsMetadata(t *testing.T) {
	t.Parallel()

	t.Run("it should group deferred props under default when no group is configured",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(
					inertiadeferred.New("a", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return "a", nil
					})),
					inertiadeferred.New("b", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return "b", nil
					})),
				).
				ExpectNoProp("a").
				ExpectNoProp("b").
				ExpectDeferredGroup("default", "a", "b").
				Run()
		},
	)

	t.Run("it should preserve named groups across multiple deferred props",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(
					inertiadeferred.New("a", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return "a", nil
					}), inertiadeferred.WithGroup("sidebar")),
					inertiadeferred.New("b", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return "b", nil
					}), inertiadeferred.WithGroup("sidebar")),
					inertiadeferred.New("c", inertiatest.NewTestLazyFunc(func(context.Context) (any, error) {
						return "c", nil
					}), inertiadeferred.WithGroup("alerts")),
				).
				ExpectDeferredGroup("sidebar", "a", "b").
				ExpectDeferredGroup("alerts", "c").
				Run()
		},
	)

	t.Run("it should not emit deferredProps when no deferred props are present",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(inertiaprop.New("name", "Roman")).
				Run()

			assert.Nil(t, page.DeferredProps)
		},
	)
}

func TestRenderer_MergePropsMetadata(t *testing.T) {
	t.Parallel()

	t.Run("it should populate mergeProps with root-level append keys when merge.Append is true without paths",
		func(t *testing.T) {
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
		},
	)

	t.Run("it should populate prependProps with root-level prepend keys when merge.Append is false without paths",
		func(t *testing.T) {
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
		},
	)

	t.Run("it should suppress root-level entry when path-based append keys are present",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(
					inertiaprop.New("conversation", map[string]any{"messages": []string{"one"}},
						inertiaprop.WithMerge(
							inertiamerge.NewPaths().Append(inertiamerge.At("messages")),
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
		},
	)

	t.Run("it should populate deepMergeProps and skip append and prepend when DeepMerge is true",
		func(t *testing.T) {
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
		},
	)

	t.Run("it should qualify matchOn keys with the prop root when matchOn keys are non-empty",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(
					inertiaprop.New("conversation", map[string]any{"messages": []string{"one"}},
						inertiaprop.WithMerge(
							inertiamerge.NewPaths().
								Append(inertiamerge.At("messages").On("id")),
						),
					),
				).
				ExpectMergeProps("conversation.messages").
				ExpectMatchPropsOn("conversation.messages.id").
				Run()
		},
	)

	t.Run("it should ignore empty append, prepend and matchOn path segments",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(
					inertiaprop.New("posts", map[string]any{"data": []string{"one"}, "items": []string{"two"}},
						inertiaprop.WithMerge(
							inertiamerge.NewPaths().
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
		},
	)

	t.Run("it should return an error when scrollMergeIntent is invalid",
		func(t *testing.T) {
			t.Parallel()

			page, err := inertiatest.TestRender(t, inertiaprotocol.Request{
				URL:               "/users",
				ScrollMergeIntent: "sideways",
			}, inertiatest.WithProps(inertiascroll.New("users", map[string]any{
				"data": []string{"one"},
			})))

			require.Error(t, err)
			require.ErrorContains(t, err, "invalid scroll merge intent")
			assert.Nil(t, page)
		},
	)

	t.Run("it should default to append when scrollMergeIntent is empty string",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:               "/users",
				ScrollMergeIntent: "",
			}).
				With(inertiascroll.New("users", map[string]any{"data": []string{"one"}})).
				ExpectMergeProps("users.data").
				ExpectNoPrependProps().
				Run()
		},
	)
}

func TestRenderer_ResetHeader(t *testing.T) {
	t.Parallel()

	t.Run(
		"it should suppress merge metadata for non-scroll props whose key is in resetProps while still returning the value",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:        "/users",
				ResetProps: []string{"posts"},
			}).
				With(
					inertiaprop.New("posts", []string{"fresh"},
						inertiaprop.WithMerge(inertiamerge.NewAppendRoot()),
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

	t.Run("it should suppress matchPropsOn entries for reset props",
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

	t.Run("it should suppress scroll-prop merge entries when its key is in resetProps",
		func(t *testing.T) {
			t.Parallel()

			inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:               "/users",
				ScrollMergeIntent: inertiainternalprop.ScrollMergeIntentAppend,
				ResetProps:        []string{"users"},
			}).
				With(inertiascroll.New("users", map[string]any{"data": []string{"one"}})).
				ExpectProp("users", map[string]any{"data": []string{"one"}}).
				ExpectNoMergeProps().
				ExpectNoPrependProps().
				Run()
		},
	)

	t.Run("it should still emit scroll metadata with reset=true when scroll prop key is in resetProps",
		func(t *testing.T) {
			t.Parallel()

			next, current := 2, 1

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:               "/users",
				ScrollMergeIntent: inertiainternalprop.ScrollMergeIntentAppend,
				ResetProps:        []string{"users"},
			}).
				With(inertiascroll.New("users", map[string]any{"data": []string{"fresh"}},
					inertiascroll.WithPagination(nil, &next, &current),
				)).
				Run()

			scroll, ok := page.ScrollProps["users"]
			require.True(t, ok)
			assert.True(t, scroll.Reset, "expected scroll prop reset flag to be set")
		},
	)

	t.Run("it should not flag scroll prop as reset when resetProps contains unrelated keys",
		func(t *testing.T) {
			t.Parallel()

			next, current := 2, 1

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:               "/users",
				ScrollMergeIntent: inertiainternalprop.ScrollMergeIntentAppend,
				ResetProps:        []string{"other"},
			}).
				With(inertiascroll.New("users", map[string]any{"data": []string{"one"}},
					inertiascroll.WithPagination(nil, &next, &current),
				)).
				ExpectMergeProps("users.data").
				Run()

			scroll, ok := page.ScrollProps["users"]
			require.True(t, ok)
			assert.False(t, scroll.Reset, "expected scroll prop reset flag to be false")
		},
	)

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

func TestRenderer_ScrollPropsMetadata(t *testing.T) {
	t.Parallel()

	t.Run("it should emit scrollProps with pageName, pages and reset=false when a scroll prop is present",
		func(t *testing.T) {
			t.Parallel()

			next, current := 2, 1

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:               "/users",
				ScrollMergeIntent: inertiainternalprop.ScrollMergeIntentAppend,
			}).
				With(inertiascroll.New("users", map[string]any{"data": []string{"one"}},
					inertiascroll.WithPagination(nil, &next, &current),
					inertiascroll.WithPageName("page"),
				)).
				Run()

			scroll, ok := page.ScrollProps["users"]
			require.True(t, ok)
			assert.Equal(t, "page", scroll.PageName)
			assert.False(t, scroll.Reset)
			assert.Equal(t, &next, scroll.NextPage)
			assert.Equal(t, &current, scroll.CurrentPage)
		},
	)

	t.Run("it should leave scrollProps empty when no scroll props are present",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{URL: "/users"}).
				With(inertiaprop.New("name", "Roman")).
				Run()

			assert.Nil(t, page.ScrollProps)
		},
	)

	t.Run("it should emit multiple scrollProps entries when multiple scroll props exist",
		func(t *testing.T) {
			t.Parallel()

			page := inertiatest.NewPropTestBuilder(t, inertiaprotocol.Request{
				URL:               "/users",
				ScrollMergeIntent: inertiainternalprop.ScrollMergeIntentAppend,
			}).
				With(
					inertiascroll.New("users", map[string]any{"data": []string{"one"}}),
					inertiascroll.New("notifications", map[string]any{"data": []string{"n"}}),
				).
				Run()

			assert.Contains(t, page.ScrollProps, "users")
			assert.Contains(t, page.ScrollProps, "notifications")
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
