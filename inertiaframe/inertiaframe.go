// inertiaframe implements an opinionated framework around Go's HTTP and Inertia
// library, abstracting out protocol level details and providing a simple
// message based API.
package inertiaframe

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/http/httperror"
	"go.inout.gg/foundations/http/httprequest"

	"go.inout.gg/inertia"
	"go.inout.gg/inertia/inertiaprops"
)

var d = debug.Debuglog("inertiaframe") //nolint:gochecknoglobals

var (
	DefaultValidator = validator.New(validator.WithRequiredStructEnabled()) //nolint:gochecknoglobals

	//nolint:gochecknoglobals
	DefaultErrorHandler httperror.ErrorHandler = httperror.ErrorHandlerFunc(
		func(w http.ResponseWriter, r *http.Request, err error) {
			httperror.DefaultErrorHandler(w, r, err)
		},
	)
)

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

	Response struct {
		msg            Message
		component      string
		clearHistory   bool
		encryptHistory bool
	}
)

type Option func(*Response)

// WithClearHistory sets the history clear.
func WithClearHistory() Option {
	return func(opt *Response) { opt.clearHistory = true }
}

// WithEncryptHistory instructs the client to encrypt the history state.
func WithEncryptHistory() Option {
	return func(resp *Response) { resp.encryptHistory = true }
}

func NewResponse(msg Message, opts ...Option) *Response {
	resp := &Response{
		component:      msg.Component(),
		msg:            msg,
		clearHistory:   false,
		encryptHistory: false,
	}
	for _, opt := range opts {
		opt(resp)
	}

	return resp
}

type redirectMessage struct {
	URL string
}

func (m *redirectMessage) Component() string { return "" }

func (m *redirectMessage) Write(w http.ResponseWriter, r *http.Request) error {
	inertia.Location(w, r, m.URL)
	return nil
}

// NewRedirectResponse creates a new response that redirects the client to the
// specified URL.
func NewRedirectResponse(url string) *Response {
	return &Response{
		msg:            &redirectMessage{URL: url},
		component:      "",
		clearHistory:   false,
		encryptHistory: false,
	}
}

type Message interface {
	// Component returns the component name to be rendered.
	//
	// Executor panics if Component returns an empty string,
	// unless the message implements RawResponseWriter.
	//
	// If the message is implementing RawResponseWriter, the default
	// behavior is prevented and the writer is used instead to
	// write the response data.
	Component() string
}

// RawRequestExtractor allows to extract data from the raw http.Request.
// If a request message implements RawRequestExtractor, the default
// behavior is prevented and the extractor is used instead to
// extract the request data.
type RawRequestExtractor interface {
	// Extract extracts data from the raw http.Request.
	// It can be used to extract
	Extract(*http.Request) error
}

// RawResponseWriter allows to write data to the http.ResponseWriter.
// If a response message implements RawResponseWriter, the default
// behavior is prevented and the writer is used instead to
// write the response data.
type RawResponseWriter interface {
	Write(http.ResponseWriter, *http.Request) error
}

// Meta is the metadata of an endpoint.
type Meta struct {
	ErrorHandler httperror.ErrorHandler
	Method       string
	Path         string
	StatusCode   int
}

type Endpoint[R any] interface {
	// Execute executes the endpoint.
	Execute(context.Context, *Request[R]) (*Response, error)

	// Meta is the metadata of the endpoint. It is used to configure
	// the endpoint's behavior when mounted on a given http.ServeMux.
	Meta() *Meta
}

type MountOpts struct {
	// Validator is the validator used to validate the request data.
	// If validator is nil, the default validator will be used.
	Validator *validator.Validate

	// ErrorHandler is the error handler used to handle errors.
	// If errorHandler is nil, the default error handler will be used.
	ErrorHandler httperror.ErrorHandler
}

// Mount mounts the executor on the given http.ServeMux.
//
// Executor must specify the HTTP method and path.
func Mount[Req any](mux *http.ServeMux, e Endpoint[Req], opts *MountOpts) {
	if opts == nil {
		opts = &MountOpts{
			Validator:    DefaultValidator,
			ErrorHandler: DefaultErrorHandler,
		}
	}

	opts.ErrorHandler = cmp.Or(opts.ErrorHandler, DefaultErrorHandler)
	opts.Validator = cmp.Or(opts.Validator, DefaultValidator)

	m := e.Meta()

	debug.Assert(m.Method != "", "Executor must specify the HTTP method")
	debug.Assert(m.Path != "", "Executor must specify the HTTP path")
	debug.Assert(opts.ErrorHandler != nil, "Executor must specify the error handler")
	debug.Assert(opts.Validator != nil, "Executor must specify the validator")

	pattern := fmt.Sprintf("%s %s", m.Method, m.Path)

	d("Mounting executor on pattern: %s", pattern)

	mux.Handle(pattern, NewHandler(e, opts.ErrorHandler))
}

func NewHandler[Req any](e Endpoint[Req], errorHandler httperror.ErrorHandler) http.Handler {
	handleError := httperror.WithErrorHandler(errorHandler)

	return handleError(httperror.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		contentType := r.Header.Get("Content-Type")

		var data *Req

		var err error

		if extract, ok := any(data).(RawRequestExtractor); ok {
			if err := extract.Extract(r); err != nil {
				return fmt.Errorf("inertiaframe: failed to extract request data: %w", err)
			}
		} else {
			// Inertia accepts only JSON, or multipart/form-data.
			switch {
			case strings.HasPrefix(contentType, contentTypeJSON):
				data, err = httprequest.DecodeJSON[Req](r)
			case strings.HasPrefix(contentType, contentTypeForm),
				strings.HasPrefix(contentType, contentTypeMultipart):
				data, err = httprequest.DecodeForm[Req](r)
			}

			if err != nil {
				return fmt.Errorf("inertiaframe: failed to decode request: %w", err)
			}
		}

		resp, err := e.Execute(ctx, &Request[Req]{Raw: r, Data: data})
		if err != nil {
			return fmt.Errorf("inertiaframe: failed to execute: %w", err)
		}

		opts := make([]inertia.Option, 0, 2)

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

		if err := inertia.Render(w, r, resp.component, opts...); err != nil {
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

	props, err := inertiaprops.ParseStruct(msg)
	if err != nil {
		return nil, fmt.Errorf("inertiaframe: failed to parse props: %w", err)
	}

	return props, nil
}
