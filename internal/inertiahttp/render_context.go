package inertiahttp

import "go.segfaultmedaddy.com/inertia/internal/inertiaprop"

// RenderContext contains all configuration and data for rendering an Inertia.js page response.
// It includes props, validation errors, history management options, and performance settings.
type RenderContext struct {
	// T is custom data passed to the HTML template via html/template.
	T any

	// Props are the properties sent to the page component.
	Props []inertiaprop.Prop

	// SharedProps are globally shared properties sent to the page component.
	SharedProps []inertiaprop.Prop

	// ErrorBag specifies the validation error bag name for scoped error handling.
	ErrorBag string

	// ValidationErrorer contains validation errors to be sent to the client.
	ValidationErrorer []inertiaprop.ValidationErrorer

	// EncryptHistory instructs the client to encrypt the history state for this page.
	EncryptHistory bool

	// ClearHistory instructs the client to clear the history stack.
	ClearHistory bool

	// PreserveFragment instructs the client to preserve the current URL fragment.
	PreserveFragment bool
}

// AddValidationErrorer appends validation errors to the context.
// Multiple calls accumulate errors into a single error bag.
func (ctx *RenderContext) AddValidationErrorer(err inertiaprop.ValidationErrorer) {
	if ctx.ValidationErrorer == nil {
		ctx.ValidationErrorer = make([]inertiaprop.ValidationErrorer, 0, 1)
	}

	ctx.ValidationErrorer = append(ctx.ValidationErrorer, err)
}
