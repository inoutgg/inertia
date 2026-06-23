package inertiaframe

import (
	"fmt"
	"mime"
	"net/http"

	"github.com/go-json-experiment/json"
	"github.com/go-playground/form/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
)

const (
	mediaTypeJSON      = "application/json"
	mediaTypeForm      = "application/x-www-form-urlencoded"
	mediaTypeMultipart = "multipart/form-data"
)

// DefaultFormDecoder is the form decoder used when Config.FormDecoder is nil.
// It is shared across all endpoints, so custom type decoders registered on it
// apply application-wide.
var DefaultFormDecoder = form.NewDecoder() //nolint:gochecknoglobals

// Request represents a parsed and validated client request passed to an
// Endpoint's Execute method.
type Request[M any] struct {
	// Message is the decoded request payload, parsed from JSON or form data
	// according to the request's Content-Type. If M implements
	// RawRequestExtractor, custom extraction logic is used instead.
	Message M
}

// newRequest creates a new request.
func newRequest[M any](m M) *Request[M] {
	return &Request[M]{Message: m}
}

// RawRequestExtractor allows a message type to bypass the default JSON/form
// parsing and provide its own extraction logic.
//
// When M implements this interface, Extract is called with the raw
// *http.Request and the default decoder is not used. This is useful when the
// request body needs custom handling, such as streaming parsing or
// non-standard formats.
type RawRequestExtractor interface {
	// Extract populates the message's fields from the raw HTTP request.
	// Return an error to abort the request and trigger error handling.
	Extract(*http.Request) error
}

func (h *handler[M]) decodeBody(r *http.Request) (M, error) {
	var msg M

	mediaType, _, err := mime.ParseMediaType(r.Header.Get(inertiaheader.HeaderContentType))
	if err != nil {
		return msg, fmt.Errorf("inertiaframe: failed to parse Content-Type header: %w", err)
	}
	defer r.Body.Close()

	span := trace.SpanFromContext(r.Context())
	span.SetAttributes(attribute.String("inertiaframe.request.mime", mediaType))

	// Inertia accepts only JSON or multipart/form-data.
	switch mediaType {
	case mediaTypeJSON:
		{
			d("received JSON request")

			if err := json.UnmarshalRead(r.Body, &msg, h.jsonOptions...); err != nil {
				return msg, fmt.Errorf("inertiaframe: failed to decode request: %w", err)
			}
		}
	case mediaTypeForm, mediaTypeMultipart:
		{
			d("received form request")

			if err := r.ParseForm(); err != nil {
				return msg, fmt.Errorf("inertiaframe: failed to parse form data: %w", err)
			}

			if err := h.formDecoder.Decode(&msg, r.Form); err != nil {
				return msg, fmt.Errorf("inertiaframe: failed to decode form data: %w", err)
			}
		}
	default:
		d("unknown Content-Type %q, leaving message empty", mediaType)
	}

	return msg, nil
}

// extractRequest parses the request body into the message type M. GET requests
// skip body decoding.
func (h *handler[M]) extractRequest(r *http.Request) (M, error) {
	var msg M

	if extract, ok := any(msg).(RawRequestExtractor); ok {
		if err := extract.Extract(r); err != nil {
			return msg, fmt.Errorf("inertiaframe: failed to extract request data: %w", err)
		}

		return msg, nil
	}

	if r.Method == http.MethodGet {
		d("GET %s, skipping body decode", r.URL.Path)
		return msg, nil
	}

	return h.decodeBody(r)
}
