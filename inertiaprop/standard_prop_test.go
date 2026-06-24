package inertiaprop_test

import (
	"testing"

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
}
