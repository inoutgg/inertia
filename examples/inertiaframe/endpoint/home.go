package endpoint

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.inout.gg/examples/inertiaframe/user"
	"go.inout.gg/inertia/inertiaframe"
	"go.inout.gg/shield/shielduser"
)

var _ inertiaframe.Endpoint[HomeRequest] = (*HomeEndpoint)(nil)

type (
	HomeRequest  struct{}
	HomeResponse struct {
		ID uuid.UUID `inertia:"user_id"`
	}
)

func (*HomeResponse) Component() string { return "Index" }

type HomeEndpoint struct{}

func (s *HomeEndpoint) Meta() *inertiaframe.Meta {
	return &inertiaframe.Meta{
		Method: http.MethodGet,
		Path:   "/",
	}
}

func (s *HomeEndpoint) Execute(ctx context.Context, req *inertiaframe.Request[HomeRequest]) (*inertiaframe.Response, error) {
	sess := shielduser.FromContext[user.Data](ctx)

	return inertiaframe.NewResponse(&HomeResponse{
		ID: sess.ID,
	}), nil
}
