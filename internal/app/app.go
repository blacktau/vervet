// Package app contains the main Vervet application
package app

import (
	"context"
	"fmt"
	"log/slog"
	"vervet/internal/api"
	"vervet/internal/connectionStrings"
	"vervet/internal/connections"
	"vervet/internal/servers"
	"vervet/internal/settings"
	"vervet/internal/system"
)

// App struct
type App struct {
	log              *slog.Logger
	ctx              context.Context
	ServersProxy     *api.ServersProxy
	ConnectionsProxy *api.ConnectionsProxy
	SystemProxy      *api.SystemProxy
	SettingsProxy    *api.SettingsProxy

	serverManager     servers.Manager
	connectionManager connections.Manager
	settingsManager   settings.Manager
	systemService     system.Service
}

// NewApp creates a new App application struct
func NewApp(log *slog.Logger) *App {
	serverManager := servers.NewManager(log)
	connectionStringsStore := connectionStrings.NewStore(log)
	connectionManager := connections.NewManager(log, connectionStringsStore)
	settingsManager := settings.NewManager(log)
	systemService := system.NewSystemService(log)

	return &App{
		log:               log,
		serverManager:     serverManager,
		connectionManager: connectionManager,
		settingsManager:   settingsManager,
		systemService:     systemService,
		ServersProxy:      api.NewServersProxy(serverManager),
		ConnectionsProxy:  api.NewConnectionsProxy(connectionManager),
		SystemProxy:       api.NewSystemProxy(systemService),
		SettingsProxy:     api.NewSettingsProxy(settingsManager),
	}
}

// Startup is called at application startup
func (a *App) Startup(ctx context.Context) {
	// Perform your setup here
	a.ctx = ctx

	err := a.serverManager.Init(ctx)
	if err != nil {
		a.log.Error("Failed to initialize registered server manager / settings database", slog.Any("error", err))
		panic(fmt.Errorf("failed to initialize registered server manager / settings database: %w", err))
	}

	err = a.connectionManager.Init(ctx)
	if err != nil {
		a.log.Error("Failed to initialize connection manager", slog.Any("error", err))
		panic(fmt.Errorf("failed to initialize connection manager: %w", err))
	}

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
	// Add your action here
}

// BeforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	return false
}

// Shutdown is called at application termination
func (a *App) Shutdown(ctx context.Context) {
	// Perform your teardown here
	_ = a.connectionManager.DisconnectAll()
}
