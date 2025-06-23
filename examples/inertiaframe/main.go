package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"go.inout.gg/examples/inertiaframe/endpoint"
	"go.inout.gg/foundations/must"

	"go.inout.gg/inertia"
	"go.inout.gg/inertia/contrib/vite"
	"go.inout.gg/inertia/inertiaframe"
)

//nolint:exhaustruct,gochecknoglobals
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

	{{viteResource "src/main.tsx"}}
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

	inertiaframe.Mount(mux, &endpoint.SignInGetEndpoint{}, nil)
	inertiaframe.Mount(mux, &endpoint.SignInPostEndpoint{}, nil)

	go func() {
		//nolint:gosec
		must.Must1(http.ListenAndServe(":8080", middleware.Middleware(mux)))
	}()

	log.InfoContext(ctx, "Server is running", slog.String("addr", "http://localhost:8080"))
	<-ctx.Done()

	log.InfoContext(ctx, "Shutting down server")
}
