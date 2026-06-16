// Package inertiaotel provides OpenTelemetry configuration helpers for the inertia package.
package inertiaotel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

const instrumentationName = "go.segfaultmedaddy.com/inertia"

// DefaultConfig is the default telemetry config.
var DefaultConfig = New() //nolint:gochecknoglobals

// Config holds OpenTelemetry providers for tracing and metrics.
type Config struct {
	// tracerProvider provides tracers for creating spans.
	//
	// By default, a global tracer provider is used.
	// If nil, no-op providers are used instead.
	tracerProvider trace.TracerProvider

	// meterProvider provides meters for recording metrics.
	//
	// By default, a global meter provider is used.
	// If nil, no-op providers are used instead.
	meterProvider metric.MeterProvider
}

// Option configures the telemetry config.
type Option func(*Config)

// WithTracer sets the TracerProvider for the inertiaotel integration.
func WithTracer(tracer trace.TracerProvider) Option {
	return func(c *Config) { c.tracerProvider = tracer }
}

// WithMeter sets the MeterProvider for the inertiaotel integration.
func WithMeter(meter metric.MeterProvider) Option {
	return func(c *Config) { c.meterProvider = meter }
}

// New creates a new telemetry Config.
//
// The config can be used to set the OTeL integration for the inertia framework.
//
// It can be used to configure both inertia and inertiaframe integrations.
func New(opts ...Option) *Config {
	cfg := Config{
		tracerProvider: otel.GetTracerProvider(),
		meterProvider:  otel.GetMeterProvider(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &cfg
}

// Tracer returns a tracer scoped to this instrumentation library.
func (c *Config) Tracer() trace.Tracer {
	if c.tracerProvider == nil {
		return tracenoop.NewTracerProvider().Tracer(instrumentationName)
	}

	return c.tracerProvider.Tracer(instrumentationName)
}

// Meter returns a meter scoped to this instrumentation library.
func (c *Config) Meter() metric.Meter {
	if c.meterProvider == nil {
		return metricnoop.NewMeterProvider().Meter(instrumentationName)
	}

	return c.meterProvider.Meter(instrumentationName)
}
