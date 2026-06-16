package inertia

import (
	"net/http"

	"go.segfaultmedaddy.com/inertia/inertiaotel"
	"go.segfaultmedaddy.com/inertia/internal/inertiassr"
)

type (
	// SSRClient communicates with a server-side rendering service to pre-render Inertia pages.
	SSRClient = inertiassr.SSRClient

	// SsrTemplateData contains the HTML head and body sections returned by SSR rendering.
	SsrTemplateData = inertiassr.SSRTemplateData
)

// SSROption configures an HTTP-based SSR client.
type SSROption = inertiassr.Option

// WithSSRClient sets the HTTP client used to send SSR requests.
func WithSSRClient(client *http.Client) SSROption {
	return inertiassr.WithHTTPClient(client)
}

// WithSSRTelemetry sets the OpenTelemetry config used to record SSR metrics.
func WithSSRTelemetry(tc *inertiaotel.Config) SSROption {
	return inertiassr.WithTelemetry(tc)
}

// NewHTTPSsrClient creates an HTTP-based SSR client that sends render requests to the specified URL.
// The telemetry config is required (use WithSSRTelemetry) to enable OpenTelemetry metrics.
func NewHTTPSsrClient(url string, opts ...SSROption) SSRClient {
	return inertiassr.NewHTTPSsrClient(url, opts...)
}
