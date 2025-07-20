// inertiaframe implements an opinionated framework around Go's HTTP and Inertia
// library, abstracting out protocol level details and providing a simple
// message based API.
package inertiaframe

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/form/v4"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	ven "github.com/go-playground/validator/v10/translations/en"
	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/http/httperror"
	"go.inout.gg/foundations/http/httpmiddleware"
	"go.inout.gg/foundations/must"

	"go.inout.gg/inertia"
	"go.inout.gg/inertia/contrib/inertiavalidationerrors"
	"go.inout.gg/inertia/inertiaprops"
	"go.inout.gg/inertia/internal/inertiaheader"
	"go.inout.gg/inertia/internal/inertiaredirect"
)

var d = debug.Debuglog("inertiaframe") //nolint:gochecknoglobals

var (
	DefaultValidator   = validator.New(validator.WithRequiredStructEnabled()) //nolint:gochecknoglobals
	DefaultFormDecoder = form.NewDecoder()                                    //nolint:gochecknoglobals
)

var (
	_ RawResponseWriter = (*redirectMessage)(nil)
	_ RawResponseWriter = (*redirectBackMessage)(nil)
	_ RawResponseWriter = (*externalRedirectMessage)(nil)
)

func RedirectBack(w http.ResponseWriter, r *http.Request) {
	referer := r.Header.Get(inertiaheader.HeaderReferer)
	if referer == "" {
		sess, err := sessionFromRequest(r)
		if err != nil {
			d("failed to get session from request, using default '/'")

			referer = "/"
		} else {
			referer = sess.Referer()
		}
	}

	d("redirecting back to %s", referer)

	inertiaredirect.Redirect(w, r, referer)
}

// DefaultValidationErrorHandler is a default error handler for validation errors.
//
// It saves flash messages and redirects back to the previous page.
func DefaultValidationErrorHandler(w http.ResponseWriter, r *http.Request, errorer inertia.ValidationErrorer) {
	errorBag := inertia.ErrorBagFromRequest(r)
	sess := must.Must(sessionFromRequest(r))

	sess.ErrorBag_ = errorBag
	sess.ValidationErrors_ = errorer.ValidationErrors()

	must.Must1(sess.Save(w))

	RedirectBack(w, r)
}

//nolint:gochecknoglobals
var DefaultErrorHandler httperror.ErrorHandler = httperror.ErrorHandlerFunc(
	func(w http.ResponseWriter, r *http.Request, err error) {
		var errorer inertia.ValidationErrorer
		if errors.As(err, &errorer) {
			DefaultValidationErrorHandler(w, r, errorer)
			return
		}

		errorer, ok := inertiavalidationerrors.FromValidationErrors(err, defaultTranslator)
		if ok {
			DefaultValidationErrorHandler(w, r, errorer)
			return
		}

		httperror.DefaultErrorHandler(w, r, err)
	},
)

const (
	mediaTypeJSON      = "application/json"
	mediaTypeForm      = "application/x-www-form-urlencoded"
	mediaTypeMultipart = "multipart/form-data"
)

var (
	defaultLocale            = en.New()              //nolint:gochecknoglobals
	defaultTranslationBundle = ut.New(defaultLocale) //nolint:gochecknoglobals

	//nolint:gochecknoglobals
	defaultTranslator, _ = defaultTranslationBundle.GetTranslator(
		defaultLocale.Locale(),
	)
)

//nolint:gochecknoinits
func init() {
	must.Must1(ven.RegisterDefaultTranslations(DefaultValidator, defaultTranslator))

	DefaultValidator.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return ""
		}

		return name
	})
}

// DefaultTranslator returns the default translator that always uses
// the default locale - English (en).
func DefaultTranslator(_ context.Context) ut.Translator {
	t, _ := defaultTranslationBundle.GetTranslator(defaultLocale.Locale())
	return t
}

type (
	Request[M any] struct {
		Message *M
	}

	Response struct {
		msg            Message
		clearHistory   bool
		encryptHistory bool
		concurrency    int
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

// WithConcurrency sets the concurrency level for response props resolution.
func WithConcurrency(concurrency int) Option {
	return func(resp *Response) { resp.concurrency = concurrency }
}

func NewResponse(msg Message, opts ...Option) *Response {
	resp := &Response{
		msg:            msg,
		clearHistory:   false,
		encryptHistory: false,
		concurrency:    inertia.DefaultConcurrency,
	}
	for _, opt := range opts {
		opt(resp)
	}

	return resp
}

type externalRedirectMessage struct{ url string }

func (m *externalRedirectMessage) Component() string { return "" }

func (m *externalRedirectMessage) Write(w http.ResponseWriter, r *http.Request) error {
	inertia.Location(w, r, m.url)
	return nil
}

// NewExternalRedirectResponse creates a new response that redirects the client to the
// external URL.
//
// External URL is any URL that is not powered by Inertia.js.
func NewExternalRedirectResponse(url string) *Response {
	return &Response{
		msg:            &externalRedirectMessage{url: url},
		clearHistory:   false,
		encryptHistory: false,
		concurrency:    inertia.DefaultConcurrency,
	}
}

type redirectBackMessage struct{}

func (m *redirectBackMessage) Component() string { return "" }

func (m *redirectBackMessage) Write(w http.ResponseWriter, r *http.Request) error {
	RedirectBack(w, r)
	return nil
}

// NewRedirectBackResponse creates a new response that redirects the client to the
// back to the previous page.
func NewRedirectBackResponse() *Response {
	return &Response{
		msg:            &redirectBackMessage{},
		clearHistory:   false,
		encryptHistory: false,
		concurrency:    inertia.DefaultConcurrency,
	}
}

type redirectMessage struct{ url string }

func (m *redirectMessage) Component() string { return "" }

func (m *redirectMessage) Write(w http.ResponseWriter, r *http.Request) error {
	RedirectBack(w, r)
	return nil
}

// NewRedirectResponse creates a new response that redirects the client to the
// specified URL.
func NewRedirectResponse(url string) *Response {
	return &Response{
		msg:            &redirectMessage{url: url},
		clearHistory:   false,
		encryptHistory: false,
		concurrency:    inertia.DefaultConcurrency,
	}
}

// Message is used to send a message to the client, it can be
// used to guide the client to render a component, or redirect to a
// specific URL.
//
// If the Message implements a RawResponseWriter, the default
// behavior is prevented and the writer is used instead to
// write the response data.
//
// The Component() method must return a non-empty string.
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
	// HTTP method of the endpoint.
	Method string

	// HTTP path of the endpoint. It supports the same path pattern as
	// the http.ServeMux.
	Path string
}

type Endpoint[R any] interface {
	// Execute executes the endpoint for the given request.
	//
	// If the returned error can automatically be converted to an Inertia
	// error, it will converted and passed down to the client.
	Execute(context.Context, *Request[R]) (*Response, error)

	// Meta is the metadata of the endpoint. It is used to configure
	// the endpoint's behavior when mounted on a given http.ServeMux.
	Meta() *Meta
}

type MountOpts struct {
	// Middleware is the middleware used to handle requests.
	// If middleware is nil, no middleware will be used.
	Middleware httpmiddleware.Middleware

	// Validator is the validator used to validate the request data.
	// If validator is nil, the default validator will be used.
	Validator *validator.Validate

	// FormDecoder is the decoded used to parse incoming request data
	// when the request type is application/x-www-form-urlencoded or
	// multipart/form-data.
	FormDecoder *form.Decoder

	// ErrorHandler is the error handler used to handle errors.
	// If errorHandler is nil, the default error handler will be used.
	ErrorHandler httperror.ErrorHandler
}

// Mount mounts the executor on the given http.ServeMux.
//
// Endpoint must specify the HTTP method and path via Endpoint.Meta().
// The mounted endpoint is automatically handles requests with JSON and form
// data.
//
// The message M is validated using the validator specified in the MountOpts.
// Validation errors are automatically handled and passed to the client
// according to Inertia protocol.
func Mount[M any](mux *http.ServeMux, e Endpoint[M], opts *MountOpts) {
	if opts == nil {
		//nolint:exhaustruct
		opts = &MountOpts{}
	}

	opts.ErrorHandler = cmp.Or(opts.ErrorHandler, DefaultErrorHandler)
	opts.Validator = cmp.Or(opts.Validator, DefaultValidator)
	opts.FormDecoder = cmp.Or(opts.FormDecoder, DefaultFormDecoder)

	m := e.Meta()

	debug.Assert(m.Method != "", "Executor must specify the HTTP method")
	debug.Assert(m.Path != "", "Executor must specify the HTTP path")
	debug.Assert(opts.ErrorHandler != nil, "Executor must specify the error handler")
	debug.Assert(opts.Validator != nil, "Executor must specify the validator")

	pattern := fmt.Sprintf("%s %s", m.Method, m.Path)

	d("Mounting executor on pattern: %s", pattern)

	h := newHandler(e, opts.ErrorHandler, opts.Validator, opts.FormDecoder)
	if opts.Middleware != nil {
		h = opts.Middleware.Middleware(h)
	}

	mux.Handle(pattern, h)
}

// newHandler creates a new http.Handler for the given endpoint.
func newHandler[M any](
	endpoint Endpoint[M],
	errorHandler httperror.ErrorHandler,
	validate *validator.Validate,
	formDecoder *form.Decoder,
) http.Handler {
	handleError := httperror.WithErrorHandler(errorHandler)

	return handleError(httperror.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		var msg M

		ctx := r.Context()

		var renderCtx inertia.RenderContext

		if extract, ok := any(msg).(RawRequestExtractor); ok {
			if err := extract.Extract(r); err != nil {
				return fmt.Errorf("inertiaframe: failed to extract request data: %w", err)
			}
		} else if r.Method != http.MethodGet {
			mediaType, _, err := mime.ParseMediaType(
				r.Header.Get(inertiaheader.HeaderContentType))
			if err != nil {
				return fmt.Errorf("inertiaframe: failed to parse Content-Type header: %w", err)
			}

			// Inertia accepts only JSON, or multipart/form-data.
			switch mediaType {
			case mediaTypeJSON:
				{
					if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
						return fmt.Errorf("inertiaframe: failed to decode request: %w", err)
					}
				}
			case mediaTypeForm, mediaTypeMultipart:
				{
					if err := r.ParseForm(); err != nil {
						return fmt.Errorf("inertiaframe: failed to parse form data: %w", err)
					}

					if err := formDecoder.Decode(&msg, r.Form); err != nil {
						return fmt.Errorf("inertiaframe: failed to decode form data: %w", err)
					}
				}
			}
		}

		if err := validate.StructCtx(ctx, &msg); err != nil {
			return fmt.Errorf("inertiaframe: failed to validate request: %w", err)
		}

		req := &Request[M]{Message: &msg}

		resp, err := endpoint.Execute(ctx, req)
		if err != nil {
			return fmt.Errorf("inertiaframe: failed to execute: %w", err)
		}

		if resp != nil {
			if writer, ok := resp.msg.(RawResponseWriter); ok {
				if err := writer.Write(w, r); err != nil {
					return fmt.Errorf("inertiaframe: failed to write response: %w", err)
				}

				return nil
			}

			if resp.clearHistory {
				renderCtx.ClearHistory = true
			}

			if resp.encryptHistory {
				renderCtx.EncryptHistory = true
			}

			props, err := extractProps(resp.msg)
			if err != nil {
				return fmt.Errorf("inertiaframe: failed to extract props: %w", err)
			}

			if props.Len() > 0 {
				renderCtx.Props = props
			}
		}

		sess, _ := sessionFromRequest(r)

		errors := sess.ValidationErrors()

		if errors != nil {
			renderCtx.ErrorBag = sess.ErrorBag()
			renderCtx.AddValidationErrorer(inertia.ValidationErrors(errors))
		}

		componentName := resp.msg.Component()
		debug.Assert(componentName != "", "component must not be empty, when using non RawResponseWriter")

		if err := inertia.Render(w, r, componentName, renderCtx); err != nil {
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
