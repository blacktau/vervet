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
	"vervet/internal/models"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gopkg.in/yaml.v3"
)

type Service interface {
	Init(ctx context.Context) error
	GetSettings() (models.Settings, error)
	SetSettings(settings *models.Settings) error
	RestoreSettings() (*models.Settings, error)
	GetWindowState() (models.WindowState, error)
	SaveWindowState(state models.WindowState) error
	SetLevelChangeHandler(cb func(slog.Level))
}

const DefaultFontSize = 14
const DefaultWindowWidth = 1024
const DefaultWindowHeight = 768
const DefaultAsideWidth = 300

type settingsService struct {
	store         infrastructure.Store
	log           *slog.Logger
	ctx           context.Context
	mutex         sync.Mutex
	isDev         bool
	onLevelChange func(slog.Level)
}

func NewService(log *slog.Logger, isDev bool) *settingsService {
	log = log.With(slog.String(logging.SourceKey, "SettingsService"))
	store, err := infrastructure.NewStore("configuration.yaml", log)
	if err != nil {
		log.Error("error accessing configuration", slog.Any("error", err))
		panic(fmt.Errorf("error accessing configuration: %v", err))
	}

	return &settingsService{
		store: store,
		log:   log,
		isDev: isDev,
	}
}

func (s *settingsService) Init(ctx context.Context) error {
	s.ctx = ctx
	return nil
}

func (s *settingsService) SetLevelChangeHandler(cb func(slog.Level)) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.onLevelChange = cb
}

func (s *settingsService) GetSettings() (models.Settings, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	settings, err := s.getSettings()
	if err != nil {
		return s.defaultSettings(), fmt.Errorf("error getting settings: %w", err)
	}

	settings.Window.Width = max(settings.Window.Width, DefaultWindowWidth)
	settings.Window.Height = max(settings.Window.Height, DefaultWindowHeight)
	settings.Window.AsideWidth = max(settings.Window.AsideWidth, DefaultAsideWidth)

	return settings, nil
}

func (s *settingsService) SetSettings(settings *models.Settings) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	prev, _ := s.getSettings()
	if err := s.saveSettings(settings); err != nil {
		return err
	}
	if s.onLevelChange != nil && prev.Logging.Level != settings.Logging.Level {
		s.onLevelChange(logging.ParseLevel(settings.Logging.Level))
	}
	return nil
}

func (s *settingsService) RestoreSettings() (*models.Settings, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	settings := s.defaultSettings()
	err := s.saveSettings(&settings)
	if err != nil {
		return nil, fmt.Errorf("error resetting configuration: %w", err)
	}

	return &settings, nil
}

func (s *settingsService) GetWindowState() (models.WindowState, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	settings, err := s.getSettings()
	if err != nil {
		return models.WindowState{}, fmt.Errorf("failed to get configuration for window state: %w", err)
	}

	width := max(settings.Window.Width, DefaultWindowWidth)
	height := max(settings.Window.Height, DefaultWindowHeight)
	x, y := settings.Window.PositionX, settings.Window.PositionY
	screenWidth, screenHeight := s.getScreenSize()

	if x <= 0 || x+width > screenWidth || y <= 0 || y+height > screenHeight {
		x, y = (screenWidth-width)/2, (screenHeight-height)/2
	}

	return models.WindowState{X: x, Y: y, Width: width, Height: height}, nil
}

func (s *settingsService) SaveWindowState(state models.WindowState) error {
	if state.Width <= 0 || state.Height <= 0 || state.X < 0 || state.Y < 0 {
		return fmt.Errorf("invalid window state: %+v", state)
	}

	err := s.update(map[string]any{
		"window.positionX": state.X,
		"window.positionY": state.Y,
		"window.width":     state.Width,
		"window.height":    state.Height,
	})

	if err != nil {
		return fmt.Errorf("failed to save window state: %w", err)
	}

	return nil
}

func (s *settingsService) update(values map[string]any) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	settings, err := s.getSettings()
	if err != nil {
		return fmt.Errorf("error getting configuration for update: %w", err)
	}

	for path, v := range values {
		if err = s.setSettings(&settings, path, v); err != nil {
			return fmt.Errorf("error updating '%s' configuration: %w", path, err)
		}
	}

	return s.saveSettings(&settings)
}

func (s *settingsService) getScreenSize() (width, height int) {
	if screens, err := runtime.ScreenGetAll(s.ctx); err == nil {
		for _, screen := range screens {
			if screen.IsCurrent {
				return screen.Size.Width, screen.Size.Height
			}
		}
	}

	return DefaultWindowWidth, DefaultWindowHeight
}

func (s *settingsService) getSettings() (models.Settings, error) {
	settings := s.defaultSettings()
	b, err := s.store.Read()
	if err != nil && !os.IsNotExist(err) {
		return settings, fmt.Errorf("error reading configuration: %w", err)
	}

	if len(b) <= 0 {
		if saveErr := s.saveSettings(&settings); saveErr != nil {
			s.log.Warn("failed to persist first-run defaults", slog.Any("error", saveErr))
		}
		return settings, nil
	}

	if err = yaml.Unmarshal(b, &settings); err != nil {
		return s.defaultSettings(), fmt.Errorf("error parsing configuration: %w", err)
	}

	settings.Logging.Normalize()
	return settings, nil
}

func (s *settingsService) setSettings(settings *models.Settings, key string, value any) error {
	parts := strings.Split(key, ".")

	if len(parts) == 0 {
		return fmt.Errorf("invalid configuration key: %s", key)
	}

	refValue := reflect.ValueOf(settings).Elem()

	for idx, part := range parts {
		part = strings.ToUpper(part[:1]) + part[1:]
		field := refValue.FieldByName(part)

		if !field.IsValid() {
			return fmt.Errorf("invalid configuration key: %s (field %s not found)", key, part)
		}

		if idx == len(parts)-1 {
			if !field.CanSet() {
				return fmt.Errorf("invalid configuration key: %s (field %s is not settable)", key, part)
			}

			val := reflect.ValueOf(value)
			if val.Type().ConvertibleTo(field.Type()) {
				field.Set(val.Convert(field.Type()))
				return nil
			}

			return fmt.Errorf("invalid configuration value: %v (expected type %s)", value, field.Type())
		}

		if field.Kind() == reflect.Struct {
			refValue = field
		} else if field.Kind() == reflect.Ptr && !field.IsNil() && field.Elem().Kind() == reflect.Struct {
			refValue = field.Elem()
		} else {
			return fmt.Errorf("invalid configuration path: %s", key)
		}
	}

	return fmt.Errorf("invalid configuration key: %s", key)
}

func (s *settingsService) saveSettings(settings *models.Settings) error {
	b, err := yaml.Marshal(settings)
	if err != nil {
		return fmt.Errorf("error marshalling configuration: %w", err)
	}

	if err = s.store.Save(b); err != nil {
		return fmt.Errorf("error saving configuration: %w", err)
	}

	return nil
}

func (s *settingsService) defaultSettings() models.Settings {
	return defaultSettingsForBuild(s.isDev)
}

func defaultSettingsForBuild(isDev bool) models.Settings {
	return models.Settings{
		Window: models.WindowSettings{
			Width:      DefaultWindowWidth,
			Height:     DefaultWindowHeight,
			AsideWidth: DefaultAsideWidth,
		},
		General: models.GeneralSettings{
			Theme:    "auto",
			Language: "auto",
			Font: models.FontSettings{
				Size: DefaultFontSize,
			},
		},
		Editor: models.EditorSettings{
			Font: models.FontSettings{
				Size: DefaultFontSize,
			},
			LineNumbers: true,
			QueryEngine: "builtin",
		},
		Terminal: models.TerminalSettings{
			Font: models.FontSettings{
				Size: DefaultFontSize,
			},
			CursorStyle: "block",
		},
		Updates: models.UpdatesSettings{
			Frequency: "daily",
		},
		Logging: models.LoggingSettings{
			Level:          "info",
			ConsoleEnabled: isDev,
			FileEnabled:    true,
			MaxSizeMB:      10,
			MaxBackups:     5,
		},
	}
}
