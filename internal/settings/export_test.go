package settings

import (
	"context"
	"log/slog"
	"vervet/internal/infrastructure"
)

func NewTestManager(store infrastructure.Store, log *slog.Logger, ctx context.Context) Manager {
	return &settingsManager{
		store: store,
		log:   log,
		ctx:   ctx,
	}
}
