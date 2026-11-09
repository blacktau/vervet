package settings

import (
	"log/slog"
	"vervet/internal/infrastructure"
)

func NewTestService(store infrastructure.Store, log *slog.Logger) Service {
	return NewTestServiceWithBuild(store, log, false, nil)
}

func NewTestServiceWithBuild(
	store infrastructure.Store,
	log *slog.Logger,
	isDev bool,
	onLevelChange func(slog.Level),
) Service {
	if log == nil {
		log = slog.Default()
	}
	return &settingsService{
		store:         store,
		log:           log,
		isDev:         isDev,
		onLevelChange: onLevelChange,
	}
}
