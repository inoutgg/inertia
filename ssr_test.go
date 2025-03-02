package inertia

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPSsrClient(t *testing.T) {
	t.Parallel()

	t.Run("creates client with default http client when nil", func(t *testing.T) {
		client := NewHTTPSsrClient("http://example.com", nil)
		assert.NotNil(t, client, "client should not be nil")
	})

	t.Run("creates client with provided http client", func(t *testing.T) {
		customClient := &http.Client{}
		client := NewHTTPSsrClient("http://example.com", customClient)
		assert.NotNil(t, client, "client should not be nil")
	})
}

func TestSsrRender(t *testing.T) {
	page := &Page{
		Component: "Test",
		Props:     map[string]interface{}{"foo": "bar"},
	}
	pageJson, err := json.Marshal(page)
	assert.NoError(t, err)

	t.Run("successfully renders page", func(t *testing.T) {
		expected := &SsrTemplateData{
			Head: "<head>Test</head>",
			Body: "<body>Content</body>",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			body := r.Body
			defer body.Close()

			buf, err := io.ReadAll(body)
			assert.NoError(t, err)

			// Assert that the request body matches the expected JSON
			assert.Equal(t, buf, pageJson)

			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(expected))
		}))
		defer server.Close()

		client := NewHTTPSsrClient(server.URL, nil)
		result, err := client.Render(page)

		require.NoError(t, err)
		assert.Equal(t, expected.Head, result.Head)
		assert.Equal(t, expected.Body, result.Body)
	})

	t.Run("handles server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewHTTPSsrClient(server.URL, nil)
		_, err := client.Render(page)
		assert.Error(t, err)
	})

	t.Run("handles invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte("invalid json"))
			require.NoError(t, err)
		}))
		defer server.Close()

		client := NewHTTPSsrClient(server.URL, nil)
		_, err := client.Render(page)
		assert.Error(t, err)
	})

	t.Run("handles invalid URL", func(t *testing.T) {
		client := NewHTTPSsrClient("invalid-url", nil)
		_, err := client.Render(page)
		assert.Error(t, err)
	})
}
