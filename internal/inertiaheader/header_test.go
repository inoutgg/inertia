package inertiaheader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestV3Headers(t *testing.T) {
	t.Parallel()

	// arrange
	expectedHeaders := map[string]string{
		"redirect":    "X-Inertia-Redirect",
		"scrollMerge": "X-Inertia-Infinite-Scroll-Merge-Intent",
		"onceProps":   "X-Inertia-Except-Once-Props",
	}

	// act
	actualHeaders := map[string]string{
		"redirect":    HeaderXInertiaRedirect,
		"scrollMerge": HeaderXInertiaScrollMerge,
		"onceProps":   HeaderXInertiaExceptOnceProps,
	}

	// assert
	assert.Equal(t, expectedHeaders, actualHeaders)
}
