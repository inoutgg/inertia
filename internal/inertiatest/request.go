package inertiatest

import (
	"cmp"
	"net/http"
	"net/http/httptest"
	"strings"

	"go.segfaultmedaddy.com/inertia/internal/inertiaheader"
)

type RequestConfig struct {
	Version                  string
	PartialComponent         string
	ErrorBag                 string
	ScrollMergeIntent        string
	Whitelist                []string
	Blacklist                []string
	ResetProps               []string
	OnceProps                []string
	PrecognitionValidateOnly []string
	Precognition             bool
	Inertia                  bool
}

// NewRequest creates a new request with an empty body.
func NewRequest(
	method string,
	target string,
	config *RequestConfig,
) (*http.Request, *httptest.ResponseRecorder) {
	r := httptest.NewRequest(method, target, nil)

	//nolint:exhaustruct
	config = cmp.Or(config, &RequestConfig{})

	if config.Inertia {
		r.Header.Set(inertiaheader.HeaderXInertia, inertiaheader.HeaderValueTrue)
	}

	if config.Version != "" {
		r.Header.Set(inertiaheader.HeaderXInertiaVersion, config.Version)
	}

	if len(config.Whitelist) > 0 {
		r.Header.Set(inertiaheader.HeaderXInertiaPartialData, strings.Join(config.Whitelist, ","))
	}

	if len(config.Blacklist) > 0 {
		r.Header.Set(inertiaheader.HeaderXInertiaPartialExcept, strings.Join(config.Blacklist, ","))
	}

	if len(config.ResetProps) > 0 {
		r.Header.Set(inertiaheader.HeaderXInertiaReset, strings.Join(config.ResetProps, ","))
	}

	if len(config.OnceProps) > 0 {
		r.Header.Set(inertiaheader.HeaderXInertiaExceptOnceProps, strings.Join(config.OnceProps, ","))
	}

	if config.PartialComponent != "" {
		r.Header.Set(inertiaheader.HeaderXInertiaPartialComponent, config.PartialComponent)
	}

	if config.ErrorBag != "" {
		r.Header.Set(inertiaheader.HeaderXInertiaErrorBag, config.ErrorBag)
	}

	if config.ScrollMergeIntent != "" {
		r.Header.Set(inertiaheader.HeaderXInertiaScrollMerge, config.ScrollMergeIntent)
	}

	if config.Precognition {
		r.Header.Set(inertiaheader.HeaderPrecognition, inertiaheader.HeaderValueTrue)
	}

	if len(config.PrecognitionValidateOnly) > 0 {
		r.Header.Set(inertiaheader.HeaderPrecognitionValidateOnly,
			strings.Join(config.PrecognitionValidateOnly, ","))
	}

	return r, httptest.NewRecorder()
}
