package inertia

import (
	"net/http"

	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
)

// ErrorBagFromRequest extracts the error bag name from the X-Inertia-Error-Bag header.
//
// Returns DefaultErrorBag if the header is not present.
// Used to scope validation errors to specific forms on a page.
func ErrorBagFromRequest(r *http.Request) string {
	errorBag := r.Header.Get(inertiaheader.HeaderXInertiaErrorBag)
	if errorBag == "" {
		return DefaultErrorBag
	}

	return errorBag
}
