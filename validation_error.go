package inertia

var (
	_ ValidationError   = (*validationError)(nil)
	_ ValidationErrorer = (*validationError)(nil)
	_ ValidationErrorer = (*wrappedValidationErrorer)(nil)
	_ ValidationErrorer = (*ValidationErrors)(nil)
)

const (
	DefaultErrorBag = ""
)

// ValidationError represents a validation error.
type ValidationError interface {
	// Field returns the field name associated with the validation error.
	Field() string

	// Error returns the error message associated with the validation error.
	Error() string
}

type ValidationErrorer interface {
	// ErrorBag returns the error bag associated with the validation error.
	// If it is empty, no error bag is associated with the validation error.
	ErrorBag() string
	ValidationErrors() []ValidationError
	Len() int
}

// ValidationErrorOptions is the options for creating a validation error.
type ValidationErrorOptions struct {
	// ErrorBag associated with the validation error.
	ErrorBag string
}

type validationError struct {
	field    string
	message  string
	errorBag string // optional
}

// NewValidationError creates a new validation error.
//
// opts can be nil.
func NewValidationError(field string, message string, opts *ValidationErrorOptions) ValidationError {
	err := &validationError{
		field:    field,
		message:  message,
		errorBag: "",
	}
	if opts != nil {
		err.errorBag = opts.ErrorBag
	}
	return err
}

func (err *validationError) Error() string                       { return err.message }
func (err *validationError) Field() string                       { return err.field }
func (err *validationError) ErrorBag() string                    { return err.errorBag }
func (err *validationError) ValidationErrors() []ValidationError { return []ValidationError{err} }
func (err *validationError) Len() int                            { return 1 }

type ValidationErrors []ValidationError

func (errs ValidationErrors) ValidationErrors() []ValidationError { return errs }
func (errs ValidationErrors) Len() int                            { return len(errs) }
func (errs ValidationErrors) ErrorBag() string                    { return "" }

type wrappedValidationErrorer struct {
	errorBag string
	errorer  ValidationErrorer
}

func WithErrorBag(errorBag string, errorer ValidationErrorer) ValidationErrorer {
	return &wrappedValidationErrorer{errorBag, errorer}
}

func (w *wrappedValidationErrorer) ErrorBag() string                    { return w.errorBag }
func (w *wrappedValidationErrorer) ValidationErrors() []ValidationError { return w.ValidationErrors() }
func (w *wrappedValidationErrorer) Len() int                            { return w.Len() }
