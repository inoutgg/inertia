package main

import (
	"log/slog"
	"net/http"
	"os"

	"go.inout.gg/foundations/must"
	"go.inout.gg/inertia"
	"go.inout.gg/inertia/contrib/vite"
)

var log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

const rootTemplate = `<!doctype html>
<html>
<head>
	{{.InertiaHead}}
</head>
<body>
	{{.InertiaBody}}
	{{template "viteClient"}}
	{{template "viteReactRefresh"}}

	{{viteResource "main.tsx"}}
</body>
</html>`

func main() {
	renderer := inertia.New(
		must.Must(vite.NewTemplate(rootTemplate, nil)),
		nil,
	)
	middleware := inertia.Middleware(renderer)

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inertia.MustRender(w, r, "Index")
	}))

	must.Must1(http.ListenAndServe(":8080", middleware.Middleware(mux)))
}
