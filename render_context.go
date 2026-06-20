package inertia

import (
	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
)

// RenderContextOption is a function that configures a RenderContext.
type RenderContextOption func(*RenderContext)

// NewRenderContext creates a RenderContext configured with the provided options.
// Options are applied in order and can be combined to build up the desired page state.
func NewRenderContext(opts ...RenderContextOption) RenderContext {
	var ctx RenderContext
	for _, opt := range opts {
		opt(&ctx)
	}

	return ctx
}

// WithClearHistory instructs the client to clear its history stack when rendering this page.
func WithClearHistory() RenderContextOption {
	return func(opt *RenderContext) { opt.ClearHistory = true }
}

// WithEncryptHistory instructs the client to encrypt the history state.
func WithEncryptHistory() RenderContextOption {
	return func(opt *RenderContext) { opt.EncryptHistory = true }
}

// WithPreserveFragment instructs the client to preserve the current URL fragment.
func WithPreserveFragment() RenderContextOption {
	return func(opt *RenderContext) { opt.PreserveFragment = true }
}

// WithProps adds properties to the page component.
//
// Multiple calls append additional props to the existing set.
func WithProps(props Proper) RenderContextOption {
	return func(renderCtx *RenderContext) {
		if props == nil {
			return
		}

		if renderCtx.Props == nil {
			renderCtx.Props = make([]Prop, 0, props.Len())
		}

		renderCtx.Props = append(renderCtx.Props, props.Props()...)
	}
}

// WithSharedProps adds shared properties to the page component.
func WithSharedProps(props Proper) RenderContextOption {
	return func(renderCtx *RenderContext) {
		if props == nil {
			return
		}

		if renderCtx.SharedProps == nil {
			renderCtx.SharedProps = make([]Prop, 0, props.Len())
		}

		renderCtx.SharedProps = append(renderCtx.SharedProps, props.Props()...)
	}
}

// WithValidationErrors adds validation errors to be displayed on the page.
// Multiple calls append errors to the same or different error bags.
//
// The errorBag parameter allows scoping errors to specific forms on the same page.
func WithValidationErrors(errorers inertiaprop.ValidationErrorer, errorBag string) RenderContextOption {
	return func(renderCtx *RenderContext) {
		if errorers == nil {
			return
		}

		renderCtx.AddValidationErrorer(errorers)
		renderCtx.ErrorBag = errorBag
	}
}
