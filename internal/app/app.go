// Package app contains the main Vervet application
package app

import (
	"context"
	"fmt"
	"log/slog"
	"vervet/internal/api"
	"vervet/internal/clientregistry"
	"vervet/internal/collections"
	"vervet/internal/connectionStrings"
	"vervet/internal/connections"
	"vervet/internal/databases"
	"vervet/internal/files"
	"vervet/internal/indexes"
	"vervet/internal/workspaces"
	"vervet/internal/models"
	"vervet/internal/oidc"
	"vervet/internal/queryexecutor"
	"vervet/internal/servers"
	"vervet/internal/settings"
	"vervet/internal/system"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	log              *slog.Logger
	ctx              context.Context
	ServersProxy     *api.ServersProxy
	ConnectionsProxy *api.ConnectionsProxy
	DatabasesProxy   *api.DatabasesProxy
	IndexesProxy     *api.IndexesProxy
	CollectionsProxy *api.CollectionsProxy
	ShellProxy       *api.ShellProxy
	SystemProxy      *api.SystemProxy
	SettingsProxy    *api.SettingsProxy
	FilesProxy       *api.FilesProxy
	WorkspacesProxy  *api.WorkspacesProxy

	serverService      *servers.ServerService
	registry           *clientregistry.ClientRegistry
	connectionManager  *connections.ConnectionManager
	databasesService   *databases.DatabasesService
	indexService       *indexes.IndexService
	collectionsService *collections.CollectionsService
	queryExecutor      *queryexecutor.QueryExecutor
	tokenManager       *oidc.TokenManager
	settingsService    settings.Service
	systemService      *system.Service
	filesService       *files.Service
}

// NewApp creates a new App application struct
func NewApp(log *slog.Logger, version string) *App {
	connectionStringsStore := connectionStrings.NewStore(log)
	serverStore, err := servers.NewServerStore(log)
	if err != nil {
		log.Error("Failed to initialize server store", slog.Any("error", err))
		panic(fmt.Errorf("failed to initialize server store: %w", err))
	}
	tokenManager := oidc.NewTokenManager(log, connectionStringsStore)
	serverService := servers.NewService(log, serverStore, connectionStringsStore, tokenManager)
	registry := clientregistry.NewClientRegistry(log, tokenManager)
	connectionManager := connections.NewManager(log, registry, connectionStringsStore, serverService)
	databasesService := databases.NewDatabasesService(log, registry)
	indexService := indexes.NewIndexService(log, registry)
	collectionsService := collections.NewCollectionsService(log, registry)
	settingsService := settings.NewService(log)
	queryExecutor := queryexecutor.NewQueryExecutor(log, registry, connectionStringsStore, settingsService)
	systemService := system.NewSystemService(log)
	fontService := system.NewFontService(log)
	filesService := files.NewService(log)
	workspaceStore, err := workspaces.NewStore(log)
	if err != nil {
		log.Error("Failed to initialize workspace store", slog.Any("error", err))
		panic(fmt.Errorf("failed to initialize workspace store: %w", err))
	}
	workspaceService := workspaces.NewService(log, workspaceStore)

	return &App{
		log:                log,
		serverService:      serverService,
		registry:           registry,
		connectionManager:  connectionManager,
		databasesService:   databasesService,
		indexService:       indexService,
		collectionsService: collectionsService,
		queryExecutor:      queryExecutor,
		tokenManager:       tokenManager,
		settingsService:    settingsService,
		systemService:      systemService,
		filesService:       filesService,
		ServersProxy:       api.NewServersProxy(serverService),
		ConnectionsProxy:   api.NewConnectionsProxy(connectionManager),
		DatabasesProxy:     api.NewDatabasesProxy(databasesService),
		IndexesProxy:       api.NewIndexesProxy(indexService),
		CollectionsProxy:   api.NewCollectionsProxy(collectionsService),
		ShellProxy:         api.NewShellProxy(queryExecutor),
		SystemProxy:        api.NewSystemProxy(systemService),
		SettingsProxy:      api.NewSettingsProxy(settingsService, fontService, version),
		FilesProxy:         api.NewFilesProxy(filesService),
		WorkspacesProxy:    api.NewWorkspacesProxy(workspaceService, settingsService),
	}
}

// Startup is called at application startup
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	a.serverService.Init(ctx)

	a.tokenManager.Init(ctx)
	a.tokenManager.SetOpenBrowser(func(url string) {
		wailsRuntime.BrowserOpenURL(ctx, url)
	})

	a.registry.Init(ctx)

	err := a.connectionManager.Init(ctx)
	if err != nil {
		a.log.Error("Failed to initialize connection manager", slog.Any("error", err))
		panic(fmt.Errorf("failed to initialize connection manager: %w", err))
	}

	a.databasesService.Init(ctx)
	a.indexService.Init(ctx)
	a.collectionsService.Init(ctx)
	a.queryExecutor.Init(ctx)

	err = a.settingsService.Init(ctx)
	if err != nil {
		a.log.Error("Failed to initialize settings service", slog.Any("error", err))
		panic(fmt.Errorf("failed to initialize settings service: %w", err))
	}

	a.filesService.Init(ctx)
	a.WorkspacesProxy.Init(ctx)
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
	cfg, err := a.settingsService.GetSettings()
	if err != nil {
		return settings.DefaultWindowWidth, settings.DefaultWindowHeight
	}
	return cfg.Window.Width, cfg.Window.Height
}

func (a *App) saveWindowState() {
	width, height := wailsRuntime.WindowGetSize(a.ctx)
	x, y := wailsRuntime.WindowGetPosition(a.ctx)

	err := a.settingsService.SaveWindowState(models.WindowState{
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
	// Cancel any in-flight OIDC browser logins (releases the callback listener)
	a.tokenManager.Shutdown()

	// Cancel any in-flight queries
	a.queryExecutor.CloseAll()

	// Disconnect all MongoDB connections
	err := a.connectionManager.DisconnectAll()
	if err != nil {
		a.log.Error("Failed to disconnect from all connections", slog.Any("error", err))
	}
}
