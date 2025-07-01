package endpoint

import (
	"context"
	"errors"
	"net/http"

	"go.inout.gg/examples/inertiaframe/user"
	"go.inout.gg/inertia"
	"go.inout.gg/inertia/inertiaframe"
	"go.inout.gg/shield"
	"go.inout.gg/shield/shieldcsrf"
	"go.inout.gg/shield/shieldpassword"
	"go.inout.gg/shield/shieldstrategy"
)

var (
	_ inertiaframe.Endpoint[SignUpGetRequest]  = (*SignUpGetEndpoint)(nil)
	_ inertiaframe.Endpoint[SignUpPostRequest] = (*SignUpPostEndpoint)(nil)
	_ inertiaframe.RawResponseWriter           = (*SignUpPostResponse)(nil)
)

type (
	SignUpGetRequest  struct{}
	SignUpGetResponse struct {
		Token string `json:"csrf_token" inertia:"csrf_token"`
	}
)

func (*SignUpGetResponse) Component() string { return "SignUp" }

type SignUpGetEndpoint struct{}

func (s *SignUpGetEndpoint) Meta() *inertiaframe.Meta {
	return &inertiaframe.Meta{
		Method: http.MethodGet,
		Path:   "/sign-up",
	}
}

func (s *SignUpGetEndpoint) Execute(ctx context.Context, req *inertiaframe.Request[SignUpGetRequest]) (*inertiaframe.Response, error) {
	tok, err := shieldcsrf.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	return inertiaframe.NewResponse(&SignUpGetResponse{
		Token: tok.String(),
	}), nil
}

type SignUpPostRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type SignUpPostResponse struct {
	user          *shield.User[user.Data]
	authenticator shieldstrategy.Authenticator[user.Data]
}

func (r *SignUpPostResponse) Component() string { return "SignUp" }

type SignUpPostEndpoint struct {
	Handler       *shieldpassword.Handler[user.Data]
	Authenticator shieldstrategy.Authenticator[user.Data]
}

func (r *SignUpPostResponse) Write(w http.ResponseWriter, req *http.Request) error {
	_, err := r.authenticator.Issue(w, req, r.user)
	if err != nil {
		return err
	}

	inertia.Redirect(w, req, "/")

	return nil
}

func (s *SignUpPostEndpoint) Meta() *inertiaframe.Meta {
	return &inertiaframe.Meta{
		Method: http.MethodPost,
		Path:   "/sign-up",
	}
}

func (s *SignUpPostEndpoint) Execute(ctx context.Context, req *inertiaframe.Request[SignUpPostRequest]) (*inertiaframe.Response, error) {
	user, err := s.Handler.HandleUserRegistration(ctx, req.Message.Email, req.Message.Password)
	if err != nil {
		if errors.Is(err, shieldpassword.ErrEmailAlreadyTaken) {
			return nil, inertia.NewValidationError("email",
				"Email is already taken")
		}

		return nil, err
	}

	return inertiaframe.NewResponse(&SignUpPostResponse{
		user,
		s.Authenticator,
	}), nil
}
