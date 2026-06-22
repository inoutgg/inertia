package inertiahttp

import (
	"context"
	"net/http"
)

type ctxKey struct{}

//nolint:gochecknoglobals
var kCtxKey = ctxKey{}

// WithRenderScope returns a copy of r with scope stored in its context,
// making it retrievable later via RenderScopeFromRequest.
//
// This allows to parse request once per its lifecycle allowing downstream consumers
// (such as Render or inertiaframe) to render the response without re-parsing
// the request.
func WithRenderScope(r *http.Request, scope *RenderScope) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), kCtxKey, scope))
}

// RenderScopeFromRequest returns the RenderScope previously attached to r
// by WithRenderScope.
//
// It panics if no scope is present, which indicates that the inertia
// middleware was not installed in the handler chain serving r.
func RenderScopeFromRequest(r *http.Request) *RenderScope {
	return r.Context().Value(kCtxKey).(*RenderScope) //nolint:forcetypeassert
}
