// Package app contains the main Vervet application
package app

import (
	"context"
	"fmt"
	"log/slog"
	"vervet/internal/api"
	"vervet/internal/clientregistry"
	"vervet/internal/connectionStrings"
	"vervet/internal/connections"
	"vervet/internal/indexes"
	"vervet/internal/models"
	"vervet/internal/servers"
	"vervet/internal/settings"
	"vervet/internal/shellmanager"
	"vervet/internal/system"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	log              *slog.Logger
	ctx              context.Context
	ServersProxy     *api.ServersProxy
	ConnectionsProxy *api.ConnectionsProxy
	IndexesProxy     *api.IndexesProxy
	SystemProxy      *api.SystemProxy
	SettingsProxy    *api.SettingsProxy

	serverManager     *servers.ServerManager
	registry          *clientregistry.ClientRegistry
	connectionManager *connections.ConnectionManager
	indexManager      *indexes.IndexManager
	shellManager      *shellmanager.ShellManager
	settingsManager   settings.Manager
	systemService     *system.Service
}

// NewApp creates a new App application struct
func NewApp(log *slog.Logger) *App {
	serverManager := servers.NewManager(log)
	connectionStringsStore := connectionStrings.NewStore(log)
	registry := clientregistry.NewClientRegistry(log)
	connectionManager := connections.NewManager(log, registry, connectionStringsStore, serverManager)
	indexManager := indexes.NewIndexManager(log, registry)
	settingsManager := settings.NewManager(log)
	shellManager := shellmanager.NewShellManager(log, registry, connectionStringsStore, settingsManager)
	systemService := system.NewSystemService(log)
	fontService := system.NewFontService(log)

	return &App{
		log:               log,
		serverManager:     serverManager,
		registry:          registry,
		connectionManager: connectionManager,
		indexManager:      indexManager,
		shellManager:      shellManager,
		settingsManager:   settingsManager,
		systemService:     systemService,
		ServersProxy:      api.NewServersProxy(serverManager),
		ConnectionsProxy:  api.NewConnectionsProxy(connectionManager, shellManager),
		IndexesProxy:      api.NewIndexesProxy(indexManager),
		SystemProxy:       api.NewSystemProxy(systemService),
		SettingsProxy:     api.NewSettingsProxy(settingsManager, fontService),
	}
}

// Startup is called at application startup
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	err := a.serverManager.Init(ctx)
	if err != nil {
		a.log.Error("Failed to initialize registered server manager", slog.Any("error", err))
		panic(fmt.Errorf("failed to initialize registered server manager: %w", err))
	}

	a.registry.Init(ctx)

	err = a.connectionManager.Init(ctx)
	if err != nil {
		a.log.Error("Failed to initialize connection manager", slog.Any("error", err))
		panic(fmt.Errorf("failed to initialize connection manager: %w", err))
	}

	a.indexManager.Init(ctx)
	a.shellManager.Init(ctx)

	err = a.settingsManager.Init(ctx)
	if err != nil {
		a.log.Error("Failed to initialize settings manager", slog.Any("error", err))
		panic(fmt.Errorf("failed to initialize settings manager: %w", err))
	}

	err = a.systemService.Init(ctx)
	if err != nil {
		a.log.Error("Failed to initialize system service", slog.Any("error", err))
		panic(fmt.Errorf("failed to initialize system service: %w", err))
	}
}

// DomReady is called after front-end resources have been loaded
func (a *App) DomReady(ctx context.Context) {
}

// BeforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	a.saveWindowState()
	return false
}

// GetInitialWindowSize returns the saved window size for use in Wails options before the window is created.
// This reads settings directly rather than using GetWindowState, which requires a Wails context.
func (a *App) GetInitialWindowSize() (width, height int) {
	cfg, err := a.settingsManager.GetSettings()
	if err != nil {
		return settings.DefaultWindowWidth, settings.DefaultWindowHeight
	}
	return cfg.Window.Width, cfg.Window.Height
}

func (a *App) saveWindowState() {
	width, height := wailsRuntime.WindowGetSize(a.ctx)
	x, y := wailsRuntime.WindowGetPosition(a.ctx)

	err := a.settingsManager.SaveWindowState(models.WindowState{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	})
	if err != nil {
		a.log.Error("Failed to save window state", slog.Any("error", err))
	}
}

// Shutdown is called at application termination
func (a *App) Shutdown(ctx context.Context) {
	// Cancel any in-flight mongosh queries
	a.shellManager.CloseAll()

	// Disconnect all MongoDB connections
	err := a.connectionManager.DisconnectAll()
	if err != nil {
		a.log.Error("Failed to disconnect from all connections", slog.Any("error", err))
	}
}
