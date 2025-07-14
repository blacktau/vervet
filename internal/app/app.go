// Package app contains the main Vervet application
package app

import (
	"context"
	"fmt"
	"log"
	"vervet/internal/connections"
	"vervet/internal/servers"
)

// App struct
type App struct {
	ctx                     context.Context
	RegisteredServerManager *servers.RegisteredServerManager
	ConnectionManager       *connections.ConnectionManager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		RegisteredServerManager: servers.NewRegisteredServerManager(),
		ConnectionManager:       connections.NewConnectionManager(),
	}
}

// Startup is called at application startup
func (a *App) Startup(ctx context.Context) {
	// Perform your setup here
	a.ctx = ctx

	err := a.RegisteredServerManager.Init(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize registered server manager / settings database: %v", err)
	}

	err = a.ConnectionManager.Init(ctx)
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
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
