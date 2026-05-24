package inertia

import (
	"context"
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

	"go.segfaultmedaddy.com/inertia/inertiadeferred"
	"go.segfaultmedaddy.com/inertia/inertiaprop"
	"go.segfaultmedaddy.com/inertia/inertiascroll"
	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
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
		options            []Option
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
			options:            []Option{},
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
			options:            []Option{},
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
			options:            []Option{},
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
			options:            []Option{},
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
			options:            []Option{},
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
			options: []Option{
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
			options: []Option{
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
			name: "with partial component request",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig: &inertiatest.RequestConfig{
				Inertia:          true,
				PartialComponent: "TestComponent",
				Whitelist:        []string{"title", "content"},
			},
			componentName: "TestComponent",
			options: []Option{
				WithProps(Props{
					inertiaprop.New("title", "Test Title"),
					inertiaprop.New("content", "Test Content"),
					inertiaprop.New("hidden", "Should Not Be Included"),
				}),
			},
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

				props, ok := page["props"].(map[string]any)
				require.True(t, ok, "props should be a map")

				assert.Contains(t, props, "title", "title prop should be included")
				assert.Contains(t, props, "content", "content prop should be included")
				assert.NotContains(t, props, "hidden", "hidden prop should not be included")
			},
		},
		{
			name: "with partial component request with blacklist",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig: &inertiatest.RequestConfig{
				Inertia:          true,
				PartialComponent: "TestComponent",
				Blacklist:        []string{"hidden"},
			},
			componentName: "TestComponent",
			options: []Option{
				WithProps(Props{
					inertiaprop.New("title", "Test Title"),
					inertiaprop.New("content", "Test Content"),
					inertiaprop.New("hidden", "Should Not Be Included"),
				}),
			},
			expectedStatusCode: http.StatusOK,
			expectJSON:         true,
			expectError:        false,
			validateResponse: func(t *testing.T, body []byte) {
				t.Helper()

				var page map[string]any

				err := json.Unmarshal(body, &page)
				require.NoError(t, err, "failed to parse JSON response")

				// Check props are correctly filtered
				props, ok := page["props"].(map[string]any)
				require.True(t, ok, "props should be a map[string]any")

				assert.Contains(t, props, "title", "title prop should be included")
				assert.Contains(t, props, "content", "content prop should be included")
				assert.NotContains(t, props, "hidden", "hidden prop should not be included")
			},
		},
		{
			name: "with deferred props",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig:     &inertiatest.RequestConfig{Inertia: true},
			componentName: "TestComponent",
			options: []Option{
				WithProps(Props{
					inertiaprop.New("visible", "Visible Content"),
					inertiadeferred.New(
						"deferred",
						LazyFunc(
							func(context.Context) (any, error) { return "Lazy Content", nil },
						),
						inertiadeferred.WithGroup("group1"),
					),
				}),
			},
			expectedStatusCode: http.StatusOK,
			expectJSON:         false,
			expectError:        false,
			validateResponse: func(t *testing.T, body []byte) {
				t.Helper()

				var page map[string]any

				err := json.Unmarshal(body, &page)
				require.NoError(t, err, "Failed to parse response JSON")

				// Check props
				props, ok := page["props"].(map[string]any)
				require.True(t, ok, "props not found")
				assert.Equal(t, "Visible Content", props["visible"], "visible prop doesn't match")

				// Check deferred props
				deferredProps, ok := page["deferredProps"].(map[string]any)
				require.True(t, ok, "deferredProps not found")

				group1, ok := deferredProps["group1"].([]any)
				require.True(t, ok, "group1 not found in deferredProps")

				assert.Contains(t, group1, "deferred", "deferred not found in group1")
			},
		},
		{
			name: "with mergeable props",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig:     &inertiatest.RequestConfig{Inertia: true},
			componentName: "TestComponent",
			options: []Option{
				WithProps(Props{
					inertiaprop.New("normalProp", "Normal Value"),
					inertiaprop.New(
						"mergeProp",
						map[string]string{"key": "value"},
						inertiaprop.WithMerge(NewMergeOpts()),
					),
				}),
			},
			expectedStatusCode: http.StatusOK,
			expectJSON:         false,
			expectError:        false,
			validateResponse: func(t *testing.T, body []byte) {
				t.Helper()

				var page map[string]any

				err := json.Unmarshal(body, &page)
				require.NoError(t, err, "Failed to parse response JSON")

				// Check merge props
				mergeProps, ok := page["mergeProps"].([]any)
				require.True(t, ok, "mergeProps not found")

				assert.Contains(t, mergeProps, "mergeProp", "mergeProp not found in mergeProps")

				// Check props
				props, ok := page["props"].(map[string]any)
				require.True(t, ok, "props not found")
				assert.Equal(t, "Normal Value", props["normalProp"], "normalProp doesn't match")

				mergeProp, ok := props["mergeProp"].(map[string]any)
				require.True(t, ok, "mergeProp not found or not a map")
				assert.Equal(t, "value", mergeProp["key"], "mergeProp.key doesn't match")
			},
		},
		{
			name: "with merge props with reset",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig: &inertiatest.RequestConfig{
				Inertia:    true,
				ResetProps: []string{"mergeProp"},
			},
			componentName: "TestComponent",
			options: []Option{
				WithProps(Props{
					inertiaprop.New(
						"mergeProp",
						map[string]string{"key": "value"},
						inertiaprop.WithMerge(NewMergeOpts()),
					),
				}),
			},
			expectedStatusCode: http.StatusOK,
			expectJSON:         false,
			expectError:        false,
			validateResponse: func(t *testing.T, body []byte) {
				t.Helper()

				// Just check that the response is valid JSON - the blacklisted prop
				// should be excluded from merge
				var responseObj map[string]any

				err := json.Unmarshal(body, &responseObj)
				require.NoError(t, err, "Failed to parse response JSON")

				_, ok := responseObj["mergeProps"]
				require.False(t, ok, "mergeProps should not be found")
			},
		},
		{
			name: "with v3 merge metadata",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig:     &inertiatest.RequestConfig{Inertia: true},
			componentName: "TestComponent",
			options: []Option{
				WithProps(Props{
					inertiaprop.New(
						"posts",
						[]string{"one"},
						inertiaprop.WithMerge(NewMergeOpts().Append(MergeKey{MatchOn: "id"})),
					),
					inertiaprop.New("notifications", []string{"one"},
						inertiaprop.WithMerge(NewMergeOpts().Prepend(MergeKey{MatchOn: "uuid"}))),
					inertiaprop.New(
						"conversation",
						map[string]any{"messages": []string{"one"}},
						inertiaprop.WithMerge(NewMergeOpts().Append(MergeKey{Key: "messages", MatchOn: "id"})),
					),
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
				assert.Contains(t, page["mergeProps"], "posts")
				assert.Contains(t, page["mergeProps"], "conversation.messages")
				assert.Contains(t, page["prependProps"], "notifications")
				assert.ElementsMatch(t, []any{
					"posts.id",
					"notifications.uuid",
					"conversation.messages.id",
				}, page["matchPropsOn"])
			},
		},
		{
			name: "with infinite scroll metadata",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig:     &inertiatest.RequestConfig{Inertia: true},
			componentName: "TestComponent",
			options: func() []Option {
				nextPage := 2
				currentPage := 1

				return []Option{
					WithProps(Props{
						inertiascroll.New(
							"users",
							map[string]any{"data": []string{"one"}},
							inertiascroll.WithPagination(nil, &nextPage, &currentPage),
							inertiascroll.WithPageName("page"),
						),
					}),
				}
			}(),
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
				assert.Contains(t, page["mergeProps"], "users.data")

				scrollProps, ok := page["scrollProps"].(map[string]any)
				require.True(t, ok)

				users, ok := scrollProps["users"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "page", users["pageName"])
				assert.Nil(t, users["previousPage"])
				assert.InEpsilon(t, 2, users["nextPage"], 0)
				assert.InEpsilon(t, 1, users["currentPage"], 0)
			},
		},
		{
			name: "infinite scroll prepend intent",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig: &inertiatest.RequestConfig{
				Inertia:           true,
				ScrollMergeIntent: "prepend",
			},
			componentName: "TestComponent",
			options: []Option{
				WithProps(Props{
					inertiascroll.New("users", map[string]any{"data": []string{"one"}}),
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
				assert.NotContains(t, page, "mergeProps")
				assert.Contains(t, page["prependProps"], "users.data")
			},
		},
		{
			name: "infinite scroll append intent",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig: &inertiatest.RequestConfig{
				Inertia:           true,
				ScrollMergeIntent: ScrollMergeIntentAppend,
			},
			componentName: "TestComponent",
			options: []Option{
				WithProps(Props{
					inertiascroll.New("users", map[string]any{"data": []string{"one"}}),
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
				assert.Contains(t, page["mergeProps"], "users.data")
				assert.NotContains(t, page, "prependProps")
			},
		},
		{
			name: "invalid infinite scroll merge intent",
			renderer: New(basicTpl, &Config{
				Version:    "1.0.0",
				RootViewID: "app",
			}),
			reqConfig: &inertiatest.RequestConfig{
				Inertia:           true,
				ScrollMergeIntent: "sideways",
			},
			componentName: "TestComponent",
			options: []Option{
				WithProps(Props{
					inertiascroll.New("users", map[string]any{"data": []string{"one"}}),
				}),
			},
			expectError:   true,
			expectedError: ErrInvalidScrollMergeIntent,
			validateResponse: func(t *testing.T, _ []byte) {
				t.Helper()
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
			options: []Option{
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
			options:            []Option{WithClearHistory()},
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
			options:            []Option{WithEncryptHistory()},
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
			options:            []Option{WithPreserveFragment()},
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
			req, w := inertiatest.NewRequest(http.MethodGet, "/", tt.reqConfig)

			// Create a RenderContext from the options
			rCtx := RenderContext{}
			for _, opt := range tt.options {
				opt(&rCtx)
			}

			req = req.WithContext(context.WithValue(req.Context(), kCtxKey, tt.renderer))

			// Call the package-level HTTP Render function
			err := Render(w, req, tt.componentName, rCtx)

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

			req, w := inertiatest.NewRequest(http.MethodGet, "/current", tt.reqConfig)

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

func TestRenderer_render(t *testing.T) {
	t.Parallel()

	t.Run("returns transport-neutral JSON response", func(t *testing.T) {
		t.Parallel()

		// arrange
		renderer := New(testTpl, &Config{Version: "1.0.0"})
		req := request{
			URL:       "/users",
			IsInertia: true,
			Version:   "1.0.0",
		}
		rCtx := NewRenderContext(WithProps(Props{inertiaprop.New("name", "Roman")}))

		// act
		resp, err := renderer.render(t.Context(), req, "Users/Index", rCtx)

		// assert
		require.NoError(t, err)
		assert.Equal(t, inertiaheader.ContentTypeJSON,
			resp.Headers[inertiaheader.HeaderContentType])
		assert.Equal(t, "true", resp.Headers[inertiaheader.HeaderXInertia])

		var page map[string]any

		err = json.Unmarshal(resp.Body, &page)
		require.NoError(t, err)
		assert.Equal(t, "Users/Index", page["component"])
		assert.Equal(t, "/users", page["url"])
	})
}

func TestParseHeaderValueList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		header   string
		expected []string
	}{
		{
			name:     "empty header",
			header:   "",
			expected: nil,
		},
		{
			name:     "single value",
			header:   "test",
			expected: []string{"test"},
		},
		{
			name:     "multiple values",
			header:   "test1,test2,test3",
			expected: []string{"test1", "test2", "test3"},
		},
		{
			name:     "values with whitespace",
			header:   " test1 , test2 , test3 ",
			expected: []string{"test1", "test2", "test3"},
		},
		{
			name:     "values with mixed whitespace",
			header:   "test1,  test2,test3  ",
			expected: []string{"test1", "test2", "test3"},
		},
		{
			name:     "values with dots",
			header:   "user.name,user.email,user.age",
			expected: []string{"user.name", "user.email", "user.age"},
		},
		{
			name:     "single value with whitespace",
			header:   " test ",
			expected: []string{"test"},
		},
		{
			name:   "empty values between commas",
			header: "test1,,test2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := parseHeaderValueList(tt.header, "Test-Header")
			if tt.expected == nil && tt.header != "" {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result, "extracted list should match expected values")
		})
	}
}

func TestRender_WithoutMiddleware(t *testing.T) {
	t.Parallel()

	// arrange
	req, w := inertiatest.NewRequest(http.MethodGet, "/", nil)

	// act
	err := Render(w, req, "TestComponent", RenderContext{})

	// assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "renderer not found in request context")
}

func TestErrorBagFromRequest(t *testing.T) {
	t.Parallel()

	t.Run("returns error bag from header", func(t *testing.T) {
		t.Parallel()

		// arrange
		req, _ := inertiatest.NewRequest(http.MethodGet, "/", &inertiatest.RequestConfig{
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
		req, _ := inertiatest.NewRequest(http.MethodGet, "/", nil)

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
		req, w := inertiatest.NewRequest(http.MethodGet, "/current", nil)

		// act
		Redirect(w, req, "/target")

		// assert
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/target", w.Header().Get("Location"))
	})

	t.Run("POST request redirects with 303", func(t *testing.T) {
		t.Parallel()

		// arrange
		req, w := inertiatest.NewRequest(http.MethodPost, "/current", nil)

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
		req, w := inertiatest.NewRequest(http.MethodGet, "/current", &inertiatest.RequestConfig{
			Inertia: true,
		})

		// act
		RedirectPreserveFragment(w, req, "/target")

		// assert
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Equal(t, "/target", w.Header().Get(inertiaheader.HeaderXInertiaRedirect))
		assert.Empty(t, w.Header().Get(inertiaheader.HeaderXInertiaLocation))
	})

	t.Run("non-Inertia request uses regular redirect", func(t *testing.T) {
		t.Parallel()

		// arrange
		req, w := inertiatest.NewRequest(http.MethodGet, "/current", nil)

		// act
		RedirectPreserveFragment(w, req, "/target")

		// assert
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/target", w.Header().Get("Location"))
	})
}

func TestRenderer_ConcurrentProps(t *testing.T) {
	t.Parallel()

	// arrange
	basicTpl := template.Must(template.New("test").Parse(`{{.InertiaBody}}`))
	renderer := New(basicTpl, &Config{
		Version:     "1.0.0",
		RootViewID:  "app",
		Concurrency: 2,
	})

	req, w := inertiatest.NewRequest(http.MethodGet, "/", &inertiatest.RequestConfig{
		Inertia:          true,
		PartialComponent: "TestComponent",
		Whitelist:        []string{"a", "b", "c"},
	})

	rCtx := NewRenderContext(
		WithProps(Props{
			inertiadeferred.New("a", LazyFunc(func(context.Context) (any, error) {
				return "val-a", nil
			}), inertiadeferred.WithConcurrent),
			inertiadeferred.New("b", LazyFunc(func(context.Context) (any, error) {
				return "val-b", nil
			}), inertiadeferred.WithConcurrent),
			inertiaprop.New("c", "val-c"),
		}),
	)

	// act
	req = req.WithContext(context.WithValue(req.Context(), kCtxKey, renderer))
	err := Render(w, req, "TestComponent", rCtx)

	// assert
	require.NoError(t, err)

	var page map[string]any

	err = json.Unmarshal(w.Body.Bytes(), &page)
	require.NoError(t, err)

	props, ok := page["props"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "val-a", props["a"])
	assert.Equal(t, "val-b", props["b"])
	assert.Equal(t, "val-c", props["c"])
}
