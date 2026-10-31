package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/example/homebrew-api-starter/internal/api/shared"
	"github.com/example/homebrew-api-starter/internal/auth"
	"github.com/example/homebrew-api-starter/internal/platform/jwt"
	"github.com/example/homebrew-api-starter/internal/router"
)

// Authenticate verifies the access JWT and attaches the principal. The public
// route set comes from the route registry, so adding a public route cannot
// accidentally leave it behind auth (and vice versa).
func Authenticate(
	reg *router.Registry,
	manager *jwt.Manager,
	permissions auth.PermissionStore,
	sessions auth.SessionValidator,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if reg.IsPublic(r.Method, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			raw, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				writeUnauthorized(w, r, "missing bearer token")
				return
			}
			claims, err := manager.Parse(raw)
			if err != nil {
				writeUnauthorized(w, r, "invalid or expired token")
				return
			}

			if sessions != nil {
				principal, err := sessions.ValidateSession(r.Context(), claims.SessionID, claims.Subject, claims.TokenVersion)
				if errors.Is(err, auth.ErrUnauthenticated) {
					writeUnauthorized(w, r, "session is no longer valid")
					return
				}
				if err != nil {
					shared.WriteProblem(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "session lookup failed")
					return
				}
				next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), principal)))
				return
			}

			var perms []string
			if permissions != nil {
				perms, err = permissions.Permissions(r.Context(), claims.Subject)
				if err != nil {
					shared.WriteProblem(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "permission lookup failed")
					return
				}
			}

			principal := auth.NewPrincipal(claims.Subject, claims.SessionID, claims.TokenVersion, perms)
			next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), principal)))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, r *http.Request, detail string) {
	w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	shared.WriteProblem(w, r, http.StatusUnauthorized, "UNAUTHORIZED", detail)
}

func bearerToken(header string) (string, bool) {
	const prefix = "bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	return token, token != ""
}
