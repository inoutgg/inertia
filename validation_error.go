package inertia

import "go.segfaultmedaddy.com/inertia/internal/inertiaprop"

// DefaultErrorBag is the error bag name used when the client does not
// select a specific bag via the X-Inertia-Error-Bag header and no bag is
// passed to WithValidationErrors.
const DefaultErrorBag = inertiaprop.DefaultErrorBag

// ValidationError represents a single field validation failure.
type ValidationError = inertiaprop.ValidationError

// ValidationErrorer is a collection of validation errors that can be sent to the client.
type ValidationErrorer = inertiaprop.ValidationErrorer

// ValidationErrors is a collection of ValidationError values that implements
// the error and ValidationErrorer interfaces; pass it wherever a
// ValidationErrorer is expected.
//
//nolint:errname // alias of inertiaprop.ValidationErrors
type ValidationErrors = inertiaprop.ValidationErrors

// NewValidationError creates a validation error for a specific field with a message.
// The error is associated with the default error bag.
func NewValidationError(field string, message string) ValidationError {
	return inertiaprop.NewValidationError(field, message)
}
