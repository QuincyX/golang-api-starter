package logx

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a JSON slog logger writing to stdout. Container log collectors
// handle rotation; the app never writes log files itself.
func New(level string) *slog.Logger {
	var l slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		l = slog.LevelDebug
	case "warn", "warning":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}
