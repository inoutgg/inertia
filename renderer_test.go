package inertia

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"testing"
	"testing/fstest"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go.segfaultmedaddy.com/inertia/inertiaprop"
	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
	"go.segfaultmedaddy.com/inertia/internal/inertiahttp"
	"go.segfaultmedaddy.com/inertia/internal/inertiassr"
	"go.segfaultmedaddy.com/inertia/internal/inertiatest"
)

//nolint:gochecknoglobals
var testTemplate = `<!DOCTYPE html>
<html>
<head>
				<title>Test Template</title>
</head>
<body>
				{{ .InertiaBody }}
</body>
</html>`

//nolint:gochecknoglobals
var testTpl = template.Must(template.New("test").Parse(testTemplate))

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"templates/app.html": &fstest.MapFile{
			Data: []byte(testTemplate),
			Mode: 0o644,
		},
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	// Test cases
	tests := []struct {
		config    *Config
		tpl       *template.Template
		name      string
		wantPanic bool
	}{
		{name: "invalidate template", tpl: nil, wantPanic: true},
		{name: "empty config", tpl: testTpl},
		{name: "valid config", tpl: testTpl, config: &Config{Version: "1.0.0", RootViewID: "test-app"}},
		{name: "invalid RootViewID", tpl: testTpl, config: &Config{RootViewID: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.wantPanic {
				assert.Panics(t, func() {
					New(tt.tpl, tt.config)
				}, "New should panic")

				return
			}

			renderer := New(tt.tpl, tt.config)
			assert.NotNil(t, renderer, "New should return renderer")
		})
	}
}

func TestFromFS(t *testing.T) {
	t.Parallel()

	// Test cases
	tests := []struct {
		name        string
		path        string
		config      *Config
		wantVersion string
		wantErr     bool
		wantPanic   bool
	}{
		{
			name:        "valid template with config",
			path:        "templates/*.html",
			config:      &Config{Version: "1.0.0", RootViewID: "test-app"},
			wantVersion: "1.0.0",
			wantErr:     false,
			wantPanic:   false,
		},
		{
			name:        "valid template without config",
			path:        "templates/*.html",
			config:      nil,
			wantVersion: "",
			wantErr:     false,
			wantPanic:   false,
		},
		{
			name:      "invalid template path",
			path:      "nonexistent/*.html",
			config:    nil,
			wantErr:   true,
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" (FromFS)", func(t *testing.T) {
			t.Parallel()

			renderer, err := FromFS(testFS(), tt.path, tt.config)

			if tt.wantErr {
				require.Error(t, err, "FromFS should return error with invalid template path")
				assert.Nil(t, renderer, "renderer should be nil when error occurs")

				return
			}

			require.NoError(t, err, "FromFS should not return error with valid template path")
			assert.NotNil(t, renderer, "renderer should not be nil")
			assert.Equal(t, tt.wantVersion, renderer.Version(), "renderer version should match config")
		})

		t.Run(tt.name+" (MustFromFS)", func(t *testing.T) {
			t.Parallel()

			if tt.wantPanic {
				assert.Panics(t, func() {
					MustFromFS(testFS(), tt.path, tt.config)
				}, "FromFS should panic")

				return
			}

			var renderer *Renderer

			assert.NotPanics(t, func() {
				renderer = MustFromFS(testFS(), tt.path, tt.config)
			}, "FromFS should not panic with valid template path")

			assert.NotNil(t, renderer, "renderer should not be nil")
			assert.Equal(t, tt.wantVersion, renderer.Version(), "renderer version should match config")
		})
	}
}

func TestRenderer_RenderProtocolResponse(t *testing.T) {
	t.Parallel()

	// Basic template for testing
	basicTemplate := `<!DOCTYPE html>
<html>
<head>
	<title>Test Template</title>
	{{.InertiaHead}}
</head>
<body>
	{{.InertiaBody}}
</body>
</html>`

	basicTpl := template.Must(template.New("test").Parse(basicTemplate))

	// Create a mock SSR client
	ctrl := gomock.NewController(t)

	t.Cleanup(func() {
		ctrl.Finish()
	})

	mockSSRClient := inertiassr.NewMockSSRClient(ctrl)
	mockSSRClient.EXPECT().Render(gomock.Any(), gomock.Any()).Return(&inertiassr.SSRTemplateData{
		Head: "<title>SSR Title</title>",
		Body: "<div>SSR Content</div>",
	}, nil).AnyTimes()

	errorMockSsrClient := inertiassr.NewMockSSRClient(ctrl)
	errorMockSsrClient.EXPECT().Render(gomock.Any(), gomock.Any()).Return(nil, errors.New("SSR error")).AnyTimes()

	type responseValidator func(t *testing.T, body []byte)

	tests := []struct {
		expectedError      error
		renderer           *Renderer
		reqConfig          *inertiatest.RequestConfig
		expectedHeaders    map[string]string
		validateResponse   responseValidator
		name               string
		componentName      string
		options            []RenderContextOption
		expectedStatusCode int
		expectJSON         bool
		expectError        bool
	}{
		{
			name: "non-inertia request - html response",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig:          &inertiatest.RequestConfig{},
			componentName:      "TestComponent",
			options:            []RenderContextOption{},
			expectedStatusCode: http.StatusOK,
			expectedHeaders: map[string]string{
				inertiaheader.HeaderContentType: inertiaheader.ContentTypeHTML,
			},
			expectJSON:  false,
			expectError: false,
			validateResponse: func(t *testing.T, body []byte) {
				t.Helper()

				// assert
				snaps.MatchSnapshot(t, string(body))
			},
		},
		{
			name: "inertia request - json response",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig: &inertiatest.RequestConfig{
				Inertia: true,
			},
			componentName:      "TestComponent",
			options:            []RenderContextOption{},
			expectedStatusCode: http.StatusOK,
			expectedHeaders: map[string]string{
				inertiaheader.HeaderContentType: inertiaheader.ContentTypeJSON,
				inertiaheader.HeaderXInertia:    "true",
			},
			expectJSON:  true,
			expectError: false,
			validateResponse: func(t *testing.T, body []byte) {
				t.Helper()

				var page map[string]any

				err := json.Unmarshal(body, &page)
				require.NoError(t, err, "failed to parse JSON response")

				assert.Equal(t, "TestComponent", page["component"])
				assert.Equal(t, "1.0.0", page["version"])
			},
		},
		{
			name: "ssr enabled - html response",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
				SSRClient:  mockSSRClient,
			}),
			reqConfig:          &inertiatest.RequestConfig{},
			componentName:      "TestComponent",
			options:            []RenderContextOption{},
			expectedStatusCode: http.StatusOK,
			expectedHeaders: map[string]string{
				inertiaheader.HeaderContentType: inertiaheader.ContentTypeHTML,
			},
			expectJSON:  false,
			expectError: false,
			validateResponse: func(t *testing.T, body []byte) {
				t.Helper()

				// assert
				snaps.MatchSnapshot(t, string(body))
			},
		},
		{
			name: "ssr with error - falls back to client rendering",
			renderer: New(basicTpl, &Config{
				Version:   "1.0.0",
				SSRClient: errorMockSsrClient,
			}),
			reqConfig:          &inertiatest.RequestConfig{},
			componentName:      "TestComponent",
			options:            []RenderContextOption{},
			expectedStatusCode: http.StatusOK,
			expectError:        false,
			validateResponse: func(t *testing.T, body []byte) {
				t.Helper()

				// assert
				snaps.MatchSnapshot(t, string(body))
			},
		},
		{
			name: "with root view attributes",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
				RootViewAttrs: map[string]string{
					"class":     "container",
					"data-test": "value",
					"data-page": "should-be-skipped", // Should be ignored
				},
			}),
			reqConfig:          &inertiatest.RequestConfig{},
			componentName:      "TestComponent",
			options:            []RenderContextOption{},
			expectedStatusCode: http.StatusOK,
			expectJSON:         false,
			expectError:        false,
			validateResponse: func(t *testing.T, body []byte) {
				t.Helper()

				// assert
				snaps.MatchSnapshot(t, string(body))
			},
		},
		{
			name: "with validation errors",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig:     &inertiatest.RequestConfig{Inertia: true},
			componentName: "TestComponent",
			options: []RenderContextOption{
				WithValidationErrors(ValidationErrors{
					NewValidationError("name", "Name is required"),
					NewValidationError("email", "Invalid email"),
				}, DefaultErrorBag),
			},
			expectedStatusCode: http.StatusOK,
			expectJSON:         false,
			expectError:        false,
			validateResponse: func(t *testing.T, body []byte) {
				t.Helper()

				var page map[string]any

				err := json.Unmarshal(body, &page)
				require.NoError(t, err, "Failed to parse response JSON")

				props, ok := page["props"].(map[string]any)
				require.True(t, ok, "props not found")

				errors, ok := props["errors"].(map[string]any)
				require.True(t, ok, "errors not found")

				assert.Equal(t, "Name is required", errors["name"], "name error doesn't match")
				assert.Equal(t, "Invalid email", errors["email"], "email error doesn't match")
			},
		},
		{
			name: "with custom error bag",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig: &inertiatest.RequestConfig{
				Inertia: true,
			},
			componentName: "TestComponent",
			options: []RenderContextOption{
				WithValidationErrors(ValidationErrors{
					NewValidationError("name", "Name is required"),
				}, "custom_errors"),
			},
			expectedStatusCode: http.StatusOK,
			expectJSON:         false,
			expectError:        false,
			validateResponse: func(t *testing.T, body []byte) {
				t.Helper()

				var page map[string]any

				err := json.Unmarshal(body, &page)
				require.NoError(t, err, "Failed to parse response JSON")

				props, ok := page["props"].(map[string]any)
				require.True(t, ok, "props not found")

				customErrors, ok := props["custom_errors"].(map[string]any)
				require.True(t, ok, "custom_errors not found")

				errors, ok := customErrors["errors"].(map[string]any)
				require.True(t, ok, "errors not found")

				assert.Equal(t, "Name is required", errors["name"], "name error doesn't match")
			},
		},
		{
			name: "with shared props metadata",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig:     &inertiatest.RequestConfig{Inertia: true},
			componentName: "TestComponent",
			options: []RenderContextOption{
				WithSharedProps(Props{
					inertiaprop.New("auth", "shared"),
				}),
				WithProps(Props{
					inertiaprop.New("auth", "response"),
				}),
			},
			expectedStatusCode: http.StatusOK,
			expectJSON:         false,
			expectError:        false,
			validateResponse: func(t *testing.T, body []byte) {
				t.Helper()

				// arrange
				var page map[string]any

				// act
				err := json.Unmarshal(body, &page)

				// assert
				require.NoError(t, err, "Failed to parse response JSON")

				props, ok := page["props"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "response", props["auth"])
				assert.Contains(t, page["sharedProps"], "auth")
			},
		},
		{
			name: "clear history flag",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig:          &inertiatest.RequestConfig{Inertia: true},
			componentName:      "TestComponent",
			options:            []RenderContextOption{WithClearHistory()},
			expectedStatusCode: http.StatusOK,
			expectJSON:         false,
			expectError:        false,
			validateResponse: func(t *testing.T, body []byte) {
				t.Helper()

				var page map[string]any

				err := json.Unmarshal(body, &page)
				require.NoError(t, err, "Failed to parse response JSON")

				clearHistory, ok := page["clearHistory"].(bool)
				require.True(t, ok, "clearHistory not found or not a boolean")
				assert.True(t, clearHistory, "clearHistory should be true")
			},
		},
		{
			name: "encrypt history flag",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig:          &inertiatest.RequestConfig{Inertia: true},
			componentName:      "TestComponent",
			options:            []RenderContextOption{WithEncryptHistory()},
			expectedStatusCode: http.StatusOK,
			expectJSON:         false,
			expectError:        false,
			validateResponse: func(t *testing.T, body []byte) {
				t.Helper()

				var page map[string]any

				err := json.Unmarshal(body, &page)
				require.NoError(t, err, "Failed to parse response JSON")

				encryptHistory, ok := page["encryptHistory"].(bool)
				require.True(t, ok, "encryptHistory not found or not a boolean")
				assert.True(t, encryptHistory, "encryptHistory should be true")
			},
		},
		{
			name: "preserve fragment flag",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig:          &inertiatest.RequestConfig{Inertia: true},
			componentName:      "TestComponent",
			options:            []RenderContextOption{WithPreserveFragment()},
			expectedStatusCode: http.StatusOK,
			expectJSON:         false,
			expectError:        false,
			validateResponse: func(t *testing.T, body []byte) {
				t.Helper()

				// arrange
				var page map[string]any

				// act
				err := json.Unmarshal(body, &page)

				// assert
				require.NoError(t, err, "Failed to parse response JSON")

				preserveFragment, ok := page["preserveFragment"].(bool)
				require.True(t, ok, "preserveFragment not found or not a boolean")
				assert.True(t, preserveFragment, "preserveFragment should be true")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create request and recorder using inertiatest
			req, w := inertiatest.NewRequest(t, http.MethodGet, "/", tt.reqConfig)

			// Create a RenderContext from the options
			rCtx := RenderContext{}
			for _, opt := range tt.options {
				opt(&rCtx)
			}

			scope, err := tt.renderer.NewScope(req)
			require.NoError(t, err)

			req = inertiahttp.WithRenderScope(req, scope)

			// Call the package-level HTTP Render function
			err = Render(w, req, tt.componentName, rCtx)

			// Check for expected error conditions
			if tt.expectError {
				assert.Error(t, err, "expected an error but got none")

				if tt.expectedError != nil {
					require.ErrorIs(t, err, tt.expectedError)
				}

				return
			}

			require.NoError(t, err, "unexpected error")

			// Check status code
			if tt.expectedStatusCode > 0 {
				assert.Equal(t, tt.expectedStatusCode, w.Code, "status code does not match")
			}

			// Check headers
			for key, value := range tt.expectedHeaders {
				assert.Equal(t, value, w.Header().Get(key), "header %s does not match", key)
			}

			// Run the custom validation function for this test case
			if tt.validateResponse != nil {
				tt.validateResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestLocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reqConfig      *inertiatest.RequestConfig
		expectedHeader map[string]string
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "non-inertia request",
			reqConfig:      &inertiatest.RequestConfig{},
			url:            "/redirect",
			expectedStatus: http.StatusFound, // 302 Found
			expectedHeader: map[string]string{
				"Location": "/redirect",
			},
		},
		{
			name: "inertia request",
			reqConfig: &inertiatest.RequestConfig{
				Inertia: true,
			},
			url:            "/redirect",
			expectedStatus: http.StatusConflict, // 409 Conflict
			expectedHeader: map[string]string{
				inertiaheader.HeaderXInertiaLocation: "/redirect",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req, w := inertiatest.NewRequest(t, http.MethodGet, "/current", tt.reqConfig)

			// Add test-specific header to test cleanup (for inertia request headers are cleaned test)
			if tt.name == "inertia request headers are cleaned" {
				req.Header.Set("Vary", "some-value")
			}

			Location(w, req, tt.url)

			assert.Equal(t, tt.expectedStatus, w.Code, "unexpected status code")

			for header, value := range tt.expectedHeader {
				assert.Equal(t, value, w.Header().Get(header),
					"unexpected header value for %s", header)
			}
		})
	}
}

func TestRenderer_Version(t *testing.T) {
	t.Parallel()

	renderer := New(testTpl, &Config{Version: "1.0.0"})
	assert.Equal(t, "1.0.0", renderer.Version(), "renderer version should match config")
}

func TestErrorBagFromRequest(t *testing.T) {
	t.Parallel()

	t.Run("returns error bag from header", func(t *testing.T) {
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

func TestRedirect(t *testing.T) {
	t.Parallel()

	t.Run("GET request redirects with 302", func(t *testing.T) {
		t.Parallel()

		// arrange
		req, w := inertiatest.NewRequest(t, http.MethodGet, "/current", nil)

		// act
		Redirect(w, req, "/target")

		// assert
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/target", w.Header().Get("Location"))
	})

	t.Run("POST request redirects with 303", func(t *testing.T) {
		t.Parallel()

		// arrange
		req, w := inertiatest.NewRequest(t, http.MethodPost, "/current", nil)

		// act
		Redirect(w, req, "/target")

		// assert
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/target", w.Header().Get("Location"))
	})
}

func TestRedirectPreserveFragment(t *testing.T) {
	t.Parallel()

	t.Run("Inertia request returns fragment redirect response", func(t *testing.T) {
		t.Parallel()

		// arrange
		req, w := inertiatest.NewRequest(t, http.MethodGet, "/current", &inertiatest.RequestConfig{
			Inertia: true,
		})

		// act
		Redirect(w, req, "/target")

		// assert
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Equal(t, "/target", w.Header().Get(inertiaheader.HeaderXInertiaRedirect))
		assert.Empty(t, w.Header().Get(inertiaheader.HeaderXInertiaLocation))
	})

	t.Run("non-Inertia request uses regular redirect", func(t *testing.T) {
		t.Parallel()

		// arrange
		req, w := inertiatest.NewRequest(t, http.MethodGet, "/current", nil)

		// act
		Redirect(w, req, "/target")

		// assert
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/target", w.Header().Get("Location"))
	})
}
