package inertia

import (
	"context"
	"net/http"
	"slices"

	"go.inout.gg/foundations/http/httpmiddleware"
	"go.inout.gg/inertia/internal/inertiaheader"
)

// https://inertiajs.com/redirects#303-response-code
var seeOtherMethods = []string{http.MethodPatch, http.MethodPut, http.MethodDelete}

type MiddlewareConfig struct {
	// HandleEmptyResponse is a function that is called when the response is empty.
	HandleEmptyResponse http.HandlerFunc

	// HandleVersionMismatch is a function that is called when the version mismatch occurs.
	HandleVersionMismatch http.HandlerFunc
}

func (m *MiddlewareConfig) defaults() {
	if m.HandleEmptyResponse == nil {
		m.HandleEmptyResponse = func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Empty response", http.StatusNoContent)
		}
	}
	if m.HandleVersionMismatch == nil {
		m.HandleVersionMismatch = func(w http.ResponseWriter, r *http.Request) {
			Location(w, r, r.RequestURI)
		}
	}
}

// Middleware provides the HTTP handling layer for Inertia.js server-side integration.
func Middleware(renderer *Renderer, opts ...func(*MiddlewareConfig)) httpmiddleware.MiddlewareFunc {
	config := MiddlewareConfig{}
	for _, opt := range opts {
		opt(&config)
	}

	config.defaults()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(inertiaheader.HeaderVary, inertiaheader.HeaderXInertia)
			r = r.WithContext(context.WithValue(r.Context(), kCtxKey, renderer))

			if !isInertiaRequest(r) {
				next.ServeHTTP(w, r)
				return
			}

			// externalVersion := r.Header.Get(headerXInertiaVersion)
			// if externalVersion != renderer.Version() {
			// 	Location(w, r, r.RequestURI)
			// 	return
			// }

			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			defer func() {
				if err := wrapped.flush(); err != nil {
					// TODO: handle error
				}
			}()

			next.ServeHTTP(wrapped, r)

			if wrapped.statusCode == http.StatusFound &&
				slices.Contains(seeOtherMethods, r.Method) {
				wrapped.WriteHeader(http.StatusSeeOther)
			}

			// if wrapped.Empty() {
			// 	config.HandleEmptyResponse(wrapped, r)
			// 	return
			// }
		})
	}
}
