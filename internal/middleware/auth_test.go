package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/homebrew-api-starter/internal/auth"
	"github.com/example/homebrew-api-starter/internal/platform/jwt"
	"github.com/example/homebrew-api-starter/internal/router"
)

func testRegistry(t *testing.T) *router.Registry {
	t.Helper()
	reg := router.NewRegistry()
	if err := reg.Add(router.Route{Method: http.MethodGet, Path: "/public", OperationID: "public", Public: true}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Add(router.Route{Method: http.MethodGet, Path: "/private", OperationID: "private"}); err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestAuthenticatePublicAndPrivate(t *testing.T) {
	manager, err := jwt.New(strings.Repeat("k", 32), "test", "admin-web", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := manager.Sign("user-1", "session-1", 1)
	if err != nil {
		t.Fatal(err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	chain := Authenticate(testRegistry(t), manager, auth.NewStaticPermissionStore([]string{"article:read"}), nil)(handler)

	cases := []struct {
		name   string
		path   string
		token  string
		status int
	}{
		{name: "public without token", path: "/public", status: http.StatusOK},
		{name: "private without token", path: "/private", status: http.StatusUnauthorized},
		{name: "private with token", path: "/private", token: token, status: http.StatusOK},
		{name: "private with bad token", path: "/private", token: token + "x", status: http.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			chain.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tc.status, rec.Body.String())
			}
		})
	}
}

func TestBearerToken(t *testing.T) {
	cases := []struct {
		header string
		token  string
		ok     bool
	}{
		{"Bearer abc", "abc", true},
		{"bearer abc", "abc", true},
		{"Bearer   abc ", "abc", true},
		{"Basic abc", "", false},
		{"Bearer ", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := bearerToken(tc.header)
		if got != tc.token || ok != tc.ok {
			t.Errorf("bearerToken(%q) = (%q, %v), want (%q, %v)", tc.header, got, ok, tc.token, tc.ok)
		}
	}
}
