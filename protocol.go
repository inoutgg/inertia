package inertia

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
)

// ErrInvalidInertiaRequest is the sentinel error returned (or wrapped) when an
// incoming Inertia request has malformed or invalid protocol headers.
// Callers can match it with errors.Is to distinguish protocol errors from
// other parse failures.
var ErrInvalidInertiaRequest = errors.New("inertia: invalid request")

type request struct {
	URL               string
	Version           string
	PartialComponent  string
	ErrorBag          string
	ScrollMergeIntent string
	PartialData       []string
	PartialExcept     []string
	ResetProps        []string
	ExceptOnceProps   []string
	IsInertia         bool
}

type response struct {
	Headers map[string]string
	Body    []byte
}

func parseRequest(r *http.Request) (request, error) {
	//nolint:exhaustruct
	req := request{
		URL:       r.RequestURI,
		IsInertia: r.Header.Get(inertiaheader.HeaderXInertia) == inertiaheader.HeaderValueTrue,
	}

	if !req.IsInertia {
		return req, nil
	}

	partialData, err := parseHeaderValueList(r.Header.Get(
		inertiaheader.HeaderXInertiaPartialData), inertiaheader.HeaderXInertiaPartialData)
	if err != nil {
		return req, err
	}

	partialExcept, err := parseHeaderValueList(r.Header.Get(
		inertiaheader.HeaderXInertiaPartialExcept), inertiaheader.HeaderXInertiaPartialExcept)
	if err != nil {
		return req, err
	}

	resetProps, err := parseHeaderValueList(
		r.Header.Get(inertiaheader.HeaderXInertiaReset),
		inertiaheader.HeaderXInertiaReset,
	)
	if err != nil {
		return req, err
	}

	exceptOnceProps, err := parseHeaderValueList(r.Header.Get(
		inertiaheader.HeaderXInertiaExceptOnceProps), inertiaheader.HeaderXInertiaExceptOnceProps)
	if err != nil {
		return req, err
	}

	scrollMergeIntent, err := parseScrollMergeIntent(
		r.Header.Get(inertiaheader.HeaderXInertiaScrollMerge))
	if err != nil {
		return req, err
	}

	req.Version = r.Header.Get(inertiaheader.HeaderXInertiaVersion)
	req.PartialComponent = r.Header.Get(inertiaheader.HeaderXInertiaPartialComponent)
	req.ErrorBag = r.Header.Get(inertiaheader.HeaderXInertiaErrorBag)
	req.ScrollMergeIntent = scrollMergeIntent
	req.PartialData = partialData
	req.PartialExcept = partialExcept
	req.ResetProps = resetProps
	req.ExceptOnceProps = exceptOnceProps

	if (len(req.PartialData) > 0 || len(req.PartialExcept) > 0) && req.PartialComponent == "" {
		return req, fmt.Errorf("%w: %s is required for partial reloads",
			ErrInvalidInertiaRequest, inertiaheader.HeaderXInertiaPartialComponent)
	}

	return req, nil
}

func parseHeaderValueList(value string, header string) ([]string, error) {
	if value == "" {
		return nil, nil
	}

	fields := strings.Split(value, ",")
	for i, field := range fields {
		fields[i] = strings.TrimSpace(field)
		if fields[i] == "" {
			return nil, fmt.Errorf("%w: %s contains an empty value", ErrInvalidInertiaRequest, header)
		}
	}

	return fields, nil
}

// parseScrollMergeIntent validates the X-Inertia-Infinite-Scroll-Merge-Intent
// header value.
//
// An empty header is treated as "no intent" and is not an error. Any non-empty
// value that is not exactly "append" or "prepend" is rejected. We intentionally
// do not fall back to a default for unknown values; the Inertia.js contract
// is strict, and silently coercing a typo to one of the two known intents
// would mask client bugs.
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
