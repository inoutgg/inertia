package endpoint

import (
	"context"
	"net/http"

	"go.inout.gg/foundations/must"
	"go.inout.gg/inertia/inertiaframe"
	"go.inout.gg/shield/shieldsession"
	"go.jetify.com/typeid/v2"
)

var _ inertiaframe.Endpoint[HomeRequest] = (*HomeEndpoint)(nil)

type (
	HomeRequest  struct{}
	HomeResponse struct {
		UserID    typeid.TypeID `inertia:"user_id"`
		SessionID typeid.TypeID `inertia:"session_id"`
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
