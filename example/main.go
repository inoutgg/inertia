package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"go.inout.gg/foundations/must"
	"go.inout.gg/inertia"
	"go.inout.gg/inertia/contrib/inertiavalidationerrors"
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
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	renderer := inertia.New(
		must.Must(vite.NewTemplate(rootTemplate, nil)),
		nil,
	)
	middleware := inertia.Middleware(renderer)

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inertia.MustRender(w, r, "Index", inertia.WithProps(inertia.Props{
			inertia.NewProp("key", "val", nil),
		}))
	}))

	mux.Handle("/createTodo", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inertia.MustRender(w, r, "Index", inertia.WithValidationErrors(inertia.WithErrorBag(
			"createTodo", inertiavalidationerrors.Map{
				"title": "Title is required",
			})))
	}))

	go func() {
		must.Must1(http.ListenAndServe(":8080", middleware.Middleware(mux)))
	}()

	log.Info("Server is running", slog.String("addr", "http://localhost:8080"))
	<-ctx.Done()

	log.Info("Shutting down server")
}
