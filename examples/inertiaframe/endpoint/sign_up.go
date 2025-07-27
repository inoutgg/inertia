package endpoint

import (
	"context"
	"errors"
	"net/http"

	"go.inout.gg/inertia"
	"go.inout.gg/inertia/inertiaframe"
	"go.inout.gg/shield"
	"go.inout.gg/shield/shieldpassword"
	"go.inout.gg/shield/shieldsession"

	"go.inout.gg/examples/inertiaframe/user"
)

var (
	_ inertiaframe.Endpoint[SignUpGetRequest]  = (*SignUpGetEndpoint)(nil)
	_ inertiaframe.Endpoint[SignUpPostRequest] = (*SignUpPostEndpoint)(nil)
	_ inertiaframe.RawResponseWriter           = (*SignUpPostResponse)(nil)
)

type (
	SignUpGetRequest  struct{}
	SignUpGetResponse struct{}
)

func (*SignUpGetResponse) Component() string { return "SignUp" }

type SignUpGetEndpoint struct{}

func (s *SignUpGetEndpoint) Meta() *inertiaframe.Meta {
	return &inertiaframe.Meta{
		Method: http.MethodGet,
		Path:   "/sign-up",
	}
}

func (s *SignUpGetEndpoint) Execute(
	ctx context.Context,
	req *inertiaframe.Request[SignUpGetRequest],
) (*inertiaframe.Response, error) {
	return inertiaframe.NewResponse(&SignUpGetResponse{}, nil), nil
}

type SignUpPostRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type SignUpPostResponse struct {
	authenticator shieldsession.Authenticator[user.Info, any]
	user          shield.User[user.Info]
}

func (r *SignUpPostResponse) Component() string { return "SignUp" }

type SignUpPostEndpoint struct {
	Handler       *shieldpassword.Handler[user.Info, any]
	Authenticator shieldsession.Authenticator[user.Info, any]
}

func (r *SignUpPostResponse) Write(w http.ResponseWriter, req *http.Request) error {
	_, err := r.authenticator.Issue(w, req, r.user)
	if err != nil {
		return err
	}

	inertia.Redirect(w, req, "/-/dashboard")

	return nil
}

func (s *SignUpPostEndpoint) Meta() *inertiaframe.Meta {
	return &inertiaframe.Meta{
		Method: http.MethodPost,
		Path:   "/sign-up",
	}
}

func (s *SignUpPostEndpoint) Execute(
	ctx context.Context,
	req *inertiaframe.Request[SignUpPostRequest],
) (*inertiaframe.Response, error) {
	user, err := s.Handler.HandleUserRegistration(ctx, req.Message.Email, req.Message.Password)
	if err != nil {
		if errors.Is(err, shieldpassword.ErrEmailAlreadyTaken) {
			return nil, inertia.NewValidationError("email",
				"Email is already taken")
		}

		return nil, err
	}

	return inertiaframe.NewResponse(&SignUpPostResponse{
		authenticator: s.Authenticator,
		user:          user,
	}, nil), nil
}
