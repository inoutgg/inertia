package inertia

import (
	"html/template"
	"net/http"
	"testing"

	"go.inout.gg/inertia/internal/inertiatest"
)

var tpl = template.Must(template.New("<inertia-test>").Parse(`<!doctype html>
<html>
<head>{{ .InertiaHead }}</head>
<body>{{ .InertiaBody }}</body>
</html>
`))

func testHandler(w http.ResponseWriter, r *http.Request) {
	Render(w, r, "inertia", nil)
}

func newMiddleware(h http.Handler, renderer *Renderer) http.Handler {
	if renderer == nil {
		renderer = New(tpl, nil)
	}
	mux := http.NewServeMux()
	middleware := Middleware(renderer)(mux)

	mux.HandleFunc("/inertia", h.ServeHTTP)

	return middleware
}

func TestMiddleware_RedirectToSeeOther(t *testing.T) {
	t.Parallel()

	redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/somewhere", http.StatusFound)
	})

	testCases := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{"PATCH should redirect with 303", http.MethodPatch, http.StatusSeeOther},
		{"PUT should redirect with 303", http.MethodPut, http.StatusSeeOther},
		{"DELETE should redirect with 303", http.MethodDelete, http.StatusSeeOther},
		{"GET should redirect with 302", http.MethodGet, http.StatusFound},
		{"POST should redirect with 302", http.MethodPost, http.StatusFound},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r, w := inertiatest.NewRequest(tc.method, "/inertia", &inertiatest.RequestConfig{
				Inertia: true,
			})

			middleware := newMiddleware(redirectHandler, nil)
			middleware.ServeHTTP(w, r)

			if w.Code != tc.expectedStatus {
				t.Errorf("expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			location := w.Header().Get("Location")
			if location != "/somewhere" {
				t.Errorf("expected Location header to be '/somewhere', got %q", location)
			}
		})
	}
}
