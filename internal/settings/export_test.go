package settings

import (
	"context"
	"log/slog"
	"vervet/internal/infrastructure"
)

func NewTestService(store infrastructure.Store, log *slog.Logger, ctx context.Context) Service {
	return &settingsService{
		store: store,
		log:   log,
		ctx:   ctx,
	}
}
