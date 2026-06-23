package inertia

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
	"go.segfaultmedaddy.com/inertia/internal/inertiatest"
)

func TestRedirect_StandardRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		target         string
		expectedStatus int
	}{
		{
			name:           "it should redirect with 302 when request method is GET",
			method:         http.MethodGet,
			target:         "/target",
			expectedStatus: http.StatusFound,
		},
		{
			name:           "it should redirect with 303 when request method is POST",
			method:         http.MethodPost,
			target:         "/target",
			expectedStatus: http.StatusSeeOther,
		},
		{
			name:           "it should redirect with 303 when request method is PUT",
			method:         http.MethodPut,
			target:         "/target",
			expectedStatus: http.StatusSeeOther,
		},
		{
			name:           "it should redirect with 303 when request method is PATCH",
			method:         http.MethodPatch,
			target:         "/target",
			expectedStatus: http.StatusSeeOther,
		},
		{
			name:           "it should redirect with 303 when request method is DELETE",
			method:         http.MethodDelete,
			target:         "/target",
			expectedStatus: http.StatusSeeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req, w := inertiatest.NewRequest(t, tt.method, "/current", &inertiatest.RequestConfig{})

			Redirect(w, req, tt.target)

			assert.Equal(t, tt.expectedStatus, w.Code, "unexpected status code")
			assert.Equal(t, tt.target, w.Header().Get("Location"), "unexpected Location header")
		})
	}
}

func TestRedirect_InertiaRequest(t *testing.T) {
	t.Parallel()

	t.Run("it should preserve original url fragment when preserve fragment option is set", func(t *testing.T) {
		t.Parallel()

		req, w := inertiatest.NewRequest(t, http.MethodGet, "/current", &inertiatest.RequestConfig{
			Inertia: true,
		})
		req.URL.Fragment = "section"

		Redirect(w, req, "/target", WithRedirectPreserveFragment())

		assert.Equal(t, http.StatusConflict, w.Code, "unexpected status code")
		assert.Equal(t, "/target#section", w.Header().Get(inertiaheader.HeaderXInertiaRedirect),
			"unexpected X-Inertia-Redirect header")
	})

	t.Run("it should delete vary and inertia headers when request is inertia", func(t *testing.T) {
		t.Parallel()

		req, w := inertiatest.NewRequest(t, http.MethodGet, "/current", &inertiatest.RequestConfig{
			Inertia: true,
		})
		w.Header().Set(inertiaheader.HeaderVary, "X-Inertia")
		w.Header().Set(inertiaheader.HeaderXInertia, inertiaheader.HeaderValueTrue)

		Redirect(w, req, "/target")

		assert.Equal(t, http.StatusConflict, w.Code, "unexpected status code")
		assert.Empty(t, w.Header().Get(inertiaheader.HeaderVary), "expected Vary header to be deleted")
		assert.Empty(t, w.Header().Get(inertiaheader.HeaderXInertia),
			"expected X-Inertia header to be deleted")
		assert.Equal(t, "/target", w.Header().Get(inertiaheader.HeaderXInertiaRedirect),
			"unexpected X-Inertia-Redirect header")
	})
}

func TestLocation_StandardRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		target         string
		expectedStatus int
	}{
		{
			name:           "it should redirect with 302 when request method is GET",
			method:         http.MethodGet,
			target:         "/target",
			expectedStatus: http.StatusFound,
		},
		{
			name:           "it should redirect with 303 when request method is POST",
			method:         http.MethodPost,
			target:         "/target",
			expectedStatus: http.StatusSeeOther,
		},
		{
			name:           "it should redirect with 303 when request method is PUT",
			method:         http.MethodPut,
			target:         "/target",
			expectedStatus: http.StatusSeeOther,
		},
		{
			name:           "it should redirect with 303 when request method is PATCH",
			method:         http.MethodPatch,
			target:         "/target",
			expectedStatus: http.StatusSeeOther,
		},
		{
			name:           "it should redirect with 303 when request method is DELETE",
			method:         http.MethodDelete,
			target:         "/target",
			expectedStatus: http.StatusSeeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req, w := inertiatest.NewRequest(t, tt.method, "/current", &inertiatest.RequestConfig{})

			Location(w, req, tt.target)

			assert.Equal(t, tt.expectedStatus, w.Code, "unexpected status code")
			assert.Equal(t, tt.target, w.Header().Get("Location"), "unexpected Location header")
		})
	}
}

func TestLocation_InertiaRequest(t *testing.T) {
	t.Parallel()

	t.Run("it should delete vary and inertia headers when request is inertia", func(t *testing.T) {
		t.Parallel()

		req, w := inertiatest.NewRequest(t, http.MethodGet, "/current", &inertiatest.RequestConfig{
			Inertia: true,
		})
		w.Header().Set(inertiaheader.HeaderVary, "X-Inertia")
		w.Header().Set(inertiaheader.HeaderXInertia, inertiaheader.HeaderValueTrue)

		Location(w, req, "/target")

		assert.Equal(t, http.StatusConflict, w.Code, "unexpected status code")
		assert.Empty(t, w.Header().Get(inertiaheader.HeaderVary), "expected Vary header to be deleted")
		assert.Empty(t, w.Header().Get(inertiaheader.HeaderXInertia),
			"expected X-Inertia header to be deleted")
		assert.Equal(t, "/target", w.Header().Get(inertiaheader.HeaderXInertiaLocation),
			"unexpected X-Inertia-Location header")
	})
}
