package inertiahttp

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"go.inout.gg/foundations/debug"

	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
	"go.segfaultmedaddy.com/inertia/internal/sliceutil"
)

// ErrInvalidInertiaRequest is the sentinel error returned (or wrapped) when an
// incoming Inertia request has malformed or invalid protocol headers.
// Callers can match it with errors.Is to distinguish protocol errors from
// other parse failures.
var ErrInvalidInertiaRequest = errors.New("inertia: invalid request")

// Request holds the parsed Inertia.js request data extracted from HTTP headers.
type Request struct {
	URL               string
	Version           string
	PartialComponent  string
	ErrorBag          string
	ScrollMergeIntent string
	PartialData       []string
	PartialExcept     []string
	ResetProps        []string
	ExceptOnceProps   []string
	PrecognitionProps []string
	Precognition      bool
	IsInertia         bool
}

// ParseRequest parses Inertia.js protocol headers from r and returns
// a Request describing the client's intent.
//
// For non-Inertia requests only URL and IsInertia are populated.
func ParseRequest(r *http.Request) (Request, error) {
	debug.Assert(r != nil, "Request must not be nil")

	//nolint:exhaustruct
	req := Request{
		URL:       r.RequestURI,
		IsInertia: r.Header.Get(inertiaheader.HeaderXInertia) == inertiaheader.HeaderValueTrue,
	}

	if !req.IsInertia {
		return req, nil
	}

	req.Version = r.Header.Get(inertiaheader.HeaderXInertiaVersion)

	precognition := r.Header.Get(inertiaheader.HeaderPrecognition) == inertiaheader.HeaderValueTrue
	if precognition {
		req.Precognition = true
		req.PrecognitionProps = parseHeaderValueList(
			r.Header.Get(inertiaheader.HeaderPrecognitionValidateOnly),
		)

		// If it is a precognition request then we can get the only necessary information
		// needed for the inertia.Middleware to handle request + precognition request
		// itself, since the main execution won't happen anyway.
		return req, nil
	}

	req.PartialComponent = r.Header.Get(inertiaheader.HeaderXInertiaPartialComponent)
	req.PartialData = parseHeaderValueList(r.Header.Get(inertiaheader.HeaderXInertiaPartialData))
	req.PartialExcept = parseHeaderValueList(r.Header.Get(
		inertiaheader.HeaderXInertiaPartialExcept))

	if (len(req.PartialData) > 0 || len(req.PartialExcept) > 0) && req.PartialComponent == "" {
		d(
			"rejecting partial reload without %s header",
			inertiaheader.HeaderXInertiaPartialComponent,
		)

		return req, fmt.Errorf(
			"%w: %s is required for partial reloads",
			ErrInvalidInertiaRequest,
			inertiaheader.HeaderXInertiaPartialComponent,
		)
	}

	scrollMergeIntent, err := parseScrollMergeIntent(
		r.Header.Get(inertiaheader.HeaderXInertiaScrollMerge),
	)
	if err != nil {
		return req, err
	}

	req.ErrorBag = r.Header.Get(inertiaheader.HeaderXInertiaErrorBag)
	req.ScrollMergeIntent = scrollMergeIntent
	req.ResetProps = parseHeaderValueList(r.Header.Get(inertiaheader.HeaderXInertiaReset))
	req.ExceptOnceProps = parseHeaderValueList(
		r.Header.Get(inertiaheader.HeaderXInertiaExceptOnceProps),
	)

	d("parsed request: method=%s url=%s inertia=%v partial=%q once-except=%v",
		r.Method, r.RequestURI, req.IsInertia, req.PartialComponent, req.ExceptOnceProps)

	return req, nil
}

func parseHeaderValueList(value string) []string {
	if value == "" {
		return nil
	}

	return sliceutil.Filter(
		sliceutil.Map(
			strings.Split(value, ","),
			strings.TrimSpace,
		),
		func(s string) bool { return s != "" },
	)
}

// parseScrollMergeIntent validates the X-Inertia-Infinite-Scroll-Merge-Intent
// header value.
//
// An empty header is treated as "no intent". Any non-empty value that is not
// exactly "append" or "prepend" is rejected.
func parseScrollMergeIntent(intent string) (string, error) {
	if intent == "" {
		return "", nil
	}

	intent = strings.TrimSpace(intent)
	switch intent {
	case inertiaprop.ScrollMergeIntentAppend, inertiaprop.ScrollMergeIntentPrepend:
		return intent, nil
	default:
		return "", fmt.Errorf("%w: invalid scroll merge intent: %q", ErrInvalidInertiaRequest, intent)
	}
}
