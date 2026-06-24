package inertiatest

import (
	"go.segfaultmedaddy.com/inertia/internal/inertiaprop"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprotocol"
)

// ContextOption is a function modifying the inertia protocol request context
// during PropTestBuilder run.
type ContextOption func(*inertiaprotocol.Context)

func WithProps(props ...inertiaprop.Prop) ContextOption {
	return func(ctx *inertiaprotocol.Context) { ctx.Props = props }
}

func WithSharedProps(props ...inertiaprop.Prop) ContextOption {
	return func(ctx *inertiaprotocol.Context) { ctx.SharedProps = props }
}

func WithPreserveFragment(v bool) ContextOption {
	return func(ctx *inertiaprotocol.Context) { ctx.PreserveFragment = v }
}

func WithClearHistory(v bool) ContextOption {
	return func(ctx *inertiaprotocol.Context) { ctx.ClearHistory = v }
}

func WithEncryptHistory(v bool) ContextOption {
	return func(ctx *inertiaprotocol.Context) { ctx.EncryptHistory = v }
}
