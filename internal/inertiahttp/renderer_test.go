package inertiahttp

import (
	"html/template"
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"go.segfaultmedaddy.com/inertia/inertiaotel"
	"go.segfaultmedaddy.com/inertia/internal/inertiassr"
)

func TestNew_Defaults(t *testing.T) {
	t.Parallel()

	t.Run("it should apply default values for all required unset fields when config is nil", func(t *testing.T) {
		t.Parallel()

		// arrange
		tmpl := template.Must(template.New("test").Parse(`<html></html>`))

		// act
		r := New(tmpl, nil)

		// assert
		assert.Equal(t, DefaultRootViewID, r.rootViewID)
		assert.Equal(t, inertiaotel.DefaultConfig, r.telemetry)
		assert.Nil(t, r.ssrClient)
		assert.Empty(t, r.version)
		assert.Same(t, tmpl, r.t)
	})

	t.Run(
		"it should default unset values to their defaults when config is partially populated",
		func(t *testing.T) {
			t.Parallel()

			// arrange
			tmpl := template.Must(template.New("test").Parse(`<html></html>`))
			ctrl := gomock.NewController(t)
			ssrClient := inertiassr.NewMockSSRClient(ctrl)

			config := &Config{
				SSRClient: ssrClient,
				Version:   "1.0.0",
			}

			// act
			r := New(tmpl, config)

			// assert
			assert.Equal(t, DefaultRootViewID, r.rootViewID)
			assert.Equal(t, DefaultConcurrency, config.Concurrency)
			assert.Equal(t, inertiaotel.DefaultConfig, r.telemetry)
			// Set values are preserved.
			assert.Same(t, ssrClient, r.ssrClient)
			assert.Equal(t, "1.0.0", r.Version())
			assert.Same(t, tmpl, r.t)
		},
	)
}

func TestNew_ConfigPropagation(t *testing.T) {
	t.Parallel()

	t.Run(
		"it should propagate all config values to the renderer when config is fully populated",
		func(t *testing.T) {
			t.Parallel()

			// arrange
			tmpl := template.Must(template.New("test").Parse(`<html></html>`))
			ctrl := gomock.NewController(t)
			ssrClient := inertiassr.NewMockSSRClient(ctrl)
			telemetry := inertiaotel.New()
			jsonOpts := []json.Options{json.Deterministic(true)}

			config := &Config{
				SSRClient:          ssrClient,
				RootViewAttrs:      map[string]string{"class": "foo"},
				Telemetry:          telemetry,
				Version:            "2.0.0",
				RootViewID:         "custom-app",
				JSONMarshalOptions: jsonOpts,
				Concurrency:        4,
			}

			// act
			r := New(tmpl, config)

			// assert
			assert.Same(t, tmpl, r.t)
			assert.Same(t, ssrClient, r.ssrClient)
			assert.Equal(t, "2.0.0", r.Version())
			assert.Equal(t, "custom-app", r.rootViewID)
			assert.Same(t, telemetry, r.telemetry)
			assert.Equal(t, jsonOpts, r.jsonMarshalOpts)
		},
	)
}
