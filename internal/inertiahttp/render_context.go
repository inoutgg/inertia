package inertiahttp

import "go.segfaultmedaddy.com/inertia/internal/inertiaprop"

var _ Proper = (Props)(nil)

type (
	// Prop represents a single property passed to an Inertia page component.
	//
	// To create a prop, use the specialized prop packages:
	// inertiaprop, inertiaalways, inertiaoptional, inertiadeferred, and inertiascroll.
	//
	// Attach props to a page using WithProps option.
	Prop = inertiaprop.Prop

	// Lazy represents a prop value that is resolved on-demand rather than eagerly.
	Lazy = inertiaprop.Lazy

	// LazyFunc is a function adapter that implements the Lazy interface.
	//
	// It allows using ordinary functions as lazy prop values.
	// The returned value must be JSON-serializable.
	LazyFunc = inertiaprop.LazyFunc
)

// Proper represents a collection of props that can be attached to a render context.
type Proper interface {
	// Props returns the underlying prop slice.
	Props() []Prop

	// Len returns the number of props in the collection.
	Len() int
}

// Props is a slice of Prop that satisfies the Proper interface, suitable for
// use with WithProps and WithSharedProps.
type Props []Prop

func (p Props) Len() int      { return len(p) }
func (p Props) Props() []Prop { return p }

// RenderContext contains all configuration and data for rendering an Inertia.js page response.
// It includes props, validation errors, history management options, and performance settings.
type RenderContext struct {
	// T is custom data passed to the HTML template via html/template.
	T any

	// Props are the properties sent to the page component.
	Props []Prop

	// SharedProps are globally shared properties sent to the page component.
	SharedProps []Prop

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
