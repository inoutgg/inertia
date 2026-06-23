package inertiahttp

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.segfaultmedaddy.com/inertia/internal/inertiatest"
)

func TestParseRequest_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
		config *inertiatest.RequestConfig
		want   Request
	}{
		{
			name:   "should populate only URL and IsInertia when request is not inertia",
			target: "/",
			config: nil,
			want:   Request{URL: "/", IsInertia: false},
		},
		{
			name:   "should parse version when inertia header is set",
			target: "/users",
			config: &inertiatest.RequestConfig{Inertia: true, Version: "1.0.0"},
			want:   Request{URL: "/users", IsInertia: true, Version: "1.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			r, _ := inertiatest.NewRequest(t, http.MethodGet, tt.target, tt.config)

			// act
			got, err := ParseRequest(r)

			// assert
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseRequest_Partial(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr error
		config  *inertiatest.RequestConfig
		name    string
		target  string
		want    Request
	}{
		{
			name:   "should parse partial component when header is set",
			target: "/users",
			config: &inertiatest.RequestConfig{Inertia: true, Version: "1.0.0", PartialComponent: "Users"},
			want:   Request{URL: "/users", IsInertia: true, Version: "1.0.0", PartialComponent: "Users"},
		},
		{
			name:   "should parse partial data when partial component is set",
			target: "/users",
			config: &inertiatest.RequestConfig{
				Inertia:          true,
				Version:          "1.0.0",
				PartialComponent: "Users",
				Whitelist:        []string{"a", "b"},
			},
			want: Request{
				URL: "/users", IsInertia: true, Version: "1.0.0",
				PartialComponent: "Users", PartialData: []string{"a", "b"},
			},
		},
		{
			name:   "should parse partial except when partial component is set",
			target: "/users",
			config: &inertiatest.RequestConfig{
				Inertia: true, Version: "1.0.0", PartialComponent: "Users", Blacklist: []string{"a"},
			},
			want: Request{
				URL: "/users", IsInertia: true, Version: "1.0.0",
				PartialComponent: "Users", PartialExcept: []string{"a"},
			},
		},
		{
			name:    "should return error when partial data is set without partial component",
			target:  "/users",
			config:  &inertiatest.RequestConfig{Inertia: true, Version: "1.0.0", Whitelist: []string{"a"}},
			wantErr: ErrInvalidInertiaRequest,
		},
		{
			name:    "should return error when partial except is set without partial component",
			target:  "/users",
			config:  &inertiatest.RequestConfig{Inertia: true, Version: "1.0.0", Blacklist: []string{"a"}},
			wantErr: ErrInvalidInertiaRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			r, _ := inertiatest.NewRequest(t, http.MethodGet, tt.target, tt.config)

			// act
			got, err := ParseRequest(r)

			// assert
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseRequest_Reset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
		config *inertiatest.RequestConfig
		want   Request
	}{
		{
			name:   "should parse reset props when header is set",
			target: "/users",
			config: &inertiatest.RequestConfig{Inertia: true, Version: "1.0.0", ResetProps: []string{"x"}},
			want: Request{
				URL: "/users", IsInertia: true, Version: "1.0.0", ResetProps: []string{"x"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			r, _ := inertiatest.NewRequest(t, http.MethodGet, tt.target, tt.config)

			// act
			got, err := ParseRequest(r)

			// assert
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseRequest_ExceptOnce(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
		config *inertiatest.RequestConfig
		want   Request
	}{
		{
			name:   "should parse except once props when header is set",
			target: "/users",
			config: &inertiatest.RequestConfig{Inertia: true, Version: "1.0.0", OnceProps: []string{"y"}},
			want: Request{
				URL: "/users", IsInertia: true, Version: "1.0.0", ExceptOnceProps: []string{"y"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			r, _ := inertiatest.NewRequest(t, http.MethodGet, tt.target, tt.config)

			// act
			got, err := ParseRequest(r)

			// assert
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseRequest_ErrorBag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
		config *inertiatest.RequestConfig
		want   Request
	}{
		{
			name:   "should parse error bag when header is set",
			target: "/users",
			config: &inertiatest.RequestConfig{Inertia: true, Version: "1.0.0", ErrorBag: "default"},
			want: Request{
				URL: "/users", IsInertia: true, Version: "1.0.0", ErrorBag: "default",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			r, _ := inertiatest.NewRequest(t, http.MethodGet, tt.target, tt.config)

			// act
			got, err := ParseRequest(r)

			// assert
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseRequest_ScrollIntent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr error
		config  *inertiatest.RequestConfig
		name    string
		target  string
		want    Request
	}{
		{
			name:   "should parse append intent when header is append",
			target: "/users",
			config: &inertiatest.RequestConfig{
				Inertia:           true,
				Version:           "1.0.0",
				ScrollMergeIntent: "append",
			},
			want: Request{URL: "/users", IsInertia: true, Version: "1.0.0", ScrollMergeIntent: "append"},
		},
		{
			name:   "should parse prepend intent when header is prepend",
			target: "/users",
			config: &inertiatest.RequestConfig{
				Inertia:           true,
				Version:           "1.0.0",
				ScrollMergeIntent: "prepend",
			},
			want: Request{URL: "/users", IsInertia: true, Version: "1.0.0", ScrollMergeIntent: "prepend"},
		},
		{
			name:   "should trim whitespace when intent has surrounding spaces",
			target: "/users",
			config: &inertiatest.RequestConfig{
				Inertia:           true,
				Version:           "1.0.0",
				ScrollMergeIntent: "  append  ",
			},
			want: Request{URL: "/users", IsInertia: true, Version: "1.0.0", ScrollMergeIntent: "append"},
		},
		{
			name:   "should return error when intent is invalid",
			target: "/users",
			config: &inertiatest.RequestConfig{
				Inertia:           true,
				Version:           "1.0.0",
				ScrollMergeIntent: "sideways",
			},
			wantErr: ErrInvalidInertiaRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			r, _ := inertiatest.NewRequest(t, http.MethodGet, tt.target, tt.config)

			// act
			got, err := ParseRequest(r)

			// assert
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseRequest_Precognition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
		config *inertiatest.RequestConfig
		want   Request
	}{
		{
			name:   "should return early with validate-only props when precognition is set",
			target: "/users",
			config: &inertiatest.RequestConfig{
				Inertia:                  true,
				Version:                  "1.0.0",
				Precognition:             true,
				PrecognitionValidateOnly: []string{"name", "email"},
			},
			want: Request{
				URL:               "/users",
				IsInertia:         true,
				Version:           "1.0.0",
				Precognition:      true,
				PrecognitionProps: []string{"name", "email"},
			},
		},
		{
			name:   "should skip partial component validation when precognition is set",
			target: "/users",
			config: &inertiatest.RequestConfig{
				Inertia:      true,
				Version:      "1.0.0",
				Whitelist:    []string{"a"},
				Precognition: true,
			},
			want: Request{
				URL:          "/users",
				IsInertia:    true,
				Version:      "1.0.0",
				Precognition: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			r, _ := inertiatest.NewRequest(t, http.MethodGet, tt.target, tt.config)

			// act
			got, err := ParseRequest(r)

			// assert
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseRequest_Complex(t *testing.T) {
	t.Parallel()

	t.Run("should parse all headers together when full request is sent", func(t *testing.T) {
		t.Parallel()

		// arrange
		r, _ := inertiatest.NewRequest(t, http.MethodGet, "/users", &inertiatest.RequestConfig{
			Inertia:           true,
			Version:           "1.0.0",
			PartialComponent:  "Users",
			ErrorBag:          "default",
			ScrollMergeIntent: "append",
			Whitelist:         []string{"a", "b"},
			Blacklist:         []string{"c"},
			ResetProps:        []string{"x"},
			OnceProps:         []string{"y"},
		})

		// act
		got, err := ParseRequest(r)

		// assert
		require.NoError(t, err)
		assert.Equal(t, Request{
			URL:               "/users",
			IsInertia:         true,
			Version:           "1.0.0",
			PartialComponent:  "Users",
			ErrorBag:          "default",
			ScrollMergeIntent: "append",
			PartialData:       []string{"a", "b"},
			PartialExcept:     []string{"c"},
			ResetProps:        []string{"x"},
			ExceptOnceProps:   []string{"y"},
		}, got)
	})

	t.Run("should ignore non-precognition headers when precognition is set", func(t *testing.T) {
		t.Parallel()

		// arrange
		r, _ := inertiatest.NewRequest(t, http.MethodGet, "/users", &inertiatest.RequestConfig{
			Inertia:                  true,
			Version:                  "1.0.0",
			PartialComponent:         "Users",
			ErrorBag:                 "default",
			ScrollMergeIntent:        "append",
			Whitelist:                []string{"a", "b"},
			Blacklist:                []string{"c"},
			ResetProps:               []string{"x"},
			OnceProps:                []string{"y"},
			Precognition:             true,
			PrecognitionValidateOnly: []string{"name", "email"},
		})

		// act
		got, err := ParseRequest(r)

		// assert
		require.NoError(t, err)
		assert.Equal(t, Request{
			URL:               "/users",
			IsInertia:         true,
			Version:           "1.0.0",
			Precognition:      true,
			PrecognitionProps: []string{"name", "email"},
		}, got)
	})

	t.Run("should parse partial data and partial except together when component is set", func(t *testing.T) {
		t.Parallel()

		// arrange
		r, _ := inertiatest.NewRequest(t, http.MethodGet, "/users", &inertiatest.RequestConfig{
			Inertia:           true,
			Version:           "1.0.0",
			PartialComponent:  "Users",
			ScrollMergeIntent: "prepend",
			Whitelist:         []string{"a", "b"},
			Blacklist:         []string{"c", "d"},
		})

		// act
		got, err := ParseRequest(r)

		// assert
		require.NoError(t, err)
		assert.Equal(t, Request{
			URL:               "/users",
			IsInertia:         true,
			Version:           "1.0.0",
			PartialComponent:  "Users",
			ScrollMergeIntent: "prepend",
			PartialData:       []string{"a", "b"},
			PartialExcept:     []string{"c", "d"},
		}, got)
	})
}

func TestParseHeaderValueList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected []string
	}{
		{
			name:  "empty header",
			value: "",
		},
		{
			name:     "single value",
			value:    "test",
			expected: []string{"test"},
		},
		{
			name:     "multiple values",
			value:    "test1,test2,test3",
			expected: []string{"test1", "test2", "test3"},
		},
		{
			name:     "values with whitespace",
			value:    " test1 , test2 , test3 ",
			expected: []string{"test1", "test2", "test3"},
		},
		{
			name:     "values with mixed whitespace",
			value:    "test1,  test2,test3  ",
			expected: []string{"test1", "test2", "test3"},
		},
		{
			name:     "values with dots",
			value:    "user.name,user.email,user.age",
			expected: []string{"user.name", "user.email", "user.age"},
		},
		{
			name:     "single value with whitespace",
			value:    " test ",
			expected: []string{"test"},
		},
		{
			name:     "empty values between commas",
			value:    "test1,,test2",
			expected: []string{"test1", "test2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := parseHeaderValueList(tt.value)
			assert.Equal(t, tt.expected, result, "extracted list should match expected values")
		})
	}
}
