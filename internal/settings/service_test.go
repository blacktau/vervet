package settings_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path"
	"testing"
	"vervet/internal/infrastructure"
	"vervet/internal/models"
	"vervet/internal/settings"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

type storeStub struct {
	path    string
	err     error
	content []byte
}

func (s *storeStub) Name() string {
	return path.Base(s.path)
}

func (s *storeStub) Path() string {
	return s.path
}

func (s *storeStub) Read() ([]byte, error) {
	return s.content, s.err
}

func (s *storeStub) Save(date []byte) error {
	s.content = date
	return s.err
}

func newTestService(store infrastructure.Store, log *slog.Logger) settings.Service {
	if store == nil {
		store = &storeStub{}
	}
	if log == nil {
		log = slog.Default()
	}
	ctx := context.Background()
	cm := settings.NewTestService(store, log, ctx)
	return cm
}

func Test_SettingsService_GetSettings(t *testing.T) {
	t.Run("returns default config if store is empty", func(t *testing.T) {
		m := newTestService(nil, nil)
		c, err := m.GetSettings()
		if err != nil {
			t.Fatal(err)
		}
		if c.Window.Width != settings.DefaultWindowWidth {
			t.Errorf("expected default window width, got %d", c.Window.Width)
		}
	})

	t.Run("returns error if store has read error", func(t *testing.T) {
		m := newTestService(&storeStub{
			err: errors.New("read error"),
		}, nil)
		_, err := m.GetSettings()
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("returns config from store", func(t *testing.T) {
		m := newTestService(&storeStub{
			content: []byte("window:\n  width: 1024\n  height: 768\n  asideWidth: 200"),
		}, nil)
		c, err := m.GetSettings()
		if err != nil {
			t.Fatal(err)
		}
		if c.Window.Width != 1024 {
			t.Errorf("expected window width 1024, got %d", c.Window.Width)
		}
	})
}

func Test_SettingsService_SetSettings(t *testing.T) {
	t.Run("saves configuration to store", func(t *testing.T) {
		store := &storeStub{}
		m := newTestService(store, nil)
		cfg := expectedSettings()
		cfg.Window.Width = 1280
		err := m.SetSettings(&cfg)
		if err != nil {
			t.Fatal(err)
		}
		if store.content == nil {
			t.Fatal("store content is nil")
		}
		expected, _ := os.ReadFile("testdata/expected_save_configuration.yaml")
		assert.YAMLEq(t, string(expected), string(store.content))
	})

	t.Run("returns error on store write failure", func(t *testing.T) {
		m := newTestService(&storeStub{
			err: errors.New("write error"),
		}, nil)
		err := m.SetSettings(&models.Settings{})
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})
}

func Test_SettingsService_DefaultQueryEngine(t *testing.T) {
	t.Run("default query engine is builtin", func(t *testing.T) {
		m := newTestService(nil, nil)
		c, err := m.GetSettings()
		if err != nil {
			t.Fatal(err)
		}
		if c.Editor.QueryEngine != "builtin" {
			t.Errorf("expected default query engine 'builtin', got '%s'", c.Editor.QueryEngine)
		}
	})
}

func Test_SettingsService_RestoreSettings(t *testing.T) {
	t.Run("restores default configuration", func(t *testing.T) {
		store := &storeStub{
			content: []byte("window:\n  width: 1024"),
		}
		m := newTestService(store, nil)
		_, err := m.RestoreSettings()
		if err != nil {
			t.Fatal(err)
		}
		c, err := m.GetSettings()
		if err != nil {
			t.Fatal(err)
		}
		if c.Window.Width != settings.DefaultWindowWidth {
			t.Errorf("expected default window width, got %d", c.Window.Width)
		}
	})
}

func Test_SettingsService_SaveWindowState(t *testing.T) {
	t.Run("saves window state", func(t *testing.T) {
		store := &storeStub{}
		m := newTestService(store, nil)
		err := m.SaveWindowState(models.WindowState{
			X: 10, Y: 20, Width: 1280, Height: 960,
		})
		if err != nil {
			t.Fatal(err)
		}
		if store.content == nil {
			t.Fatal("store content is nil")
		}
		expected, _ := os.ReadFile("testdata/expected_save_window_state.yaml")
		assert.YAMLEq(t, string(expected), string(store.content))
	})

	t.Run("returns error for invalid window state", func(t *testing.T) {
		m := newTestService(nil, nil)
		err := m.SaveWindowState(models.WindowState{
			Width: -1,
		})
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})
}

func expectedSettings() models.Settings {
	return models.Settings{
		Window: models.WindowSettings{
			Width:      settings.DefaultWindowWidth,
			Height:     settings.DefaultWindowHeight,
			AsideWidth: settings.DefaultAsideWidth,
		},
		General: models.GeneralSettings{
			Theme:    "auto",
			Language: "auto",
			Font: models.FontSettings{
				Size: settings.DefaultFontSize,
			},
		},
		Editor: models.EditorSettings{
			Font: models.FontSettings{
				Size: settings.DefaultFontSize,
			},
			LineNumbers:  true,
			QueryEngine: "builtin",
		},
		Terminal: models.TerminalSettings{
			Font: models.FontSettings{
				Size: settings.DefaultFontSize,
			},
			CursorStyle: "block",
		},
	}
}
