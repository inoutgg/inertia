# inertia

Inertia is an adapter for the [Inertia.js](https://inertiajs.com/) library adapted for the Go.

The library is designed to feel natural and look nature to Go developers.
As such the API diverges from the reference implementation of Inertia in PHP.

## Shared props

While this library does not provide first class support for shared props, similar to the one provided in the PHP reference implementation, it is possible to achieve it by using the `inertia.WithProps` with pre-defined props.

```go
import (
	"github.com/inoutgg/inertia"
)

func WithDefaultProps(props ...[]*inertia.Prop) inertia.Option {
	defaultProps := []*Prop{
		inertia.NewProp("appVersion", "1.0.0", nil),
		inertia.NewProp("appName", "ACME Inc.", nil),
	}

	return inertia.WithProps(append(defaultProps, props...)...)
}
```

## Vite

This library provides a minimal adapter for the [Vite](https://vitejs.dev/) build tool available via `go.inout.gg/intertia/contrib/vite`.

## License

MIT licensed
