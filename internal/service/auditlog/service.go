package auditlog

import (
	"context"

	"github.com/example/homebrew-api-starter/internal/model"
	"github.com/example/homebrew-api-starter/internal/store"
)

type Service struct {
	store *store.Store
}

func NewService(st *store.Store) *Service {
	return &Service{store: st}
}

func (s *Service) List(ctx context.Context, limit int, cursor int64, action, actorID string) ([]model.AuditLog, int64, error) {
	return s.store.Queries().ListAuditLogs(ctx, limit, cursor, action, actorID)
}
