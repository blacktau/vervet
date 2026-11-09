// Package app contains the main Vervet application
package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"vervet/internal/api"
	"vervet/internal/clientregistry"
	"vervet/internal/closeguard"
	"vervet/internal/collections"
	"vervet/internal/connectionStrings"
	"vervet/internal/connections"
	"vervet/internal/databases"
	"vervet/internal/desktop"
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

	"github.com/wailsapp/wails/v3/pkg/application"
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

// updatesEmitter adapts desktop.Runtime to updates.Emitter.
type updatesEmitter struct{ rt *desktop.Runtime }

func (e updatesEmitter) EmitEvent(name string, data any) { e.rt.Emit(name, data) }

// urlOpener adapts desktop.Runtime to api.URLOpener, logging failures.
type urlOpener struct {
	rt  *desktop.Runtime
	log *slog.Logger
}

func (o urlOpener) OpenURL(url string) {
	if err := o.rt.OpenURL(url); err != nil {
		o.log.Warn("failed to open URL in the browser", slog.Any("error", err))
	}
}

// mainWindow is what App needs from the main window. desktop.Runtime
// provides it; tests use a fake.
type mainWindow interface {
	Emit(name string, data ...any)
	RevealWindow()
	CloseWindow()
	OnWindowClosing(fn func() (keepOpen bool))
	WindowBounds() (x, y, width, height int, ok bool)
}

// App struct
type App struct {
	log              *slog.Logger
	closeGuard       *closeguard.Guard
	runtime          *desktop.Runtime
	window           mainWindow
	ctx              context.Context
	cancel           context.CancelFunc
	ServersProxy     *api.ServersProxy
	ConnectionsProxy *api.ConnectionsProxy
	DatabasesProxy   *api.DatabasesProxy
	IndexesProxy     *api.IndexesProxy
	CollectionsProxy *api.CollectionsProxy
	ShellProxy       *api.ShellProxy
	SystemProxy      *api.SystemProxy
	SettingsProxy    *api.SettingsProxy
	FilesProxy       *api.FilesProxy
	CloseGuardProxy  *api.CloseGuardProxy
	WorkspacesProxy  *api.WorkspacesProxy
	UpdatesProxy     *api.UpdatesProxy
	ExportProxy      *api.ExportProxy
	OIDCProxy        *api.OIDCProxy

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
	exportService      *export.Service
	updatesService     *updates.Service
	appVersion         string
}

// NewApp creates a new App application struct
func NewApp(log *slog.Logger, settingsService settings.Service, version string) *App {
	rt := desktop.New()
	connectionStringsStore := connectionStrings.NewStore(log)
	serverStore, err := servers.NewServerStore(log)
	if err != nil {
		log.Error("Failed to initialize server store", slog.Any("error", err))
		panic(fmt.Errorf("failed to initialize server store: %w", err))
	}
	tokenManager := oidc.NewTokenManager(log, connectionStringsStore)
	opener := urlOpener{rt: rt, log: log}
	tokenManager.SetOpenBrowser(opener.OpenURL)
	tokenManager.SetEmitAuthURL(func(serverID, url string) {
		rt.Emit("oidc-auth-url", map[string]string{
			"serverID": serverID,
			"url":      url,
		})
	})
	serverService := servers.NewService(log, serverStore, connectionStringsStore, tokenManager)
	serverService.SetEmitter(rt.Emit)
	registry := clientregistry.NewClientRegistry(log, tokenManager)
	connectionManager := connections.NewManager(log, registry, connectionStringsStore, serverService)
	connectionManager.SetEmitter(rt.Emit)
	serverService.SetDisconnector(connectionManager)
	settingsService.SetScreenSizeProvider(rt.ScreenSize)
	databasesService := databases.NewDatabasesService(log, registry)
	indexService := indexes.NewIndexService(log, registry)
	collectionsService := collections.NewCollectionsService(log, registry)
	updatesService := updates.NewService(log, updates.Config{
		CurrentVersion: version,
		Settings:       updates.NewSettingsAdapter(settingsService),
		Emitter:        updatesEmitter{rt: rt},
	})
	queryExecutor := queryexecutor.NewQueryExecutor(log, registry, connectionStringsStore, settingsService)
	systemService := system.NewSystemService(log)
	fontService := system.NewFontService(log)
	filesService := files.NewService(log, rt)
	exportService := export.NewService(log, &filesServiceSaveDialogAdapter{svc: filesService})
	workspaceStore, err := workspaces.NewStore(log)
	if err != nil {
		log.Error("Failed to initialize workspace store", slog.Any("error", err))
		panic(fmt.Errorf("failed to initialize workspace store: %w", err))
	}
	workspaceService := workspaces.NewService(log, workspaceStore)

	closeGuard := closeguard.New(log)

	return &App{
		log:                log,
		runtime:            rt,
		window:             rt,
		closeGuard:         closeGuard,
		CloseGuardProxy:    api.NewCloseGuardProxy(log, closeGuard),
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
		WorkspacesProxy:    api.NewWorkspacesProxy(log, workspaceService, settingsService, rt),
		ExportProxy:        api.NewExportProxy(log, exportService),
		OIDCProxy:          api.NewOIDCProxy(log, tokenManager),
		UpdatesProxy:       api.NewUpdatesProxy(log, updatesService, opener),
		appVersion:         version,
		updatesService:     updatesService,
	}
}

// Services lists what Wails binds and starts. The lifecycle service comes
// first so the backend is initialised before any proxy's ServiceStartup.
func (a *App) Services() []application.Service {
	return []application.Service{
		application.NewService(&lifecycle{app: a}),
		application.NewService(a.ServersProxy),
		application.NewService(a.ConnectionsProxy),
		application.NewService(a.DatabasesProxy),
		application.NewService(a.IndexesProxy),
		application.NewService(a.CollectionsProxy),
		application.NewService(a.ShellProxy),
		application.NewService(a.SystemProxy),
		application.NewService(a.SettingsProxy),
		application.NewService(a.FilesProxy),
		application.NewService(a.CloseGuardProxy),
		application.NewService(a.WorkspacesProxy),
		application.NewService(a.UpdatesProxy),
		application.NewService(a.ExportProxy),
		application.NewService(a.OIDCProxy),
	}
}

// Attach connects the backend to the Wails app and its main window. Call it
// after creating the window and before app.Run.
func (a *App) Attach(wailsApp *application.App, window *application.WebviewWindow) {
	a.runtime.Attach(wailsApp, window)
	a.guardWindow()
}

// guardWindow routes closing the main window through the close guard, and
// saves the window's size and position when it does close.
func (a *App) guardWindow() {
	a.closeGuard.Init(
		func() {
			// Closing from the taskbar can happen while minimised, where the prompt can't be seen.
			a.window.RevealWindow()
			a.window.Emit(closeguard.EventCloseRequested)
		},
		a.window.CloseWindow,
	)

	a.window.OnWindowClosing(func() bool {
		if a.closeGuard.BeforeClose() {
			return true
		}
		a.saveWindowState()
		return false
	})
}

// ShouldQuit answers an app-level quit (e.g. Cmd+Q on macOS), which doesn't
// close the window first, so it applies the same guard as closing it.
func (a *App) ShouldQuit() bool {
	if a.closeGuard.BeforeClose() {
		return false
	}
	a.saveWindowState()
	return true
}

// lifecycle starts and stops the backend with the Wails app. It has no
// exported methods besides the Wails hooks, so nothing is bound for it.
type lifecycle struct{ app *App }

func (l *lifecycle) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	return l.app.startup()
}

func (l *lifecycle) ServiceShutdown() error {
	l.app.shutdown()
	return nil
}

func (a *App) startup() error {
	// Wails cancels the service context before shutdown, but disconnecting
	// from MongoDB during shutdown still needs a live one, so own it here.
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	a.cancel = cancel

	a.tokenManager.Init(ctx)
	a.registry.Init(ctx)

	if err := a.connectionManager.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize connection manager: %w", err)
	}

	a.databasesService.Init(ctx)
	a.indexService.Init(ctx)
	a.collectionsService.Init(ctx)
	a.queryExecutor.Init(ctx)

	if err := a.settingsService.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize settings service: %w", err)
	}

	a.exportService.Init(ctx)
	a.systemService.Init(ctx)

	go a.runUpdateChecks(ctx)
	return nil
}

// updateCheckTick is how often the background timer asks the updates service
// whether a check is due. The configured frequency (daily/weekly) is enforced
// by the service itself; this only needs to be finer-grained than that.
const updateCheckTick = time.Hour

// runUpdateChecks re-checks for updates while the app stays open, so a
// long-running instance honours the daily/weekly frequency instead of only
// ever checking at startup.
func (a *App) runUpdateChecks(ctx context.Context) {
	ticker := time.NewTicker(updateCheckTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.updatesService.CheckPeriodic(ctx); err != nil {
				a.log.Warn("periodic update check failed", slog.Any("error", err))
			}
		}
	}
}

// GetInitialWindowSize returns the saved window size for creating the window.
func (a *App) GetInitialWindowSize() (width, height int) {
	cfg, err := a.settingsService.GetSettings()
	if err != nil {
		return settings.DefaultWindowWidth, settings.DefaultWindowHeight
	}
	return cfg.Window.Width, cfg.Window.Height
}

func (a *App) saveWindowState() {
	if a.window == nil {
		return
	}
	// Not ok when the window is already gone, e.g. the app quits after it closed.
	x, y, width, height, ok := a.window.WindowBounds()
	if !ok {
		return
	}

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

func (a *App) shutdown() {
	if a.cancel != nil {
		defer a.cancel()
	}

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
