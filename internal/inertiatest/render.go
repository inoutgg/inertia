package inertiatest

import (
	"testing"

	"github.com/alitto/pond/v2"

	"go.segfaultmedaddy.com/inertia/inertiaotel"
	"go.segfaultmedaddy.com/inertia/internal/inertiaprotocol"
)

//nolint:gochecknoglobals
var defaultRenderer = inertiaprotocol.New(
	pond.NewResultPool[inertiaprotocol.Result](0),
	inertiaotel.DefaultConfig,
)

// TestRender bypasses the PropTestBuilder, used when asserting on error
// wrapping or Page fields the builder does not surface.
func TestRender(
	t *testing.T,
	req inertiaprotocol.Request,
	opts ...ContextOption,
) (*inertiaprotocol.Page, error) {
	t.Helper()

	ctx := inertiaprotocol.Context{ //nolint:exhaustruct
		Component: DefaultComponent,
		Version:   DefaultVersion,
	}
	for _, opt := range opts {
		opt(&ctx)
	}

	page, err := defaultRenderer.Render(t.Context(), req, ctx)

	return page, err //nolint:wrapcheck
}
