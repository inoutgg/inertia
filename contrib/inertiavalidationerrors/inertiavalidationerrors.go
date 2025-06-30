package inertiavalidationerrors

import (
	"encoding/gob"
	"errors"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"

	"go.inout.gg/inertia"
)

var (
	_ error                     = (*Map)(nil)
	_ inertia.ValidationErrorer = (*Map)(nil)
)

//nolint:gochecknoinits
func init() {
	gob.Register(&Map{})
}

// Map is a map of key-value pairs that can be used as validation errors.
// Key is the field name and value is the error message.
type Map map[string]string

func (m Map) ValidationErrors() []inertia.ValidationError {
	errors := make([]inertia.ValidationError, 0, len(m))
	for k, v := range m {
		errors = append(errors, inertia.NewValidationError(k, v))
	}

	return errors
}

func (m Map) Error() string    { return "validation errors" }
func (m Map) Len() int         { return len(m) }
func (m Map) ErrorBag() string { return inertia.DefaultErrorBag }

// FromValidationErrors creates a Map from a validator error.
//
// FromValidationErrors supports nested errors implemented via the `Unwrap() error`
// method. Unwrap method returning multiple errors `Unwrap() []error` is not supported.
func FromValidationErrors(err error, t ut.Translator) (Map, bool) {
	var verr validator.ValidationErrors
	if errors.As(err, &verr) {
		m := make(Map)

		for _, e := range verr {
			f := e.Field()
			msg := e.Translate(t)
			m[f] = msg
		}

		return m, true
	}

	return nil, false
}
