package inertiahttp

import (
	"context"
	"net/http"
)

type ctxKey struct{}

//nolint:gochecknoglobals
var kCtxKey = ctxKey{}

func RenderScopeFromRequest(r *http.Request) *RenderScope {
	return r.Context().Value(kCtxKey).(*RenderScope) //nolint:forcetypeassert
}

func WithRenderScope(r *http.Request, scope *RenderScope) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), kCtxKey, scope))
}
