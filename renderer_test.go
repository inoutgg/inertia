package inertia

import (
	"html/template"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

var testTemplate = `<!DOCTYPE html>
<html>
<head>
				<title>Test Template</title>
</head>
<body>
				{{ .InertiaBody }}
</body>
</html>`
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
		name      string
		config    *Config
		tpl       *template.Template
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
			renderer, err := FromFS(testFS(), tt.path, tt.config)

			if tt.wantErr {
				assert.Error(t, err, "FromFS should return error with invalid template path")
				assert.Nil(t, renderer, "renderer should be nil when error occurs")
				return
			}

			assert.NoError(t, err, "FromFS should not return error with valid template path")
			assert.NotNil(t, renderer, "renderer should not be nil")
			assert.Equal(t, tt.wantVersion, renderer.Version(), "renderer version should match config")
		})

		t.Run(tt.name+" (MustFromFS)", func(t *testing.T) {
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

func TestRenderer_Version(t *testing.T) {
	t.Parallel()

	renderer := New(testTpl, &Config{Version: "1.0.0"})
	assert.Equal(t, "1.0.0", renderer.Version(), "renderer version should match config")
}

func TestExtractHeaderValueList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		header   string
		expected []string
	}{
		{
			name:     "empty header",
			header:   "",
			expected: []string{},
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
			name:     "empty values between commas",
			header:   "test1,,test2",
			expected: []string{"test1", "", "test2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := extractHeaderValueList(tt.header)
			assert.Equal(t, tt.expected, result, "extracted list should match expected values")
		})
	}
}
