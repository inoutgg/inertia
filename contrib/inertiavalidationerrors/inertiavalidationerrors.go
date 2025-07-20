package inertiavalidationerrors

import (
	"encoding/gob"
	"errors"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"

	"go.inout.gg/inertia"
)

var (
	_ error                     = (*MapError)(nil)
	_ inertia.ValidationErrorer = (*MapError)(nil)
)

//nolint:gochecknoinits
func init() {
	gob.Register(&MapError{})
}

// MapError is a map of key-value pairs that can be used as validation errors.
// Key is the field name and value is the error message.
type MapError map[string]string

func (m MapError) ValidationErrors() []inertia.ValidationError {
	errors := make([]inertia.ValidationError, 0, len(m))
	for k, v := range m {
		errors = append(errors, inertia.NewValidationError(k, v))
	}

	return errors
}

func (m MapError) Error() string    { return "validation errors" }
func (m MapError) Len() int         { return len(m) }
func (m MapError) ErrorBag() string { return inertia.DefaultErrorBag }

// FromValidationErrors creates a Map from a validator error.
//
// FromValidationErrors supports nested errors implemented via the `Unwrap() error`
// method. Unwrap method returning multiple errors `Unwrap() []error` is not supported.
func FromValidationErrors(err error, t ut.Translator) (MapError, bool) {
	var verr validator.ValidationErrors
	if errors.As(err, &verr) {
		m := make(MapError)

		for _, e := range verr {
			f := e.Field()
			msg := e.Translate(t)
			m[f] = msg
		}

		return m, true
	}

	return nil, false
}
