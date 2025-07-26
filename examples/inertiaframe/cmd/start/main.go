package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.inout.gg/foundations/must"
	"go.inout.gg/inertia"
	"go.inout.gg/inertia/contrib/vite"
	"go.inout.gg/inertia/inertiaframe"
	"go.inout.gg/shield/shieldmigrate"
	"go.inout.gg/shield/shieldpassword"
	"go.inout.gg/shield/shieldsession"
	"go.inout.gg/shield/shieldsession/serversession"

	"go.inout.gg/examples/inertiaframe/endpoint"
	"go.inout.gg/examples/inertiaframe/sender"
	"go.inout.gg/examples/inertiaframe/user"
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

	authenticator := serversession.New[user.Info, any](pool, nil)
	passwordHandler := shieldpassword.NewHandler(pool, authenticator, sender.New(), nil)

	unprotectedMux := chi.NewRouter().With(
		shieldsession.Middleware(
			authenticator,
			inertiaframe.DefaultErrorHandler,
			shieldsession.NewConfig(shieldsession.WithPassthrough()),
		),
		shieldsession.RedirectAuthenticatedUserMiddleware("/-/dashboard"),
	)

	protectedMux := chi.NewRouter().With(
		shieldsession.Middleware(authenticator, inertiaframe.DefaultErrorHandler, nil),
	)

	logoutHandler := serversession.NewLogoutHandler[user.Info, any](pool, nil)

	// Sign up
	inertiaframe.Mount(unprotectedMux, &endpoint.SignUpGetEndpoint{}, nil)
	inertiaframe.Mount(unprotectedMux, &endpoint.SignUpPostEndpoint{
		Handler:       passwordHandler,
		Authenticator: authenticator,
	}, nil)

	// Sign in
	inertiaframe.Mount(unprotectedMux, &endpoint.SignInGetEndpoint{}, nil)
	inertiaframe.Mount(unprotectedMux, &endpoint.SignInPostEndpoint{
		Handler:       passwordHandler,
		Authenticator: authenticator,
	}, nil)

	// Sign out
	protectedMux.Get("/logout", func(w http.ResponseWriter, r *http.Request) {
		err := logoutHandler.Logout(w, r)
		if err != nil {
			inertiaframe.DefaultErrorHandler.ServeHTTP(w, r, err)
		}

		http.Redirect(w, r, "/", http.StatusFound)
	})

	// Home
	inertiaframe.Mount(protectedMux, &endpoint.HomeEndpoint{}, nil)

	mux := chi.NewMux()

	mux.Mount("/", unprotectedMux)
	mux.Mount("/-/", protectedMux)

	go func() {
		//nolint:gosec
		must.Must1(http.ListenAndServe(":8080", middleware.Middleware(mux)))
	}()

	log.InfoContext(ctx, "Server is running", slog.String("addr", "http://localhost:8080"))
	<-ctx.Done()

	log.InfoContext(ctx, "Shutting down server")
}
