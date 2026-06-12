# Inertia.js Go Adapter — Metrics & Traces

This document tells you exactly what metrics and traces to add, and where to add them. It assumes you already use `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` for HTTP request/response instrumentation. Only Inertia-specific signals are covered here.

All names follow the [OpenTelemetry semantic convention naming guidelines](https://opentelemetry.io/docs/specs/semconv/general/naming/):

- Lowercase only
- Dots separate namespaces
- Underscores separate words within a name
- Names are descriptive and not abbreviated

## Enabling Telemetry

Telemetry is **disabled by default**. Pass a `TracerProvider` and/or `MeterProvider` to enable it. Each layer of the library has its own config field.

### Renderer

```go
import "go.opentelemetry.io/otel/sdk/trace"
import "go.opentelemetry.io/otel/sdk/metric"

renderer := inertia.New(t, &inertia.Config{
    Telemetry: inertia.TelemetryConfig{
        TracerProvider: tp,  // trace.TracerProvider
        MeterProvider: mp,   // metric.MeterProvider
    },
})
```

### Middleware

```go
middleware := inertia.NewMiddleware(renderer, func(c *inertia.MiddlewareConfig) {
    c.Telemetry.TracerProvider = tp
    c.Telemetry.MeterProvider = mp
})
```

### inertiaframe

```go
inertiaframe.Mount(mux, endpoint, &inertiaframe.MountConfig[MyMsg]{
    Telemetry: inertia.TelemetryConfig{
        TracerProvider: tp,
        MeterProvider: mp,
    },
})
```

If both fields are nil, the library does not create spans or record metrics. There is no overhead.

## Cardinality Rules

- **Metrics:** Never add `inertia.component` or `inertia.prop.key` as metric dimensions. These are high-cardinality and will explode your time-series storage.
- **Traces:** High-cardinality attributes are fine on spans. `inertia.component` is the single most useful trace attribute.
- **Histograms:** Keep dimensions minimal. One or two low-cardinality dimensions per histogram is the maximum.

## otelutil Package

The `otelutil` package (`go.segfaultmedaddy.com/inertia/otelutil`) provides a nil-safe wrapper around OpenTelemetry providers. You never need to check `if tc.TracerProvider != nil` or `var span trace.Span`. The wrapper handles all of that internally.

```go
import "go.segfaultmedaddy.com/inertia/otelutil"
```

### Key API

```go
type TelemetryConfig struct {
    TracerProvider trace.TracerProvider
    MeterProvider  metric.MeterProvider
}

// Defaults sets no-op providers for any nil fields.
// Call this once before using the config.
func (t *TelemetryConfig) Defaults()

// Span creates a span. The returned trace.Span is always valid.
// If tracing is disabled, it is a no-op from the standard OpenTelemetry noop
// tracer. Always call defer span.End() — it does nothing when telemetry is off.
func (t *TelemetryConfig) Span(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span)

// Int64Counter returns a counter. Always valid — no-op when telemetry is off.
func (t *TelemetryConfig) Int64Counter(name string, opts ...metric.Int64CounterOption) metric.Int64Counter

// Float64Histogram returns a histogram. Always valid — no-op when telemetry is off.
func (t *TelemetryConfig) Float64Histogram(name string, opts ...metric.Float64HistogramOption) metric.Float64Histogram
```

---

## 1. `middleware.go` — `NewMiddleware`

### Traces

Set attributes on the existing otelhttp server span. Do not create a new span.

```go
func NewMiddleware(renderer *Renderer, opts ...func(*MiddlewareConfig)) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            req, err := parseRequest(r)

            span := trace.SpanFromContext(r.Context())
            span.SetAttributes(
                attribute.String("inertia.request.type", requestType(req)),
                attribute.Bool("inertia.version.mismatch",
                    req.IsInertia && req.Version != renderer.Version()),
            )

            if err != nil {
                span.AddEvent("inertia.request.invalid",
                    trace.WithAttributes(attribute.String("error", err.Error())))
            }

            if err != nil {
                config.InvalidRequestHandler(w, r, err)
                return
            }

            // ... rest of middleware
        })
    }
}
```

**Attributes on the otelhttp span:**

| Attribute                  | Type   | Values                                      | Why                                                                              |
| -------------------------- | ------ | ------------------------------------------- | -------------------------------------------------------------------------------- |
| `inertia.request.type`     | string | `full`, `partial`, `initial`, `non_inertia` | What the client is doing. Partial reloads behave differently from full loads.    |
| `inertia.version.mismatch` | bool   | `true`, `false`                             | `true` means the client forced a full reload. Indicates asset deployment issues. |

### Metrics

No metrics needed here. `otelhttp` already counts requests and measures latency. The `inertia.request.type` attribute on the trace is sufficient for debugging.

---

## 2. `renderer.go` — `Renderer.render()`

### Traces

Create the `inertia.render` span. No nil checks, no `var span trace.Span`.

```go
func (r *Renderer) render(ctx context.Context, req request, name string, renderCtx RenderContext) (response, error) {
    ctx, span := r.Telemetry.Span(ctx, "inertia.render",
        trace.WithSpanKind(trace.SpanKindInternal))
    defer span.End()

    span.SetAttributes(
        attribute.String("inertia.component", name),
        attribute.String("inertia.request.type", requestType(req)),
        attribute.String("inertia.render.type", renderType(req)),
        attribute.Bool("inertia.ssr.enabled", r.ssrClient != nil),
    )

    // ... prop resolution ...

    span.AddEvent("inertia.render.props.resolved",
        trace.WithAttributes(
            attribute.Int("inertia.props.resolved_count", len(props)),
            attribute.Int("inertia.props.rescued_count", len(rescuedProps)),
        ))

    // ... render ...
}
```

**Span attributes:**

| Attribute              | Type   | Values                                      | Why                                                                                 |
| ---------------------- | ------ | ------------------------------------------- | ----------------------------------------------------------------------------------- |
| `inertia.component`    | string | Component name                              | **High cardinality — trace only.** If one page is slow, you need to know which one. |
| `inertia.request.type` | string | `full`, `partial`, `initial`, `non_inertia` | Same as middleware.                                                                 |
| `inertia.render.type`  | string | `json`, `html`                              | JSON is fast; HTML involves template rendering and SSR.                             |
| `inertia.ssr.enabled`  | bool   | `true`, `false`                             | Whether SSR is configured for this render.                                          |

**Span event:**

| Event                           | Attributes                                                    | Why                                                                                     |
| ------------------------------- | ------------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| `inertia.render.props.resolved` | `inertia.props.resolved_count`, `inertia.props.rescued_count` | One event per render. Captures prop resolution outcome without creating per-prop spans. |

### Metrics

Add a histogram measuring total render time, dimensioned only by `inertia.request.type`. The histogram is a no-op when telemetry is disabled.

```go
var renderDuration = r.Telemetry.Float64Histogram("inertia.render.duration",
    metric.WithUnit("ms"),
    metric.WithDescription("Duration of Inertia page rendering"),
)

// In render()
start := time.Now()
// ... render logic ...
renderDuration.Record(ctx, float64(time.Since(start).Milliseconds()),
    metric.WithAttributes(attribute.String("inertia.request.type", requestType(req))),
)
```

| Metric                    | Type      | Dimensions             | Why                                                                                       |
| ------------------------- | --------- | ---------------------- | ----------------------------------------------------------------------------------------- |
| `inertia.render.duration` | Histogram | `inertia.request.type` | Core SLO metric. Partial reloads are much faster than full loads, so you need this split. |

---

## 3. `internal/inertiaprotocol/render.go` — `resolveProps()`

### Traces

Create the `inertia.props.resolve` span. Only create the `inertia.props.resolve.concurrent` span if there are actually concurrent props.

The `TelemetryConfig` should be passed through `context.Context` (e.g., `ctx = context.WithValue(ctx, telemetryKey, tc)`) so the pool workers can access it.

```go
func resolveProps(ctx context.Context, req Request, componentName string, props []inertiaprop.Prop, pool pond.ResultPool[Result]) (map[string]any, []string, error) {
    tc := telemetryFromContext(ctx) // retrieves TelemetryConfig from context

    ctx, span := tc.Span(ctx, "inertia.props.resolve",
        trace.WithSpanKind(trace.SpanKindInternal))
    defer span.End()

    span.SetAttributes(
        attribute.Int("inertia.props.total", len(props)),
        attribute.Int("inertia.props.concurrent", len(concurrentProps)),
    )

    // ... resolve synchronous props ...

    if len(concurrentProps) > 0 {
        ctx, span := tc.Span(ctx, "inertia.props.resolve.concurrent",
            trace.WithSpanKind(trace.SpanKindInternal))
        defer span.End()

        group := pool.NewGroupContext(ctx)
        // ... submit tasks with ctx ...
    }

    // ...
}
```

**Span attributes:**

| Attribute                  | Type | Values  | Why                              |
| -------------------------- | ---- | ------- | -------------------------------- |
| `inertia.props.total`      | int  | `0`–`N` | How many props total.            |
| `inertia.props.concurrent` | int  | `0`–`N` | How many are in the worker pool. |

### Metrics

Add a histogram for concurrent prop resolution. Only recorded when there are concurrent props. The histogram is a no-op when telemetry is disabled.

```go
var concurrentPropsDuration = tc.Float64Histogram("inertia.props.concurrent.resolution.duration",
    metric.WithUnit("ms"),
    metric.WithDescription("Duration of concurrent prop resolution"),
)

// In the concurrent block
start := time.Now()
// ... group.Wait() ...
concurrentPropsDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
```

Add a counter for rescued props. The counter is a no-op when telemetry is disabled.

```go
var rescuedProps = tc.Int64Counter("inertia.props.rescued",
    metric.WithDescription("Number of rescued deferred props"),
)

// When a prop is rescued
rescuedProps.Add(ctx, 1)
```

| Metric                                         | Type      | Dimensions | Why                                                                      |
| ---------------------------------------------- | --------- | ---------- | ------------------------------------------------------------------------ |
| `inertia.props.concurrent.resolution.duration` | Histogram | none       | If this spikes, your worker pool is saturated or the resolvers are slow. |
| `inertia.props.rescued`                        | Counter   | none       | Rescued props mean deferred props failed silently. Non-zero is a signal. |

---

## 4. `internal/inertiassr/ssr.go` — `ssr.Render()`

### Traces

Create the `inertia.render.ssr` span. If the SSR `http.Client` is wrapped with `otelhttp.NewTransport`, the HTTP client span is already created automatically — do not duplicate it.

```go
func (s *ssr) Render(ctx context.Context, p *inertiaprotocol.Page, tc inertia.TelemetryConfig) (*SSRTemplateData, error) {
    ctx, span := tc.Span(ctx, "inertia.render.ssr",
        trace.WithSpanKind(trace.SpanKindInternal))
    defer span.End()

    // ... marshal page ...
    // ... otelhttp handles the HTTP client span ...

    resp, err := s.client.Do(r)
    if err != nil {
        span.AddEvent("inertia.render.ssr.fallback",
            trace.WithAttributes(attribute.String("error", err.Error())))
        span.SetAttributes(attribute.Bool("inertia.ssr.fallback", true))
        return nil, err
    }

    // ...
}
```

**Span attributes:**

| Attribute              | Type | Values          | Why                                    |
| ---------------------- | ---- | --------------- | -------------------------------------- |
| `inertia.ssr.fallback` | bool | `true`, `false` | `true` if SSR failed and CSR was used. |

**Span event:**

| Event                         | Attributes | Why                                      |
| ----------------------------- | ---------- | ---------------------------------------- |
| `inertia.render.ssr.fallback` | `error`    | Only fires when SSR fails. Captures why. |

### Metrics

Add a counter for SSR fallbacks. The counter is a no-op when telemetry is disabled.

```go
var ssrFallback = tc.Int64Counter("inertia.ssr.fallback",
    metric.WithDescription("Number of SSR fallbacks to client-side rendering"),
)

// On error
ssrFallback.Add(ctx, 1)
```

| Metric                 | Type    | Dimensions | Why                                          |
| ---------------------- | ------- | ---------- | -------------------------------------------- |
| `inertia.ssr.fallback` | Counter | none       | Non-zero means the SSR service is unhealthy. |

---

## 5. `inertiaframe/inertiaframe.go` — `newHandler()`

### Traces

Create the `inertiaframe.endpoint` span. The `inertiaframe.execute` span is optional — only add it if you need to separate user code from framework overhead.

```go
func newHandler[M any](endpoint Endpoint[M], errorHandler httphandler.ErrorHandler, validator Validator[M], formDecoder *form.Decoder, jsonUnmarshalOptions []json.Options, tc inertia.TelemetryConfig) http.Handler {
    return handleError(httphandler.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
        ctx := r.Context()

        ctx, span := tc.Span(ctx, "inertiaframe.endpoint",
            trace.WithSpanKind(trace.SpanKindInternal))
        defer span.End()

        span.SetAttributes(
            attribute.String("inertiaframe.endpoint.method", endpoint.Meta().Method),
            attribute.String("inertiaframe.endpoint.path", endpoint.Meta().Path),
        )

        // ... parse, validate ...

        ctx, execSpan := tc.Span(ctx, "inertiaframe.execute",
            trace.WithSpanKind(trace.SpanKindInternal))
        resp, err := endpoint.Execute(ctx, newRequest(msg))
        execSpan.End()

        // ...

        span.SetAttributes(
            attribute.String("inertiaframe.response.type", responseType(resp)),
        )

        // ...
    }))
}
```

**Span attributes:**

| Attribute                    | Type   | Values                                                             | Why                                                                            |
| ---------------------------- | ------ | ------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `inertiaframe.response.type` | string | `inertia`, `raw`, `redirect`, `redirect_back`, `external_redirect` | What kind of response the endpoint produced. Useful for routing and debugging. |

### Metrics

Add a histogram for endpoint execution time. The histogram is a no-op when telemetry is disabled.

```go
var endpointDuration = tc.Float64Histogram("inertiaframe.endpoint.duration",
    metric.WithUnit("ms"),
    metric.WithDescription("Duration of inertiaframe endpoint execution"),
)

// Around endpoint.Execute()
start := time.Now()
resp, err := endpoint.Execute(ctx, newRequest(msg))
endpointDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
```

Add a counter for validation errors. The counter is a no-op when telemetry is disabled.

```go
var validationErrors = tc.Int64Counter("inertiaframe.validation_error",
    metric.WithDescription("Number of validation errors"),
)

// On validation failure
validationErrors.Add(ctx, 1, metric.WithAttributes(
    attribute.String("inertiaframe.error_bag", errorBag),
))
```

Add a counter for empty responses. The counter is a no-op when telemetry is disabled.

```go
var emptyResponses = tc.Int64Counter("inertiaframe.empty_response",
    metric.WithDescription("Number of empty responses from endpoints"),
)

// When resp == nil
emptyResponses.Add(ctx, 1)
```

| Metric                           | Type      | Dimensions               | Why                                                                                  |
| -------------------------------- | --------- | ------------------------ | ------------------------------------------------------------------------------------ |
| `inertiaframe.endpoint.duration` | Histogram | none                     | If user code is slow, this metric spikes. Separates user code from library overhead. |
| `inertiaframe.validation_error`  | Counter   | `inertiaframe.error_bag` | Identifies which forms are failing. `error_bag` is low cardinality.                  |
| `inertiaframe.empty_response`    | Counter   | none                     | Endpoints returning nil.                                                             |

---

## 6. `inertiaframe/session.go` — `sessionFromRequest()` and `Save()`

### Traces

No spans. Not critical enough.

### Metrics

Add a histogram for session deserialization time. The histogram is a no-op when telemetry is disabled.

```go
var sessionReadDuration = tc.Float64Histogram("inertiaframe.session.read.duration",
    metric.WithUnit("ms"),
    metric.WithDescription("Duration of session deserialization"),
)

func sessionFromRequest(r *http.Request) (*session, error) {
    start := time.Now()
    defer func() {
        tc := telemetryFromRequest(r) // retrieve TelemetryConfig from request context
        tc.Float64Histogram("inertiaframe.session.read.duration").Record(
            r.Context(), float64(time.Since(start).Milliseconds()))
    }()

    // ... decode session ...
}
```

| Metric                               | Type      | Dimensions | Why                                                 |
| ------------------------------------ | --------- | ---------- | --------------------------------------------------- |
| `inertiaframe.session.read.duration` | Histogram | none       | If this spikes, your session cookies are too large. |

---

## Complete Metrics & Traces Summary

### Traces

| Span                               | Where                                | Attributes                                                                                 | Events                          |
| ---------------------------------- | ------------------------------------ | ------------------------------------------------------------------------------------------ | ------------------------------- |
| `inertia.render`                   | `renderer.go`                        | `inertia.component`, `inertia.request.type`, `inertia.render.type`, `inertia.ssr.enabled`  | `inertia.render.props.resolved` |
| `inertia.props.resolve`            | `internal/inertiaprotocol/render.go` | `inertia.props.total`, `inertia.props.concurrent`                                          | —                               |
| `inertia.props.resolve.concurrent` | `internal/inertiaprotocol/render.go` | —                                                                                          | —                               |
| `inertia.render.ssr`               | `internal/inertiassr/ssr.go`         | `inertia.ssr.fallback`                                                                     | `inertia.render.ssr.fallback`   |
| `inertiaframe.endpoint`            | `inertiaframe/inertiaframe.go`       | `inertiaframe.endpoint.method`, `inertiaframe.endpoint.path`, `inertiaframe.response.type` | —                               |
| `inertiaframe.execute`             | `inertiaframe/inertiaframe.go`       | —                                                                                          | —                               |
| _(otelhttp span)_                  | `middleware.go`                      | `inertia.request.type`, `inertia.version.mismatch`                                         | `inertia.request.invalid`       |

### Metrics

| Metric                                         | Type      | Dimensions               | Where                                |
| ---------------------------------------------- | --------- | ------------------------ | ------------------------------------ |
| `inertia.render.duration`                      | Histogram | `inertia.request.type`   | `renderer.go`                        |
| `inertia.props.concurrent.resolution.duration` | Histogram | none                     | `internal/inertiaprotocol/render.go` |
| `inertia.props.rescued`                        | Counter   | none                     | `internal/inertiaprotocol/render.go` |
| `inertia.ssr.fallback`                         | Counter   | none                     | `internal/inertiassr/ssr.go`         |
| `inertiaframe.endpoint.duration`               | Histogram | none                     | `inertiaframe/inertiaframe.go`       |
| `inertiaframe.validation_error`                | Counter   | `inertiaframe.error_bag` | `inertiaframe/inertiaframe.go`       |
| `inertiaframe.empty_response`                  | Counter   | none                     | `inertiaframe/inertiaframe.go`       |
| `inertiaframe.session.read.duration`           | Histogram | none                     | `inertiaframe/session.go`            |

### Configuration

| Layer        | Config Field                 | Type                      |
| ------------ | ---------------------------- | ------------------------- |
| Renderer     | `Config.Telemetry`           | `inertia.TelemetryConfig` |
| Middleware   | `MiddlewareConfig.Telemetry` | `inertia.TelemetryConfig` |
| inertiaframe | `MountConfig.Telemetry`      | `inertia.TelemetryConfig` |

### otelutil API

| Method                               | Returns                   | Behavior when disabled                                                        |
| ------------------------------------ | ------------------------- | ----------------------------------------------------------------------------- |
| `TelemetryConfig.Defaults()`         | —                         | Sets no-op providers for any nil fields. Called automatically by the library. |
| `TelemetryConfig.Span()`             | `trace.Span`              | No-op span from `trace.NewNoopTracerProvider()`, `End()` does nothing         |
| `TelemetryConfig.Int64Counter()`     | `metric.Int64Counter`     | No-op counter, `Add()` does nothing                                           |
| `TelemetryConfig.Float64Histogram()` | `metric.Float64Histogram` | No-op histogram, `Record()` does nothing                                      |

`Defaults()` is called automatically by `Config.defaults()`, `MiddlewareConfig.defaults()`, and `Mount()`. After calling `Defaults()`, `Span()` and the metric methods always use the configured provider directly — no conditional selection, no `var span trace.Span`, no `if` guards. The provider is either real (telemetry enabled) or no-op (telemetry disabled), and both are safe to use unconditionally.

This is the complete set. If a signal does not help you debug a production issue, do not add it.
