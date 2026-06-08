package inertiaonce

import "go.segfaultmedaddy.com/inertia/internal/inertiaprop"

// OnceOpts configures the once-fetch behavior for a prop built with
// inertiaprop.WithOnce; chain Key, ExpiresAt, and Fresh on the returned
// value to set the cache key, expiration timestamp, and force-refresh flag.
type OnceOpts = inertiaprop.OnceOpts

// NewOnceOpts returns a OnceOpts; configure it via the chainable Key,
// ExpiresAt, and Fresh methods before passing it to inertiaprop.WithOnce.
func NewOnceOpts() *OnceOpts { return inertiaprop.NewOnceOpts() }
