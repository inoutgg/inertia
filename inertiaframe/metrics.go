package inertiaframe

import (
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

// metrics holds the OpenTelemetry instruments used by a mounted endpoint.
type metrics struct {
	endpointDuration        metric.Float64Histogram
	validationErrorsCounter metric.Int64Counter
	emptyResponsesCounter   metric.Int64Counter
	errorResponsesCounter   metric.Int64Counter
}

func newMetrics(meter metric.Meter) (*metrics, error) {
	var m metrics

	var err error
	if m.endpointDuration, err = meter.Float64Histogram(
		"inertiaframe.endpoint.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Duration of inertiaframe endpoint execution"),
	); err != nil {
		return nil, fmt.Errorf("inertiaframe: failed to create inertiaframe.endpoint.duration metric: %w", err)
	}

	if m.validationErrorsCounter, err = meter.Int64Counter(
		"inertiaframe.validation_error",
		metric.WithDescription("Number of validation errors"),
	); err != nil {
		return nil, fmt.Errorf("inertiaframe: failed to create inertiaframe.validation_error metric: %w", err)
	}

	if m.emptyResponsesCounter, err = meter.Int64Counter(
		"inertiaframe.empty_response",
		metric.WithDescription("Number of empty responses from endpoints"),
	); err != nil {
		return nil, fmt.Errorf("inertiaframe: failed to create inertiaframe.empty_response metric: %w", err)
	}

	if m.errorResponsesCounter, err = meter.Int64Counter(
		"inertiaframe.error_response",
		metric.WithDescription("Number of error responses from endpoints"),
	); err != nil {
		return nil, fmt.Errorf("inertiaframe: failed to create inertiaframe.error_response metric: %w", err)
	}

	return &m, nil
}
