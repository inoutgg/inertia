package inertia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithErrorBag(t *testing.T) {
	m := NewValidationError("Name", "Name is required", nil)

	t.Run("Initial validation error", func(t *testing.T) {
		assert.Equal(t, "", m.ErrorBag())
		assert.Equal(t, 1, m.Len())
		assert.Equal(t, "Name", m.ValidationErrors()[0].Field())
		assert.Equal(t, "Name is required", m.ValidationErrors()[0].Error())
	})

	t.Run("One wrapping", func(t *testing.T) {
		withBag := WithErrorBag("login", m)

		assert.Equal(t, "login", withBag.ErrorBag())
		assert.Equal(t, 1, withBag.Len())

		// Original validation error should be unchanged
		assert.Equal(t, "", m.ErrorBag())

		// Ensure validation errors are still accessible
		assert.Equal(t, "Name", withBag.ValidationErrors()[0].Field())
		assert.Equal(t, "Name is required", withBag.ValidationErrors()[0].Error())
	})

	t.Run("Multiple wrappings", func(t *testing.T) {
		withBag := WithErrorBag("login", m)
		withSecondBag := WithErrorBag("register", withBag)

		// Should use the latest error bag
		assert.Equal(t, "register", withSecondBag.ErrorBag())
		assert.Equal(t, 1, withSecondBag.Len())

		// Previous wrapped error should maintain its error bag
		assert.Equal(t, "login", withBag.ErrorBag())

		// Test one more level of wrapping
		withThirdBag := WithErrorBag("profile", withSecondBag)

		assert.Equal(t, "profile", withThirdBag.ErrorBag())
		assert.Equal(t, 1, withThirdBag.Len())

		// Make sure the ValidationErrors are still accessible through all wrappers
		assert.Equal(t, "Name", withThirdBag.ValidationErrors()[0].Field())
		assert.Equal(t, "Name is required", withThirdBag.ValidationErrors()[0].Error())
	})
}
