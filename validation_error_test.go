package inertia

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("email", "Email is invalid")

	assert.Equal(t, "email", err.Field_, "Field should match")
	assert.Equal(t, "Email is invalid", err.Message_, "Message should match")
	assert.Empty(t, err.ErrorBag_, "ErrorBag should be empty")
}

func TestValidationError_Error(t *testing.T) {
	err := NewValidationError("email", "Email is invalid")
	assert.Equal(t, "Email is invalid", err.Error(), "Error() should return the message")
}

func TestValidationError_Field(t *testing.T) {
	err := NewValidationError("email", "Email is invalid")
	assert.Equal(t, "email", err.Field(), "Field() should return the field name")
}

func TestValidationError_ValidationErrors(t *testing.T) {
	err := NewValidationError("email", "Email is invalid")
	errors := err.ValidationErrors()

	require.Len(t, errors, 1, "ValidationErrors() should return a slice with 1 element")
	assert.Same(t, err, errors[0], "The returned validation error should be the same instance")
}

func TestValidationError_Len(t *testing.T) {
	err := NewValidationError("email", "Email is invalid")
	assert.Equal(t, 1, err.Len(), "Len() should return 1")
}

func TestValidationErrors_ValidationErrors(t *testing.T) {
	err1 := NewValidationError("email", "Email is invalid")
	err2 := NewValidationError("password", "Password is too short")

	errs := ValidationErrors{err1, err2}
	returnedErrs := errs.ValidationErrors()

	require.Len(t, returnedErrs, 2, "ValidationErrors() should return 2 errors")
	assert.Same(t, err1, returnedErrs[0], "First error should be the same instance")
	assert.Same(t, err2, returnedErrs[1], "Second error should be the same instance")
}

func TestValidationErrors_Len(t *testing.T) {
	err1 := NewValidationError("email", "Email is invalid")
	err2 := NewValidationError("password", "Password is too short")

	errs := ValidationErrors{err1, err2}
	assert.Equal(t, 2, errs.Len(), "Len() should return 2")

	// Test empty errors
	emptyErrs := ValidationErrors{}
	assert.Equal(t, 0, emptyErrs.Len(), "Len() should return 0 for empty errors")
}
