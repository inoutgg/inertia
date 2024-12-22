package inertia

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"go.inout.gg/inertia/internal/inertiaheader"
)

var _ SsrClient = (*ssr)(nil)

type SsrTemplateData struct {
	Head string `json:"head"`
	Body string `json:"body"`
}

// SsrClient is a client that makes requests to a server-side rendering service.
type SsrClient interface {
	// Render makes a request to the server-side rendering service with the given page data.
	Render(*Page) (*SsrTemplateData, error)
}

// ssr is an HTTP client that makes requests to a server-side rendering service.
type ssr struct {
	client *http.Client
	url    string
}

// NewHTTPSsrClient creates a new SsrClient that makes requests to the given HTTP client.
// If client is nil, http.DefaultClient is used.
func NewHTTPSsrClient(url string, client *http.Client) SsrClient {
	if client == nil {
		client = http.DefaultClient
	}

	return &ssr{client, url}
}

func (s *ssr) Render(p *Page) (*SsrTemplateData, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("inertia: failed to marshal page: %w", err)
	}

	r, err := http.NewRequest(http.MethodGet, s.url, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("inertia: failed to create HTTP request: %w", err)
	}
	r.Header.Set(inertiaheader.HeaderContentType, contentTypeJSON)

	resp, err := s.client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("inertia: failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inertia: unexpected HTTP status code: %d", resp.StatusCode)
	}

	var data SsrTemplateData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("inertia: failed to decode JSON response: %w", err)
	}

	return &data, nil
}
