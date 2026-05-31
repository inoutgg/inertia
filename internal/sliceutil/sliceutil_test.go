package sliceutil

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		in       []int
		f        func(int) string
		expected []string
	}{
		{
			name:     "should convert integers to strings for non-empty slice",
			in:       []int{1, 2, 3},
			f:        strconv.Itoa,
			expected: []string{"1", "2", "3"},
		},
		{
			name:     "should return empty slice for empty input",
			in:       []int{},
			f:        strconv.Itoa,
			expected: []string{},
		},
		{
			name:     "should return empty slice for nil input",
			in:       nil,
			f:        strconv.Itoa,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Map(tt.in, tt.f)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		in       []int
		f        func(int) bool
		expected []int
	}{
		{
			name:     "should return even numbers for mixed slice",
			in:       []int{1, 2, 3, 4, 5},
			f:        func(i int) bool { return i%2 == 0 },
			expected: []int{2, 4},
		},
		{
			name:     "should return empty slice when no elements match",
			in:       []int{1, 3, 5},
			f:        func(i int) bool { return i%2 == 0 },
			expected: []int{},
		},
		{
			name:     "should return empty slice for empty input",
			in:       []int{},
			f:        func(i int) bool { return i%2 == 0 },
			expected: []int{},
		},
		{
			name:     "should return empty slice for nil input",
			in:       nil,
			f:        func(i int) bool { return i%2 == 0 },
			expected: []int{},
		},
		{
			name:     "should return all elements when all match",
			in:       []int{2, 4, 6},
			f:        func(i int) bool { return i%2 == 0 },
			expected: []int{2, 4, 6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Filter(tt.in, tt.f)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestReduce(t *testing.T) {
	t.Parallel()

	tests := []struct {
		f        func(int, int) int
		name     string
		in       []int
		initial  int
		expected int
	}{
		{
			name:     "should sum all elements for non-empty slice",
			in:       []int{1, 2, 3, 4},
			f:        func(i, acc int) int { return acc + i },
			initial:  0,
			expected: 10,
		},
		{
			name:     "should return initial value for empty slice",
			in:       []int{},
			f:        func(i, acc int) int { return acc + i },
			initial:  5,
			expected: 5,
		},
		{
			name:     "should return initial value for nil input",
			in:       nil,
			f:        func(i, acc int) int { return acc + i },
			initial:  7,
			expected: 7,
		},
		{
			name:     "should multiply all elements for non-empty slice",
			in:       []int{2, 3, 4},
			f:        func(i, acc int) int { return acc * i },
			initial:  1,
			expected: 24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Reduce(tt.in, tt.f, tt.initial)
			assert.Equal(t, tt.expected, got)
		})
	}
}
