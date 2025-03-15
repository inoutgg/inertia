package inertia

var (
	_ ValidationError   = (*validationError)(nil)
	_ ValidationErrorer = (*validationError)(nil)
	_ ValidationErrorer = (*ValidationErrors)(nil)
)

const (
	DefaultErrorBag = "default"
)

// ValidationError represents a validation error.
type ValidationError interface {
	// Field returns the field name associated with the validation error.
	Field() string

	// Error returns the error message associated with the validation error.
	Error() string
}

type ValidationErrorer interface {
	ValidationErrors() []ValidationError
	Len() int
}

type validationError struct {
	field    string
	message  string
	errorBag string // optional
}

// NewValidationError creates a new validation error.
//
// opts can be nil.
func NewValidationError(field string, message string) *validationError {
	return &validationError{
		field:    field,
		message:  message,
		errorBag: "",
	}
}

func (err *validationError) Error() string                       { return err.message }
func (err *validationError) Field() string                       { return err.field }
func (err *validationError) ValidationErrors() []ValidationError { return []ValidationError{err} }
func (err *validationError) Len() int                            { return 1 }

type ValidationErrors []ValidationError

func (errs ValidationErrors) ValidationErrors() []ValidationError { return errs }
func (errs ValidationErrors) Len() int                            { return len(errs) }
