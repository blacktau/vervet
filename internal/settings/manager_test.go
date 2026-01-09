package settings_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path"
	"testing"
	"vervet/internal/infrastructure"
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

func newTestManager(store infrastructure.Store, log *slog.Logger) settings.Manager {
	if store == nil {
		store = &storeStub{}
	}
	if log == nil {
		log = slog.Default()
	}
	ctx := context.Background()
	cm := settings.NewTestManager(store, log, ctx)
	return cm
}

func Test_SettingsManager_GetSettings(t *testing.T) {
	t.Run("returns default config if store is empty", func(t *testing.T) {
		m := newTestManager(nil, nil)
		c, err := m.GetSettings()
		if err != nil {
			t.Fatal(err)
		}
		if c.Window.Width != settings.DefaultWindowWidth {
			t.Errorf("expected default window width, got %d", c.Window.Width)
		}
	})

	t.Run("returns error if store has read error", func(t *testing.T) {
		m := newTestManager(&storeStub{
			err: errors.New("read error"),
		}, nil)
		_, err := m.GetSettings()
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("returns config from store", func(t *testing.T) {
		m := newTestManager(&storeStub{
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

func Test_SettingsManager_SetSettings(t *testing.T) {
	t.Run("saves configuration to store", func(t *testing.T) {
		store := &storeStub{}
		m := newTestManager(store, nil)
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
		m := newTestManager(&storeStub{
			err: errors.New("write error"),
		}, nil)
		err := m.SetSettings(&settings.Settings{})
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})
}

func Test_SettingsManager_RestoreSettings(t *testing.T) {
	t.Run("restores default configuration", func(t *testing.T) {
		store := &storeStub{
			content: []byte("window:\n  width: 1024"),
		}
		m := newTestManager(store, nil)
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

func Test_SettingsManager_SaveWindowState(t *testing.T) {
	t.Run("saves window state", func(t *testing.T) {
		store := &storeStub{}
		m := newTestManager(store, nil)
		err := m.SaveWindowState(settings.WindowState{
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
		m := newTestManager(nil, nil)
		err := m.SaveWindowState(settings.WindowState{
			Width: -1,
		})
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})
}

func expectedSettings() settings.Settings {
	return settings.Settings{
		Window: settings.WindowSettings{
			Width:      settings.DefaultWindowWidth,
			Height:     settings.DefaultWindowHeight,
			AsideWidth: settings.DefaultAsideWidth,
		},
		General: settings.GeneralSettings{
			Theme:    "auto",
			Language: "auto",
			Font: settings.FontSettings{
				Size: settings.DefaultFontSize,
			},
		},
		Editor: settings.EditorSettings{
			Font: settings.FontSettings{
				Size: settings.DefaultFontSize,
			},
			LineNumbers: true,
		},
		Terminal: settings.TerminalSettings{
			Font: settings.FontSettings{
				Size: settings.DefaultFontSize,
			},
			CursorStyle: "block",
		},
	}
}
