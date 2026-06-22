# inertia

Inertia is an adapter for the [Inertia.js](https://inertiajs.com/) library adapted for the Go.

The library is designed to feel and look nature to Go developers.
As such the API diverges from the reference implementation of Inertia in PHP.

The package also exposes an opinionated framework that works on top of the
inertia adapter -- inertiaframe.

The inertiaframe abstracts away raw request and response via
inertia-specific messages.

## Vite

This library bundles a minimal adapter for the [Vite](https://vitejs.dev/) build tool available via `go.segfaultmedaddy.com/inertia/contrib/vite`.

## License

MIT licensed
