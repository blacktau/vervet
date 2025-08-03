// Package app contains the main Vervet application
package app

import (
	"context"
	"log"
	"vervet/internal/api"
	"vervet/internal/connections"
	"vervet/internal/servers"
)

// App struct
type App struct {
	ctx              context.Context
	ServersProxy     *api.ServersProxy
	ConnectionsProxy *api.ConnectionsProxy
	SystemProxy      *api.SystemProxy

	serverManager     *servers.ServerManager
	connectionManager *connections.ConnectionManager
}

// NewApp creates a new App application struct
func NewApp() *App {
	serverManager := servers.NewRegisteredServerManager()
	connectionManager := connections.NewConnectionManager()

	return &App{
		serverManager:     serverManager,
		connectionManager: connectionManager,
		ServersProxy:      api.NewServersProxy(serverManager),
		ConnectionsProxy:  api.NewConnectionsProxy(connectionManager),
		SystemProxy:       api.NewSystemProxy(),
	}
}

// Startup is called at application startup
func (a *App) Startup(ctx context.Context) {
	// Perform your setup here
	a.ctx = ctx

	err := a.serverManager.Init(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize registered server manager / settings database: %v", err)
	}

	err = a.connectionManager.Init(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize connection manager: %v", err)
	}
}

// DomReady is called after front-end resources have been loaded
func (a App) DomReady(ctx context.Context) {
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
	a.connectionManager.DisconnectAll()
}
