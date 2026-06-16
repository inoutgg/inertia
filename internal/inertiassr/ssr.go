package inertiassr

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/go-json-experiment/json"
	"go.inout.gg/foundations/debug"
	"go.opentelemetry.io/otel/metric"

	"go.segfaultmedaddy.com/inertia/inertiaotel"
	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprotocol"
)

var _ SSRClient = (*ssr)(nil)

var d = debug.Debuglog("inertiassr") //nolint:gochecknoglobals

type SSRTemplateData struct {
	Head string `json:"head"`
	Body string `json:"body"`
}

//go:generate mockgen -destination ssr_mock.go -package inertiassr . SSRClient
type SSRClient interface {
	// Render makes a request to the server-side rendering service with the given page data.
	Render(context.Context, *inertiaprotocol.Page) (*SSRTemplateData, error)
}

// ssr is an HTTP client that makes requests to a server-side rendering service.
type ssr struct {
	ssrFallbackCounter metric.Int64Counter
	client             *http.Client
	telemetry          *inertiaotel.Config
	url                string
}

type Option func(*ssr)

func WithHTTPClient(client *http.Client) Option {
	return func(s *ssr) { s.client = client }
}

func WithTelemetry(t *inertiaotel.Config) Option {
	return func(s *ssr) { s.telemetry = t }
}

func NewHTTPSsrClient(url string, opts ...Option) SSRClient {
	//nolint:exhaustruct
	client := &ssr{
		client:    http.DefaultClient,
		telemetry: inertiaotel.DefaultConfig,
		url:       url,
	}
	for _, opt := range opts {
		opt(client)
	}

	debug.Assert(url != "", "url must be provided")
	debug.Assert(client != nil, "client must be provided")
	debug.Assert(client.telemetry != nil, "telemetry config must be provided")

	var err error

	client.ssrFallbackCounter, err = client.telemetry.Meter().Int64Counter(
		"inertia.ssr.client_render_fallback",
		metric.WithDescription("Number of SSR fallbacks to client-side rendering"),
	)
	if err != nil {
		d("failed to create inertia.ssr.client_render_fallback counter: %v", err)
	}

	return client
}

func (s *ssr) Render(ctx context.Context, page *inertiaprotocol.Page) (*SSRTemplateData, error) {
	debug.Assert(page != nil, "page must be set")

	d("requesting render of %q from %s", page.Component, s.url)

	var err error

	defer func() {
		if err != nil {
			s.ssrFallbackCounter.Add(ctx, 1)
		}
	}()

	b, err := json.Marshal(page)
	if err != nil {
		return nil, fmt.Errorf("inertia: failed to marshal page: %w", err)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("inertia: failed to create HTTP request: %w", err)
	}

	r.Header.Set(inertiaheader.HeaderContentType, inertiaheader.ContentTypeJSON)

	// #nosec G704 - URL is set by application during initialization, not user-provided
	resp, err := s.client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("inertia: failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		d("upstream returned status %d for %q", resp.StatusCode, page.Component)

		return nil, fmt.Errorf("inertia: unexpected HTTP status code: %d", resp.StatusCode)
	}

	var data SSRTemplateData
	if err := json.UnmarshalRead(resp.Body, &data); err != nil {
		return nil, fmt.Errorf("inertia: failed to decode JSON response: %w", err)
	}

	d("rendered %q head=%d body=%d", page.Component, len(data.Head), len(data.Body))

	return &data, nil
}
