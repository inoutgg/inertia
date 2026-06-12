// Package otelutil provides a thin, nil-safe wrapper around OpenTelemetry
// providers so callers do not need to repeat nil / enabled checks everywhere.
package otelutil

import (
	"context"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

const instrumentationName = "go.segfaultmedaddy.com/inertia"

// TelemetryConfig holds OpenTelemetry providers for tracing and metrics.
// If both fields are nil, telemetry is a no-op after calling Defaults().
type TelemetryConfig struct {
	// TracerProvider provides tracers for creating spans.
	// If nil, a no-op tracer is used after calling Defaults().
	TracerProvider trace.TracerProvider

	// MeterProvider provides meters for recording metrics.
	// If nil, a no-op meter is used after calling Defaults().
	MeterProvider metric.MeterProvider
}

// Defaults sets no-op providers for any nil fields.
// Callers should invoke this before using the config.
func (t *TelemetryConfig) Defaults() {
	if t.TracerProvider == nil {
		t.TracerProvider = tracenoop.NewTracerProvider()
	}

	if t.MeterProvider == nil {
		t.MeterProvider = noop.NewMeterProvider()
	}
}

// Span creates a span. The returned trace.Span is always valid; it is a real
// span when telemetry is enabled and a no-op span otherwise.
//
//nolint:spancheck // The caller is responsible for calling span.End().
func (t *TelemetryConfig) Span(
	ctx context.Context,
	name string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	tracer := t.TracerProvider.Tracer(instrumentationName)
	ctx, span := tracer.Start(ctx, name, opts...)

	return ctx, span
}

// Int64Counter returns a counter. The returned counter is always valid; it is a
// real counter when telemetry is enabled and a no-op counter otherwise.
func (t *TelemetryConfig) Int64Counter(name string, opts ...metric.Int64CounterOption) metric.Int64Counter {
	c, _ := t.MeterProvider.Meter(instrumentationName).Int64Counter(name, opts...)

	return c
}

// Float64Histogram returns a histogram. The returned histogram is always valid;
// it is a real histogram when telemetry is enabled and a no-op histogram otherwise.
func (t *TelemetryConfig) Float64Histogram(name string, opts ...metric.Float64HistogramOption) metric.Float64Histogram {
	h, _ := t.MeterProvider.Meter(instrumentationName).Float64Histogram(name, opts...)

	return h
}
