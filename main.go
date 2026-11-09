package main

import (
	"embed"
	"flag"
	"log/slog"
	"os"

	"vervet/internal/app"
	"vervet/internal/logging"
	"vervet/internal/settings"

	"github.com/wailsapp/wails/v3/pkg/application"
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

	vervet := app.NewApp(slogger, settingsService, version)

	slogger.Info("starting", slog.Bool("debugUI", *debugUI))

	windowWidth, windowHeight := vervet.GetInitialWindowSize()

	wailsApp := application.New(application.Options{
		Name:        "Vervet",
		Description: "A desktop MongoDB explorer",
		Icon:        icon,
		Services:    vervet.Services(),
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Logger:     slogger,
		LogLevel:   slog.LevelDebug,
		ShouldQuit: vervet.ShouldQuit,
		Linux: application.LinuxOptions{
			ProgramName:   "Vervet",
			ApplicationID: "com.blacktau.vervet",
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:                   "main",
		Title:                  "Vervet",
		Width:                  windowWidth,
		Height:                 windowHeight,
		MinWidth:               1024,
		MinHeight:              768,
		MaxWidth:               8192,
		MaxHeight:              4608,
		Frameless:              true,
		InitialPosition:        application.WindowCentered,
		BackgroundColour:       application.NewRGB(255, 255, 255),
		URL:                    "/",
		DevToolsEnabled:        isDev || *debugUI,
		OpenInspectorOnStartup: *debugUI,
		Linux: application.LinuxWindow{
			Icon:             icon,
			WebviewGpuPolicy: application.WebviewGpuPolicyAlways,
		},
		Windows: application.WindowsWindow{
			Theme: application.SystemDefault,
		},
		Mac: application.MacWindow{
			Appearance: application.NSAppearanceNameDarkAqua,
			Backdrop:   application.MacBackdropTranslucent,
		},
	})

	vervet.Attach(wailsApp, window)

	if err := wailsApp.Run(); err != nil {
		slogger.Error("application exited with an error", slog.Any("error", err))
		os.Exit(1)
	}
}
