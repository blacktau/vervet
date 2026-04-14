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

var version string

func main() {
	debugUI := flag.Bool("debug-ui", false, "enable ui inspector")
	logLevel := &slog.LevelVar{}
	logLevel.Set(slog.LevelDebug)
	slogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	log := logging.NewLogger(slogger)

	application := app.NewApp(slogger, version)

	log.Info(fmt.Sprintf("--debug-ui: %v", *debugUI))

	windowWidth, windowHeight := application.GetInitialWindowSize()

	// Create application with options
	err := wails.Run(&options.App{
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
