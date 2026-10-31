package middleware

import (
	"net/http"

	"github.com/example/homebrew-api-starter/internal/platform/reqid"
)

const RequestIDHeader = "X-Request-Id"

func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(RequestIDHeader)
			if !reqid.Valid(id) {
				id = reqid.New()
			}
			w.Header().Set(RequestIDHeader, id)
			next.ServeHTTP(w, r.WithContext(reqid.With(r.Context(), id)))
		})
	}
}
