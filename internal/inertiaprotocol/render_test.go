package inertiaprotocol

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.segfaultmedaddy.com/inertia/inertiaotel"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("New should create a new instance of Renderer", func(t *testing.T) {
		t.Parallel()

		renderer := New(inertiaotel.DefaultConfig)

		assert.NotNil(t, renderer)
	})
}

// func TestRenderer_Render(t *testing.T) {
// 	t.Parallel()

// 	renderer := New(inertiaotel.DefaultConfig)

// 	t.Run("", func(t *testing.T) {
// 		t.Parallel()
// 	})
// }
