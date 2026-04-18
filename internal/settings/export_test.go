package settings

import (
	"context"
	"log/slog"
	"vervet/internal/infrastructure"
)

func NewTestService(store infrastructure.Store, log *slog.Logger, ctx context.Context) Service {
	return NewTestServiceWithBuild(store, log, ctx, false, nil)
}

func NewTestServiceWithBuild(
	store infrastructure.Store,
	log *slog.Logger,
	ctx context.Context,
	isDev bool,
	onLevelChange func(slog.Level),
) Service {
	if log == nil {
		log = slog.Default()
	}
	return &settingsService{
		store:         store,
		log:           log,
		ctx:           ctx,
		isDev:         isDev,
		onLevelChange: onLevelChange,
	}
}
