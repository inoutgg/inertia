package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.inout.gg/examples/inertiaframe/endpoint"
	"go.inout.gg/examples/inertiaframe/user"
	"go.inout.gg/foundations/http/httperror"
	"go.inout.gg/foundations/http/httpmiddleware"
	"go.inout.gg/foundations/must"
	"go.inout.gg/shield/shieldcsrf"
	"go.inout.gg/shield/shieldmigrate"
	"go.inout.gg/shield/shieldpassword"
	"go.inout.gg/shield/shieldstrategy/session"
	"go.inout.gg/shield/shielduser"

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

func setupPool(ctx context.Context, logger *slog.Logger, url string) *pgxpool.Pool {
	logger.InfoContext(ctx, "setting up database pool")

	pool := must.Must(pgxpool.New(ctx, url))

	conn := must.Must(pool.Acquire(ctx))
	defer conn.Release()

	logger.InfoContext(ctx, "running migrations")

	must.Must1(shieldmigrate.New().Up(ctx, conn.Conn(), nil))

	return pool
}

func main() {
	fmt.Println("running")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Init resources
	pool := setupPool(ctx, logger, "postgres://local:local@localhost:5432/local")

	renderer := inertia.New(
		must.Must(vite.NewTemplate(rootTemplate, nil)),
		nil,
	)
	middleware := inertia.Middleware(renderer)

	mux := http.NewServeMux()
	csrf := must.Must(shieldcsrf.Middleware("secret"))
	passwordHandler := shieldpassword.NewHandler[user.Data](pool, nil)
	authenticator := session.New[user.Data](pool)

	inertiaframe.Mount(mux, &endpoint.SignInGetEndpoint{}, &inertiaframe.MountOpts{
		Middleware: httpmiddleware.NewChain(csrf, httpmiddleware.MiddlewareFunc(func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tok := must.Must(shieldcsrf.FromRequest(r))
				shieldcsrf.SetToken(w, tok)
				h.ServeHTTP(w, r)
			})
		})),
	})
	inertiaframe.Mount(mux, &endpoint.SignInPostEndpoint{
		Handler:       passwordHandler,
		Authenticator: authenticator,
	}, &inertiaframe.MountOpts{
		Middleware: httpmiddleware.NewChain(csrf),
	})

	inertiaframe.Mount(mux, &endpoint.SignUpGetEndpoint{}, &inertiaframe.MountOpts{
		Middleware: httpmiddleware.NewChain(csrf, httpmiddleware.MiddlewareFunc(func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tok := must.Must(shieldcsrf.FromRequest(r))
				shieldcsrf.SetToken(w, tok)
				h.ServeHTTP(w, r)
			})
		})),
	})
	inertiaframe.Mount(mux, &endpoint.SignUpPostEndpoint{
		Handler:       passwordHandler,
		Authenticator: authenticator,
	}, &inertiaframe.MountOpts{
		Middleware: httpmiddleware.NewChain(csrf),
	})

	inertiaframe.Mount(mux, &endpoint.HomeEndpoint{}, &inertiaframe.MountOpts{
		Middleware: httpmiddleware.MiddlewareFunc(
			shielduser.Middleware[user.Data](authenticator, httperror.DefaultErrorHandler, nil)),
	})

	go func() {
		//nolint:gosec
		must.Must1(http.ListenAndServe(":8080", middleware.Middleware(mux)))
	}()

	log.InfoContext(ctx, "Server is running", slog.String("addr", "http://localhost:8080"))
	<-ctx.Done()

	log.InfoContext(ctx, "Shutting down server")
}
