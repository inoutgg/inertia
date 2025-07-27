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
	_ inertiaframe.Endpoint[SignInGetRequest]  = (*SignInGetEndpoint)(nil)
	_ inertiaframe.Endpoint[SignInPostRequest] = (*SignInPostEndpoint)(nil)
	_ inertiaframe.RawResponseWriter           = (*SignInPostResponse)(nil)
)

type (
	SignInGetRequest  struct{}
	SignInGetResponse struct{}
)

func (*SignInGetResponse) Component() string { return "SignIn" }

type SignInGetEndpoint struct{}

func (s *SignInGetEndpoint) Meta() *inertiaframe.Meta {
	return &inertiaframe.Meta{
		Method: http.MethodGet,
		Path:   "/sign-in",
	}
}

func (s *SignInGetEndpoint) Execute(
	ctx context.Context,
	req *inertiaframe.Request[SignInGetRequest],
) (*inertiaframe.Response, error) {
	return inertiaframe.NewResponse(&SignInGetResponse{}, nil), nil
}

type SignInPostRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type SignInPostResponse struct {
	authenticator shieldsession.Authenticator[user.Info, any]
	user          shield.User[user.Info]
}

func (r *SignInPostResponse) Component() string { return "SignIn" }

type SignInPostEndpoint struct {
	Handler       *shieldpassword.Handler[user.Info, any]
	Authenticator shieldsession.Authenticator[user.Info, any]
}

func (r *SignInPostResponse) Write(w http.ResponseWriter, req *http.Request) error {
	_, err := r.authenticator.Issue(w, req, r.user)
	if err != nil {
		return err
	}

	inertia.Redirect(w, req, "/-/dashboard")

	return nil
}

func (s *SignInPostEndpoint) Meta() *inertiaframe.Meta {
	return &inertiaframe.Meta{
		Method: http.MethodPost,
		Path:   "/sign-in",
	}
}

func (s *SignInPostEndpoint) Execute(
	ctx context.Context,
	req *inertiaframe.Request[SignInPostRequest],
) (*inertiaframe.Response, error) {
	user, err := s.Handler.HandleUserLogin(ctx, req.Message.Email, req.Message.Password)
	if err != nil {
		if errors.Is(err, shield.ErrUserNotFound) ||
			errors.Is(err, shieldpassword.ErrPasswordIncorrect) {
			return nil, inertia.NewValidationError("email",
				"Either email or password is incorrect")
		}

		return nil, err
	}

	return inertiaframe.NewResponse(&SignInPostResponse{
		user:          user,
		authenticator: s.Authenticator,
	}, nil), nil
}
