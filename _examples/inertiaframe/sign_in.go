package main

import (
	"context"
	"net/http"

	"go.inout.gg/inertia/contrib/inertiaframe"
)

var _ inertiaframe.Endpoint[SignInReq, SignInResp] = (*SignInEndpoint)(nil)

type SignInReq struct {
	Email    string
	Password string
}

type SignInResp struct {
	Token string
}

type SignInEndpoint struct{}

func (s *SignInEndpoint) Meta() *inertiaframe.Meta {
	return &inertiaframe.Meta{
		Method: http.MethodGet,
		Path:   "/sign-in",
	}
}

func (s *SignInEndpoint) Execute(ctx context.Context, req *inertiaframe.Request[SignInReq]) (*inertiaframe.Response[SignInResp], error) {
	return inertiaframe.NewResponse("SignIn", &SignInResp{
		Token: "example_token",
	}), nil
}
