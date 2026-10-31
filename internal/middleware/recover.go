package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/example/homebrew-api-starter/internal/api/shared"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.ErrorContext(r.Context(), "panic recovered",
						"panic", rec,
						"stack", string(debug.Stack()),
					)
					shared.WriteProblem(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
