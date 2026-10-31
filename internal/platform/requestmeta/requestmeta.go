package requestmeta

import "context"

type Meta struct {
	IP        string
	UserAgent string
	RequestID string
	TraceID   string
}

type contextKey struct{}

func With(ctx context.Context, meta Meta) context.Context {
	return context.WithValue(ctx, contextKey{}, meta)
}

func From(ctx context.Context) Meta {
	if ctx == nil {
		return Meta{}
	}
	meta, _ := ctx.Value(contextKey{}).(Meta)
	return meta
}
