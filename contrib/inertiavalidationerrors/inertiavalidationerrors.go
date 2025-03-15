package inertiavalidationerrors

import (
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"go.inout.gg/inertia"
)

var _ inertia.ValidationErrorer = (*Map)(nil)

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

func (m Map) Len() int { return len(m) }

func (m Map) ErrorBag() string { return "" }

// FromValidationErrors creates a Map from a validator error.
//
// FromValidationErrors supports nested errors implemented via the Unwrap() error
// method. Unwrap() []error interface is not supported.
func FromValidationErrors(err error, t ut.Translator) (Map, bool) {
	switch cerr := err.(type) {
	case validator.ValidationErrors:
		{
			m := make(Map)
			for _, e := range cerr {
				f := e.Field()
				msg := e.Translate(t)
				m[f] = msg
			}

			return m, true
		}

	case interface{ Unwrap() error }:
		err = cerr.Unwrap()
		if err == nil {
			return nil, false
		}

		return FromValidationErrors(err, t)

	default:
		return nil, false
	}
}
