package middleware

import (
	"net"
	"net/http"

	"github.com/example/homebrew-api-starter/internal/platform/reqid"
	"github.com/example/homebrew-api-starter/internal/platform/requestmeta"
)

func RequestMeta() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
				ip = host
			}
			meta := requestmeta.Meta{
				IP:        ip,
				UserAgent: r.UserAgent(),
				RequestID: reqid.From(r.Context()),
			}
			next.ServeHTTP(w, r.WithContext(requestmeta.With(r.Context(), meta)))
		})
	}
}
