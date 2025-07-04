package endpoint

import (
	"context"
	"errors"
	"fmt"
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
	_ inertiaframe.Endpoint[SignInGetRequest]  = (*SignInGetEndpoint)(nil)
	_ inertiaframe.Endpoint[SignInPostRequest] = (*SignInPostEndpoint)(nil)
	_ inertiaframe.RawResponseWriter           = (*SignInPostResponse)(nil)
)

type (
	SignInGetRequest  struct{}
	SignInGetResponse struct {
		Token string `json:"csrf_token" inertia:"csrf_token"`
	}
)

func (*SignInGetResponse) Component() string { return "SignIn" }

type SignInGetEndpoint struct{}

func (s *SignInGetEndpoint) Meta() *inertiaframe.Meta {
	return &inertiaframe.Meta{
		Method: http.MethodGet,
		Path:   "/sign-in",
	}
}

func (s *SignInGetEndpoint) Execute(ctx context.Context, req *inertiaframe.Request[SignInGetRequest]) (*inertiaframe.Response, error) {
	tok, err := shieldcsrf.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	return inertiaframe.NewResponse(&SignInGetResponse{
		Token: tok.String(),
	}), nil
}

type SignInPostRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type SignInPostResponse struct {
	user          *shield.User[user.Data]
	authenticator shieldstrategy.Authenticator[user.Data]
}

func (r *SignInPostResponse) Component() string { return "SignIn" }

type SignInPostEndpoint struct {
	Handler       *shieldpassword.Handler[user.Data]
	Authenticator shieldstrategy.Authenticator[user.Data]
}

func (r *SignInPostResponse) Write(w http.ResponseWriter, req *http.Request) error {
	_, err := r.authenticator.Issue(w, req, r.user)
	if err != nil {
		return err
	}

	fmt.Println("redirecting here")
	inertia.Redirect(w, req, "/")

	return nil
}

func (s *SignInPostEndpoint) Meta() *inertiaframe.Meta {
	return &inertiaframe.Meta{
		Method: http.MethodPost,
		Path:   "/sign-in",
	}
}

func (s *SignInPostEndpoint) Execute(ctx context.Context, req *inertiaframe.Request[SignInPostRequest]) (*inertiaframe.Response, error) {
	user, err := s.Handler.HandleUserLogin(ctx, req.Message.Email, req.Message.Password)
	if err != nil {
		if errors.Is(err, shield.ErrUserNotFound) {
			return nil, inertia.NewValidationError("email",
				"Either email or password is incorrect")
		}

		return nil, err
	}

	return inertiaframe.NewResponse(&SignInPostResponse{
		user,
		s.Authenticator,
	}), nil
}
