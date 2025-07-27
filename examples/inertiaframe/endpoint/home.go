package endpoint

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.inout.gg/foundations/must"
	"go.inout.gg/inertia/inertiaframe"
	"go.inout.gg/shield/shieldsession"
)

var _ inertiaframe.Endpoint[HomeRequest] = (*HomeEndpoint)(nil)

type (
	HomeRequest  struct{}
	HomeResponse struct {
		UserID    uuid.UUID `inertia:"user_id"`
		SessionID uuid.UUID `inertia:"session_id"`
	}
)

func (*HomeResponse) Component() string { return "Index" }

type HomeEndpoint struct{}

func (s *HomeEndpoint) Meta() *inertiaframe.Meta {
	return &inertiaframe.Meta{
		Method: http.MethodGet,
		Path:   "/dashboard",
	}
}

func (s *HomeEndpoint) Execute(
	ctx context.Context,
	req *inertiaframe.Request[HomeRequest],
) (*inertiaframe.Response, error) {
	sess := must.Must(shieldsession.FromContext[any](ctx))

	return inertiaframe.NewResponse(&HomeResponse{
		UserID:    sess.UserID,
		SessionID: sess.ID,
	}, nil), nil
}
