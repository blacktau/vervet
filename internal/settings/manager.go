package settings

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"sync"
	"vervet/internal/infrastructure"
	"vervet/internal/logging"

	"github.com/adrg/sysfont"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gopkg.in/yaml.v3"
)

type Manager interface {
	Init(ctx context.Context) error
	GetSettings() (Settings, error)
	SetSettings(settings *Settings) error
	RestoreSettings() (*Settings, error)
	GetFonts() ([]Font, error)
	GetWindowState() (WindowState, error)
	SaveWindowState(state WindowState) error
}

type settingsManager struct {
	store infrastructure.Store
	log   *slog.Logger
	ctx   context.Context
	mutex sync.Mutex
}

func NewManager(log *slog.Logger) Manager {
	log = log.With(slog.String(logging.SourceKey, "SettingsManager"))
	store, err := infrastructure.NewStore("configuration.yaml", log)
	if err != nil {
		log.Error("error accessing configuration", slog.Any("error", err))
		panic(fmt.Errorf("error accessing configuration: %v", err))
	}

	return &settingsManager{
		store: store,
		log:   log,
	}
}

func (cm *settingsManager) Init(ctx context.Context) error {
	cm.log.Debug("Initializing Settings Manager")
	cm.ctx = ctx
	return nil
}

func (cm *settingsManager) GetSettings() (Settings, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.log.Info("Loading settings...")

	settings, err := cm.getSettings()
	if err != nil {
		cm.log.Error("error getting settings", slog.Any("error", err))
		return defaultSettings(), fmt.Errorf("error getting settings: %v", err)
	}

	settings.Window.Width = max(settings.Window.Width, DefaultWindowWidth)
	settings.Window.Height = max(settings.Window.Height, DefaultWindowHeight)
	settings.Window.AsideWidth = max(settings.Window.AsideWidth, DefaultAsideWidth)

	cm.log.Debug("Settings loaded", slog.Any("settings", settings))

	return settings, nil
}

func (cm *settingsManager) SetSettings(settings *Settings) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.log.Debug("Saving settings", slog.Any("settings", settings))

	return cm.saveSettings(settings)
}

func (cm *settingsManager) RestoreSettings() (*Settings, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.log.Info("Resetting configuration...")

	settings := defaultSettings()
	err := cm.saveSettings(&settings)
	if err != nil {
		cm.log.Error("error resetting configuration", slog.Any("error", err))
		return nil, fmt.Errorf("error resetting configuration: %v", err)
	}

	return &settings, nil
}

func (cm *settingsManager) GetFonts() ([]Font, error) {
	cm.log.Debug("Getting system font list")
	finder := sysfont.NewFinder(nil)
	fontSet := map[string]Font{}
	for _, f := range finder.List() {
		if len(f.Family) <= 0 {
			continue
		}

		if _, ok := fontSet[f.Family]; ok {
			continue
		}

		fontSet[f.Family] = Font{
			Name: f.Family,
			Path: f.Filename,
		}
	}

	fonts := make([]Font, 0, len(fontSet))
	for _, f := range fontSet {
		fonts = append(fonts, f)
	}

	return fonts, nil
}

func (cm *settingsManager) GetWindowState() (WindowState, error) {
	cm.log.Debug("Getting window state")
	settings, err := cm.GetSettings()
	if err != nil {
		cm.log.Error("failed to get configuration for window state", slog.Any("error", err))
		return WindowState{}, fmt.Errorf("failed to get configuration for window state: %w", err)
	}

	x, y, width, height := settings.Window.PositionX, settings.Window.PositionY, settings.Window.Width, settings.Window.Height
	screenWidth, screenHeight := cm.getScreenSize()

	if x <= 0 || x+width > screenWidth || y <= 0 || y+height > screenHeight {
		x, y = (screenWidth-width)/2, (screenHeight-height)/2
	}

	return WindowState{x, y, width, height}, nil
}

func (cm *settingsManager) SaveWindowState(state WindowState) error {
	log := cm.log.With(slog.Any("windowState", state))
	log.Info("Saving window state")

	if state.Width <= 0 || state.Height <= 0 || state.X < 0 || state.Y < 0 {
		cm.log.Error("invalid window state", slog.Any("windowState", state))
		return fmt.Errorf("invalid window state: %+v", state)
	}

	err := cm.update(map[string]any{
		"window.positionX": state.X,
		"window.positionY": state.Y,
		"window.width":     state.Width,
		"window.height":    state.Height,
	})

	if err != nil {
		log.Error("failed to save window state: %v", slog.Any("error", err))
		return fmt.Errorf("failed to save window state: %w", err)
	}

	return nil
}

func (cm *settingsManager) update(values map[string]any) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	log := cm.log.With(slog.Any("values", values))

	settings, err := cm.getSettings()
	if err != nil {
		log.Error("error getting configuration for update", slog.Any("error", err))
		return fmt.Errorf("error getting configuration for update: %v", err)
	}

	for path, v := range values {
		if err = cm.setSettings(&settings, path, v); err != nil {
			log.Error("error updating configuration", slog.String("path", path), slog.Any("error", err))
			return fmt.Errorf("error updating '%s' configuration: %v", path, err)
		}
	}

	return cm.saveSettings(&settings)
}

func (cm *settingsManager) getScreenSize() (width, height int) {
	if screens, err := runtime.ScreenGetAll(cm.ctx); err == nil {
		for _, screen := range screens {
			if screen.IsCurrent {
				return screen.Size.Width, screen.Size.Height
			}
		}
	}

	return DefaultWindowWidth, DefaultWindowHeight
}

func (cm *settingsManager) getSettings() (Settings, error) {
	settings := defaultSettings()
	b, err := cm.store.Read()
	if err != nil && !os.IsNotExist(err) {
		slog.Error("error reading configuration", slog.Any("error", err))
		return settings, fmt.Errorf("error reading configuration: %v", err)
	}

	if len(b) <= 0 {
		cm.log.Info("No configuration found, using defaults.")
		return settings, nil
	}

	if err = yaml.Unmarshal(b, &settings); err != nil {
		cm.log.Error("error parsing configuration", slog.Any("error", err))
		return defaultSettings(), fmt.Errorf("error parsing configuration: %v", err)
	}

	return settings, nil
}

func (cm *settingsManager) setSettings(settings *Settings, key string, value any) error {
	parts := strings.Split(key, ".")

	log := cm.log.With(slog.String("key", key), slog.Any("value", value))
	log.Debug("Setting configuration value")

	if len(parts) == 0 {

		log.Error("invalid configuration key")
		return fmt.Errorf("invalid configuration key: %s", key)
	}

	refValue := reflect.ValueOf(settings).Elem()

	for idx, part := range parts {
		part = strings.ToUpper(part[:1]) + part[1:]
		field := refValue.FieldByName(part)

		if !field.IsValid() {
			log.Error("invalid configuration key: field not found", slog.String("field", part))
			return fmt.Errorf("invalid configuration key: %s (field %s not found)", key, part)
		}

		if idx == len(parts)-1 {
			if !field.CanSet() {
				log.Error(fmt.Sprintf("invalid configuration key: %s (field %s is not settable)", key, part))
				return fmt.Errorf("invalid configuration key: %s (field %s is not settable)", key, part)
			}

			val := reflect.ValueOf(value)
			if val.Type().ConvertibleTo(field.Type()) {
				field.Set(val.Convert(field.Type()))
				return nil
			}

			log.Error("invalid configuration value: expected different type", slog.Any("expectedType", field.Type()))
			return fmt.Errorf("invalid configuration value: %v (expected type %s)", value, field.Type())
		}

		if field.Kind() == reflect.Struct {
			refValue = field
		} else if field.Kind() == reflect.Ptr && !field.IsNil() && field.Elem().Kind() == reflect.Struct {
			refValue = field.Elem()
		} else {
			log.Error("invalid configuration path", slog.String("path", key))
			return fmt.Errorf("invalid configuration path: %s", key)
		}
	}

	log.Error("invalid configuration key")
	return fmt.Errorf("invalid configuration key: %s", key)
}

func (cm *settingsManager) saveSettings(settings *Settings) error {
	b, err := yaml.Marshal(settings)
	if err != nil {
		cm.log.Error("error marshalling configuration", slog.Any("error", err))
		return fmt.Errorf("error marshalling configuration: %v", err)
	}

	if err = cm.store.Save(b); err != nil {
		cm.log.Error("error saving configuration", slog.Any("error", err))
		return fmt.Errorf("error saving configuration: %v", err)
	}

	return nil
}

func defaultSettings() Settings {
	return Settings{
		Window: WindowSettings{
			Width:      DefaultWindowWidth,
			Height:     DefaultWindowHeight,
			AsideWidth: DefaultAsideWidth,
		},
		General: GeneralSettings{
			Theme:    "auto",
			Language: "auto",
			Font: FontSettings{
				Size: DefaultFontSize,
			},
		},
		Editor: EditorSettings{
			Font: FontSettings{
				Size: DefaultFontSize,
			},
			LineNumbers: true,
		},
		Terminal: TerminalSettings{
			Font: FontSettings{
				Size: DefaultFontSize,
			},
			CursorStyle: "block",
		},
	}
}
