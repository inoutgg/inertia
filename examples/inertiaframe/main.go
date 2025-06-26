package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"go.inout.gg/examples/inertiaframe/endpoint"
	"go.inout.gg/foundations/http/httpmiddleware"
	"go.inout.gg/foundations/must"
	"go.inout.gg/shield/shieldcsrf"

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
	csrf := must.Must(shieldcsrf.Middleware("secret"))

	inertiaframe.Mount(mux, &endpoint.SignInGetEndpoint{}, &inertiaframe.MountOpts{
		Middleware: httpmiddleware.NewChain(csrf, httpmiddleware.MiddlewareFunc(func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tok := must.Must(shieldcsrf.FromRequest(r))
				shieldcsrf.SetToken(w, tok)
				h.ServeHTTP(w, r)
			})
		})),
	})
	inertiaframe.Mount(mux, &endpoint.SignInPostEndpoint{}, &inertiaframe.MountOpts{
		Middleware: httpmiddleware.NewChain(csrf),
	})

	go func() {
		//nolint:gosec
		must.Must1(http.ListenAndServe(":8080", middleware.Middleware(mux)))
	}()

	log.InfoContext(ctx, "Server is running", slog.String("addr", "http://localhost:8080"))
	<-ctx.Done()

	log.InfoContext(ctx, "Shutting down server")
}
