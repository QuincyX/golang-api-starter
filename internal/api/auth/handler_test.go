package authapi

import (
	"context"
	"errors"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	authpkg "github.com/example/homebrew-api-starter/internal/auth"
	"github.com/example/homebrew-api-starter/internal/model"
	authsvc "github.com/example/homebrew-api-starter/internal/service/auth"
)

type fakeLoginService struct {
	result *authsvc.LoginResult
	err    error
}

func (f fakeLoginService) Login(context.Context, string, string) (*authsvc.LoginResult, error) {
	return f.result, f.err
}

func (fakeLoginService) Logout(context.Context, authpkg.Principal) error { return nil }

func (fakeLoginService) ChangePassword(context.Context, string, string, string) error { return nil }

func TestLoginSuccess(t *testing.T) {
	h := &Handler{svc: fakeLoginService{result: &authsvc.LoginResult{
		AccessToken: "token",
		ExpiresIn:   3600,
		User:        &model.User{ID: "u1", Email: "a@b.c", DisplayName: "A"},
		Permissions: []string{"article:read"},
		Roles:       []string{"editor"},
	}}}
	var in LoginInput
	in.Body.Email = "a@b.c"
	in.Body.Password = "secret"

	out, err := h.login(context.Background(), &in)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if out.Body.AccessToken != "token" || out.Body.ExpiresIn != 3600 {
		t.Fatalf("unexpected output: %+v", out.Body)
	}
	if out.Body.User.ID != "u1" || len(out.Body.Permissions) != 1 {
		t.Fatalf("user/permissions not mapped: %+v", out.Body)
	}
}

func TestLoginMapsServiceErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"invalid credentials", authsvc.ErrInvalidCredentials, 401},
		{"rate limited", authsvc.ErrRateLimited, 429},
		{"internal", errors.New("db down"), 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := &Handler{svc: fakeLoginService{err: tc.err}}
			var in LoginInput
			in.Body.Email = "a@b.c"
			in.Body.Password = "secret"
			_, err := h.login(context.Background(), &in)
			var problem *huma.ErrorModel
			if !errors.As(err, &problem) {
				t.Fatalf("expected huma problem, got %T", err)
			}
			if problem.Status != tc.want {
				t.Fatalf("status = %d, want %d", problem.Status, tc.want)
			}
		})
	}
}
