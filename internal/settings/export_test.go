package settings

import (
	"context"
	"vervet/internal/infrastructure"

	"github.com/wailsapp/wails/v2/pkg/logger"
)

func NewTestManager(store infrastructure.Store, log logger.Logger, ctx context.Context) Manager {
	return &settingsManager{
		store: store,
		log:   log,
		ctx:   ctx,
	}
}
