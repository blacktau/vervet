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
	"vervet/internal/export"
	"vervet/internal/files"
	"vervet/internal/indexes"
	"vervet/internal/models"
	"vervet/internal/oidc"
	"vervet/internal/queryexecutor"
	"vervet/internal/servers"
	"vervet/internal/settings"
	"vervet/internal/system"
	"vervet/internal/updates"
	"vervet/internal/workspaces"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// filesServiceSaveDialogAdapter adapts files.Service to export.SaveDialog,
// converting between api.FileFilter and export.FileFilter.
type filesServiceSaveDialogAdapter struct {
	svc *files.Service
}

func (a *filesServiceSaveDialogAdapter) SaveFile(title *string, name *string, filters []export.FileFilter) (string, error) {
	apiFilters := make([]api.FileFilter, len(filters))
	for i, f := range filters {
		apiFilters[i] = api.FileFilter{
			DisplayName: f.DisplayName,
			Pattern:     f.Pattern,
		}
	}
	return a.svc.SaveFile(title, name, apiFilters)
}

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
	UpdatesProxy     *api.UpdatesProxy
	ExportProxy      *api.ExportProxy

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
	exportService      *export.Service
	updatesService     *updates.Service
	updatesEmitter     *updates.WailsEmitter
	updatesOpener      *updates.BrowserOpener
	appVersion         string
}

// NewApp creates a new App application struct
func NewApp(log *slog.Logger, settingsService settings.Service, version string) *App {
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
	updatesEmitter := updates.NewWailsEmitter(nil)
	updatesOpener := updates.NewBrowserOpener(nil)
	updatesService := updates.NewService(log, updates.Config{
		CurrentVersion: version,
		Settings:       updates.NewSettingsAdapter(settingsService),
		Emitter:        updatesEmitter,
	})
	queryExecutor := queryexecutor.NewQueryExecutor(log, registry, connectionStringsStore, settingsService)
	systemService := system.NewSystemService(log)
	fontService := system.NewFontService(log)
	filesService := files.NewService(log)
	exportService := export.NewService(log, &filesServiceSaveDialogAdapter{svc: filesService})
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
		exportService:      exportService,
		ServersProxy:       api.NewServersProxy(log, serverService),
		ConnectionsProxy:   api.NewConnectionsProxy(log, connectionManager),
		DatabasesProxy:     api.NewDatabasesProxy(log, databasesService),
		IndexesProxy:       api.NewIndexesProxy(log, indexService),
		CollectionsProxy:   api.NewCollectionsProxy(log, collectionsService),
		ShellProxy:         api.NewShellProxy(log, queryExecutor),
		SystemProxy:        api.NewSystemProxy(log, systemService),
		SettingsProxy:      api.NewSettingsProxy(log, settingsService, fontService, version),
		FilesProxy:         api.NewFilesProxy(log, filesService),
		WorkspacesProxy:    api.NewWorkspacesProxy(log, workspaceService, settingsService),
		ExportProxy:        api.NewExportProxy(log, exportService),
		UpdatesProxy:       api.NewUpdatesProxy(log, updatesService, updatesOpener),
		appVersion:         version,
		updatesService:     updatesService,
		updatesEmitter:     updatesEmitter,
		updatesOpener:      updatesOpener,
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
	a.exportService.Init(ctx)
	a.systemService.Init(ctx)
	a.WorkspacesProxy.Init(ctx)

	a.updatesEmitter.SetContext(ctx)
	a.updatesOpener.SetContext(ctx)
	a.UpdatesProxy.Init(ctx)

	go func() {
		if err := a.updatesService.CheckIfDue(ctx); err != nil {
			a.log.Warn("update check failed", slog.Any("error", err))
		}
	}()
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
