package inertia

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.segfaultmedaddy.com/inertia/internal/inertiatest"
)

func TestErrorBagFromRequest(t *testing.T) {
	t.Parallel()

	t.Run("it should return error bag from header when header is present", func(t *testing.T) {
		t.Parallel()

		// arrange
		req, _ := inertiatest.NewRequest(t, http.MethodGet, "/", &inertiatest.RequestConfig{
			ErrorBag: "custom_bag",
		})
		// act
		result := ErrorBagFromRequest(req)

		// assert
		assert.Equal(t, "custom_bag", result)
	})

	t.Run("returns default error bag when header is empty", func(t *testing.T) {
		t.Parallel()

		// arrange
		req, _ := inertiatest.NewRequest(t, http.MethodGet, "/", nil)

		// act
		result := ErrorBagFromRequest(req)

		// assert
		assert.Equal(t, DefaultErrorBag, result)
	})
}
