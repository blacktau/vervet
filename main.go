package main

import (
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"vervet/internal/api"
	"vervet/internal/app"
	"vervet/internal/logging"
	"vervet/internal/settings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

var version = "dev"

func main() {
	debugUI := flag.Bool("debug-ui", false, "enable ui inspector")
	flag.Parse()

	isDev := version == "dev"

	// Bootstrap logger before everything else
	bootstrap := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Build settings service
	settingsService := settings.NewService(bootstrap, isDev)
	cfg, err := settingsService.GetSettings()
	if err != nil {
		bootstrap.Warn("failed to load settings; using defaults", slog.Any("error", err))
	}

	// Initialize logging
	slogger, initErr := logging.Init(cfg.Logging, isDev)
	if initErr != nil {
		bootstrap.Warn("logging init had issues", slog.Any("error", initErr))
	}
	settingsService.SetLevelChangeHandler(logging.SetLevel)

	// Create Wails logger adapter
	log := logging.NewLogger(slogger)

	// Build app with settings service
	application := app.NewApp(slogger, settingsService, version)

	log.Info(fmt.Sprintf("--debug-ui: %v", *debugUI))

	windowWidth, windowHeight := application.GetInitialWindowSize()

	// Create application with options
	err = wails.Run(&options.App{
		Title:             "Vervet",
		Width:             windowWidth,
		Height:            windowHeight,
		MinWidth:          1024,
		MinHeight:         768,
		MaxWidth:          8192,
		MaxHeight:         4608,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         true,
		StartHidden:       false,
		HideWindowOnClose: false,
		BackgroundColour:  &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Menu:             nil,
		Logger:           log,
		LogLevel:         logger.DEBUG,
		OnStartup:        application.Startup,
		OnDomReady:       application.DomReady,
		OnBeforeClose:    application.BeforeClose,
		OnShutdown:       application.Shutdown,
		WindowStartState: options.Normal,
		Bind: []any{
			application.ServersProxy,
			application.ConnectionsProxy,
			application.DatabasesProxy,
			application.IndexesProxy,
			application.CollectionsProxy,
			application.ShellProxy,
			application.SystemProxy,
			application.SettingsProxy,
			application.FilesProxy,
			application.WorkspacesProxy,
			application.UpdatesProxy,
			application.ExportProxy,
		},
		EnumBind: []any{
			api.AllOperatingSystems,
		},
		// Linux platform specific options
		Linux: &linux.Options{
			Icon:                icon,
			WindowIsTranslucent: false,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyAlways,
			ProgramName:         "Vervet",
		},
		// Windows platform specific options
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			// DisableFramelessWindowDecorations: false,
			WebviewUserDataPath: "",
			ZoomFactor:          1.0,
			Theme:               windows.SystemDefault,
		},
		// Mac platform specific options
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: false,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "Vervet",
				Message: "",
				Icon:    icon,
			},
		},
		Debug: options.Debug{
			OpenInspectorOnStartup: *debugUI,
		},
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
