# inertia

Build single-page apps with a Go backend and zero API boilerplate.

`inertia` is a Go adapter for [Inertia.js](https://inertiajs.com/) — you keep
server-side routing and controllers, render React/Vue/Svelte components from Go,
and skip the JSON API layer entirely. Wire up your `*http.ServeMux`, render
components with typed props, and let the adapter handle the Inertia.js protocol.

## Features

- **Inertia.js protocol** — request/response handling, partial reloads,
  redirect upgrades, and asset-version checks out of the box.
- **Typed props** — `inertiaprop`, `inertiadeferred`, `inertiascroll`,
  `inertiaalways`, `inertiaoptional`, `inertiaonce`, `inertiamerge`:
  compose props with merge, once, deferral, scroll, and concurrency behavior.
- **`inertiaframe`** — a framework layer with type-safe `Endpoint[M]` handlers,
  automatic request parsing, a pluggable `Validator[M]`. You return a `Response`; it does the rest.
- **`contrib/vite`** — minimal [Vite](https://vitejs.dev/) adapter for dev/PROD
  asset tags and manifest parsing.
- **OpenTelemetry** — built-in tracing and metrics via `inertiaotel`.

## Quick start: `inertiaframe`

```go
package main

import (
	"context"
	"net/http"

	"go.segfaultmedaddy.com/inertia"
	"go.segfaultmedaddy.com/inertia/contrib/vite"
	"go.segfaultmedaddy.com/inertia/inertiaframe"
	"go.segfaultmedaddy.com/inertia/inertiaprop"
)

// CreateUserInput is the parsed request message.
type CreateUserInput struct {
	Name  string `form:"name"`
	Email string `form:"email"`
}

// validate implements inertiaframe.Validator[CreateUserInput]. Returning
// inertia.ValidationErrors routes the field-level errors back to the client
// through the Inertia.js validation flow (flash session + redirect back).
func validate(in CreateUserInput) error {
	var errs inertia.ValidationErrors
	if in.Name == "" {
		errs = append(errs, inertia.NewValidationError("name", "required"))
	}
	if in.Email == "" {
		errs = append(errs, inertia.NewValidationError("email", "required"))
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// createUserEndpoint implements inertiaframe.Endpoint[CreateUserInput].
type createUserEndpoint struct{}

func (createUserEndpoint) Meta() inertiaframe.Meta {
	return inertiaframe.Meta{Method: http.MethodPost, Path: "/users"}
}

func (createUserEndpoint) Execute(
	_ context.Context,
	req *inertiaframe.Request[CreateUserInput],
) (inertiaframe.Response, error) {
	user := req.Message // already parsed and validated

	props := inertia.Props{
		inertiaprop.New("user", map[string]string{
			"name":  user.Name,
			"email": user.Email,
		}),
	}

	return inertiaframe.NewResponse("Users/Show", props), nil
}

func main() {
	mux := http.NewServeMux()

	app := inertiaframe.New(mux, &inertiaframe.Config{})

	// Mount the endpoint; pass a *MountConfig to add a validator or middleware.
	inertiaframe.Mount(app, createUserEndpoint{}, &inertiaframe.MountConfig[CreateUserInput]{
		Validator: inertiaframe.ValidatorFunc[CreateUserInput](validate),
	})

	// Wrap the mux in the Inertia.js protocol middleware.
	handler := inertia.NewMiddleware(
		inertia.New(vite.MustTemplate(appHTML, &vite.Config{}), &inertia.Config{
			Version: "1.0.0",
		}),
	)(mux)

	_ = http.ListenAndServe(":8080", handler)
}
```

Prefer the core adapter with no framework? Use `inertia.Render` directly in
your handlers and `inertia.NewMiddleware` to wrap the mux:

```go
package main

import (
	"net/http"

	"go.segfaultmedaddy.com/inertia"
	"go.segfaultmedaddy.com/inertia/inertiaprop"
)

func main() {
	renderer := inertia.MustFromFS(assetsFS, "app.tmpl", &inertia.Config{
		Version: "1.0.0",
	})

	mux := http.NewServeMux()

	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		props := inertia.Props{
			inertiaprop.New("user", map[string]string{
				"id":   r.PathValue("id"),
				"name": "Roman",
			}),
		}

		rctx := inertia.NewRenderContext(inertia.WithProps(props))
		if err := inertia.Render(w, r, "Users/Show", rctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	_ = http.ListenAndServe(":8080", inertia.NewMiddleware(renderer)(mux))
}
```

## License

MIT licensed.
