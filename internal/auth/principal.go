package auth

import (
	"context"
	"sort"
	"strings"
)

// Principal is the authenticated caller for one request. It is built by the
// authentication middleware and consumed by the API layer.
type Principal struct {
	UserID       string
	SessionID    string
	TokenVersion int

	permissions map[string]struct{}
}

func NewPrincipal(userID, sessionID string, tokenVersion int, permissions []string) Principal {
	set := make(map[string]struct{}, len(permissions))
	for _, perm := range permissions {
		perm = strings.TrimSpace(perm)
		if perm != "" {
			set[perm] = struct{}{}
		}
	}
	return Principal{
		UserID:       userID,
		SessionID:    sessionID,
		TokenVersion: tokenVersion,
		permissions:  set,
	}
}

func (p Principal) Has(permission string) bool {
	if permission == "" {
		return true
	}
	_, ok := p.permissions[permission]
	return ok
}

func (p Principal) Permissions() []string {
	out := make([]string, 0, len(p.permissions))
	for perm := range p.permissions {
		out = append(out, perm)
	}
	sort.Strings(out)
	return out
}

type contextKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, p)
}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	if ctx == nil {
		return Principal{}, false
	}
	p, ok := ctx.Value(contextKey{}).(Principal)
	if !ok {
		return Principal{}, false
	}
	return p, true
}

// PermissionStore loads the effective permission set for a user. The P1
// implementation reads the RBAC tables; P0 uses a static dev store.
type PermissionStore interface {
	Permissions(ctx context.Context, userID string) ([]string, error)
}

type StaticPermissionStore struct {
	permissions []string
}

func NewStaticPermissionStore(permissions []string) *StaticPermissionStore {
	clean := make([]string, 0, len(permissions))
	for _, perm := range permissions {
		perm = strings.TrimSpace(perm)
		if perm != "" {
			clean = append(clean, perm)
		}
	}
	return &StaticPermissionStore{permissions: clean}
}

func (s *StaticPermissionStore) Permissions(context.Context, string) ([]string, error) {
	return append([]string(nil), s.permissions...), nil
}
