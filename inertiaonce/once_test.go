package inertiaonce_test

import (
	"context"
	"testing"

	"go.segfaultmedaddy.com/inertia/inertiaonce"
	"go.segfaultmedaddy.com/inertia/inertiaprop"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprotocol"
	"go.segfaultmedaddy.com/inertia/internal/inertiatest"
)

func TestOnce(t *testing.T) {
	t.Parallel()

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

	t.Run("it should skip once prop value resolution when its key appears in exceptOnceProps header",
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

	t.Run("it should not skip once prop when Fresh is true despite key being in exceptOnceProps",
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

	t.Run("it should preserve custom once key in onceProps when WithOnce key differs from prop key",
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
}
