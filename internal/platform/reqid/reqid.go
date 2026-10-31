package reqid

import (
	"context"

	"github.com/google/uuid"
)

type contextKey struct{}

// New returns a UUIDv7 request id. UUIDv7 is time-ordered, which makes request
// ids sortable in logs.
func New() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}

func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

func From(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(contextKey{}).(string)
	return id
}

// Valid accepts a caller-supplied request id only when it is short and safe.
func Valid(id string) bool {
	if len(id) < 8 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}
