package inertia

import (
	"go.segfaultmedaddy.com/inertia/internal/inertiahttp"
)

// ErrInvalidInertiaRequest is the sentinel error returned (or wrapped) when an
// incoming Inertia request has malformed or invalid protocol headers.
// Callers can match it with errors.Is to distinguish protocol errors from
// other parse failures.
var ErrInvalidInertiaRequest = inertiahttp.ErrInvalidInertiaRequest
