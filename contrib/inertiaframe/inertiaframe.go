// inertiaframe implements an opinionated framework around Go's HTTP and Inertia
// library, abstracting out protocol level details and providing a simple
// message based API.
package inertiaframe

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/http/httperror"
	"go.inout.gg/foundations/http/httprequest"

	"go.inout.gg/inertia"
)

var d = debug.Debuglog("inertiaframe") //nolint:gochecknoglobals

const (
	contentTypeJSON      = "application/json"
	contentTypeForm      = "application/x-www-form-urlencoded"
	contentTypeMultipart = "multipart/form-data"
)

type (
	Request[M any] struct {
		Raw  *http.Request
		Data *M
	}

	Response[M any] struct {
		msg            M
		clearHistory   bool
		encryptHistory bool
		headers        http.Header
	}
)

type Option[M any] func(*Response[M])

// WithClearHistory sets the history clear.
func WithClearHistory() Option[any] {
	return func(opt *Response[any]) { opt.clearHistory = true }
}

// WithEncryptHistory instructs the client to encrypt the history state.
func WithEncryptHistory() Option[any] {
	return func(resp *Response[any]) { resp.encryptHistory = true }
}

// WithHeaders sets the HTTP headers of the response,
// it copies the headers from the provided http.Header.
func WithHeaders(headers http.Header) Option[any] {
	return func(resp *Response[any]) {
		if resp.headers == nil {
			resp.headers = make(http.Header, len(headers))
		}

		for k, v := range headers {
			resp.headers[k] = v
		}
	}
}

func NewResponse[M any](msg M, opts ...Option[M]) *Response[M] {
	resp := &Response[M]{
		msg:            msg,
		clearHistory:   false,
		encryptHistory: false,
		headers:        nil,
	}
	for _, opt := range opts {
		opt(resp)
	}

	return resp
}

// Meta is the metadata of an endpoint.
type Meta struct {
	ErrorHandler httperror.ErrorHandler
	Method       string
	Path         string
	StatusCode   int
}

type Executor[R, W any] interface {
	// Component is the name of the inertia component to render.
	Component() string

	Execute(context.Context, *Request[R]) (*Response[W], error)

	// Meta is the metadata of the endpoint. It is used to configure
	// the endpoint's behavior when mounted on a given http.ServeMux.
	Meta() *Meta
}

// Mount mounts the executor on the given http.ServeMux.
//
// Executor must specify the HTTP method and path.
func Mount(mux *http.ServeMux, e Executor[any, any]) {
	m := e.Meta()

	debug.Assert(m.Method != "", "Executor must specify the HTTP method")
	debug.Assert(m.Path != "", "Executor must specify the HTTP path")

	pattern := fmt.Sprintf("%s %s", m.Method, m.Path)

	d("Mounting executor on pattern: %s", pattern)

	mux.Handle(pattern, NewHandler(e, httperror.ErrorHandlerFunc(DefaultErrorHandler)))
}

func DefaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	httperror.DefaultErrorHandler(w, r, err)
}

func NewHandler[R, W any](e Executor[R, W], errorHandler httperror.ErrorHandler) http.Handler {
	handleError := httperror.WithErrorHandler(errorHandler)

	return handleError(httperror.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		contentType := r.Header.Get("Content-Type")

		var data *R

		var err error

		switch {
		case strings.HasPrefix(contentType, contentTypeJSON):
			data, err = httprequest.DecodeJSON[R](r)
		case strings.HasPrefix(contentType, contentTypeForm),
			strings.HasPrefix(contentType, contentTypeMultipart):
			data, err = httprequest.DecodeForm[R](r)
		}

		if err != nil {
			return fmt.Errorf("inertiaframe: failed to decode request: %w", err)
		}

		resp, err := e.Execute(ctx, &Request[R]{
			Data: data,
			Raw:  r,
		})
		if err != nil {
			return fmt.Errorf("inertiaframe: failed to execute: %w", err)
		}

		opts := make([]inertia.Option, 0, 1)

		if resp != nil {
			if resp.clearHistory {
				opts = append(opts, inertia.WithClearHistory())
			}

			if resp.encryptHistory {
				opts = append(opts, inertia.WithEncryptHistory())
			}

			props, err := extractProps(resp.msg)
			if err != nil {
				return fmt.Errorf("inertiaframe: failed to extract props: %w", err)
			}

			if props.Len() > 0 {
				opts = append(opts, inertia.WithProps(props))
			}
		}

		if err := inertia.Render(w, r, e.Component(), opts...); err != nil {
			return fmt.Errorf("inertiaframe: failed to render: %w", err)
		}

		return nil
	}))
}

// extractProps extracts props from the given message.
//
// If the message implements the inertia.Proper interface,
// it returns the props from the message.
// Otherwise, it attempts to parse the message as a struct and
// returns the props from the struct.
func extractProps(msg any) (inertia.Props, error) {
	proper, ok := msg.(inertia.Proper)
	if ok {
		return proper.Props(), nil
	}

	props, err := inertia.ParseStruct(msg)
	if err != nil {
		return nil, fmt.Errorf("inertiaframe: failed to parse props: %w", err)
	}

	return props, nil
}
