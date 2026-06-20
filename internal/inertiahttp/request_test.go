package inertiahttp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
