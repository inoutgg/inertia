package inertiavalidationerrors

import "go.inout.gg/inertia"

var _ inertia.ValidationErrorer = (*Map)(nil)

// Map is a map of key-value pairs that can be used as validation errors.
// Key is the field name and value is the error message.
type Map map[string]string

func (m Map) ValidationErrors() []inertia.ValidationError {
	errors := make([]inertia.ValidationError, 0, len(m))
	for k, v := range m {
		errors = append(errors, inertia.NewValidationError(k, v, nil))
	}

	return errors
}

func (m Map) Len() int { return len(m) }

func (m Map) ErrorBag() string { return "" }
