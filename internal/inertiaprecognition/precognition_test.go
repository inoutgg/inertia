package inertiaprecognition

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
)

type testValidationError struct {
	field   string
	message string
}

func (err testValidationError) Field() string { return err.field }
func (err testValidationError) Error() string { return err.message }

func TestFilterValidationErrors(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/users", nil)
	req.Header.Set(inertiaheader.HeaderPrecognitionValidateOnly, "users.*.email, profile.name")

	errs := []testValidationError{
		{field: "users.0.email", message: "Email is invalid"},
		{field: "users.0.name", message: "Name is required"},
		{field: "profile.name", message: "Profile name is required"},
	}

	filtered := FilterValidationErrors(req, errs)

	assert.Equal(t, []testValidationError{errs[0], errs[2]}, filtered)
}

func TestWriteValidationErrors(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	w.Header().Add(inertiaheader.HeaderVary, inertiaheader.HeaderXInertia)

	WriteValidationErrors(w, []testValidationError{
		{field: "email", message: "Email is invalid"},
		{field: "email", message: "Email must be unique"},
	})

	resp := w.Result()
	defer resp.Body.Close()

	var body struct {
		Errors map[string][]string `json:"errors"`
	}

	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	assert.Equal(t, inertiaheader.HeaderValueTrue,
		resp.Header.Get(inertiaheader.HeaderPrecognition))
	assert.Contains(t, resp.Header.Values(inertiaheader.HeaderVary), inertiaheader.HeaderXInertia)
	assert.Contains(t, resp.Header.Values(inertiaheader.HeaderVary), inertiaheader.HeaderPrecognition)
	assert.Equal(t, []string{"Email is invalid", "Email must be unique"}, body.Errors["email"])
}
