package api

import (
	"context"
	"testing"

	"vervet/internal/models"

	"github.com/stretchr/testify/assert"
)

type MockSettingsProvider struct {
	err         error
	settings    models.Settings
	windowState models.WindowState
}

func (m *MockSettingsProvider) Init(ctx context.Context) error {
	return nil
}

func (m *MockSettingsProvider) GetSettings() (models.Settings, error) {
	if m.err != nil {
		return models.Settings{}, m.err
	}
	return m.settings, nil
}

func (m *MockSettingsProvider) SetSettings(settings *models.Settings) error {
	if m.err != nil {
		return m.err
	}
	m.settings = *settings
	return nil
}

func (m *MockSettingsProvider) RestoreSettings() (*models.Settings, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &m.settings, nil
}

func (m *MockSettingsProvider) GetWindowState() (models.WindowState, error) {
	if m.err != nil {
		return models.WindowState{}, m.err
	}
	return m.windowState, nil
}

func (m *MockSettingsProvider) SaveWindowState(state models.WindowState) error {
	if m.err != nil {
		return m.err
	}
	m.windowState = state
	return nil
}

type MockFontProvider struct {
	fonts []models.Font
}

func (m *MockFontProvider) GetInstalledFonts() []models.Font {
	return m.fonts
}

func TestSettingsProxy_GetSettings(t *testing.T) {
	t.Run("successful get settings", func(t *testing.T) {
		provider := &MockSettingsProvider{
			settings: models.Settings{
				General: models.GeneralSettings{Theme: "dark"},
			},
		}
		fontProvider := &MockFontProvider{}
		proxy := NewSettingsProxy(provider, fontProvider, "")

		result := proxy.GetSettings()

		assert.True(t, result.IsSuccess)
		assert.Equal(t, "dark", result.Data.General.Theme)
	})

	t.Run("get settings error", func(t *testing.T) {
		provider := &MockSettingsProvider{
			err: assert.AnError,
		}
		fontProvider := &MockFontProvider{}
		proxy := NewSettingsProxy(provider, fontProvider, "")

		result := proxy.GetSettings()

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestSettingsProxy_SetSettings(t *testing.T) {
	t.Run("successful set settings", func(t *testing.T) {
		provider := &MockSettingsProvider{}
		fontProvider := &MockFontProvider{}
		proxy := NewSettingsProxy(provider, fontProvider, "")

		result := proxy.SetSettings(models.Settings{
			General: models.GeneralSettings{Theme: "light"},
		})

		assert.True(t, result.IsSuccess)
	})

	t.Run("set settings error", func(t *testing.T) {
		provider := &MockSettingsProvider{
			err: assert.AnError,
		}
		fontProvider := &MockFontProvider{}
		proxy := NewSettingsProxy(provider, fontProvider, "")

		result := proxy.SetSettings(models.Settings{})

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestSettingsProxy_ResetSettings(t *testing.T) {
	t.Run("successful reset settings", func(t *testing.T) {
		provider := &MockSettingsProvider{
			settings: models.Settings{
				General: models.GeneralSettings{Theme: "default"},
			},
		}
		fontProvider := &MockFontProvider{}
		proxy := NewSettingsProxy(provider, fontProvider, "")

		result := proxy.ResetSettings()

		assert.True(t, result.IsSuccess)
		assert.NotNil(t, result.Data)
	})

	t.Run("reset settings error", func(t *testing.T) {
		provider := &MockSettingsProvider{
			err: assert.AnError,
		}
		fontProvider := &MockFontProvider{}
		proxy := NewSettingsProxy(provider, fontProvider, "")

		result := proxy.ResetSettings()

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestSettingsProxy_GetAvailableFonts(t *testing.T) {
	t.Run("get fonts", func(t *testing.T) {
		provider := &MockSettingsProvider{}
		fontProvider := &MockFontProvider{
			fonts: []models.Font{
				{Family: "Arial"},
				{Family: "Courier"},
			},
		}
		proxy := NewSettingsProxy(provider, fontProvider, "")

		result := proxy.GetAvailableFonts()

		assert.True(t, result.IsSuccess)
		assert.Len(t, result.Data, 2)
	})
}

func TestSettingsProxy_GetWindowState(t *testing.T) {
	t.Run("successful get window state", func(t *testing.T) {
		provider := &MockSettingsProvider{
			windowState: models.WindowState{
				X: 100, Y: 100, Width: 1024, Height: 768,
			},
		}
		fontProvider := &MockFontProvider{}
		proxy := NewSettingsProxy(provider, fontProvider, "")

		result := proxy.GetWindowState()

		assert.True(t, result.IsSuccess)
		assert.Equal(t, 1024, result.Data.Width)
	})

	t.Run("get window state error", func(t *testing.T) {
		provider := &MockSettingsProvider{
			err: assert.AnError,
		}
		fontProvider := &MockFontProvider{}
		proxy := NewSettingsProxy(provider, fontProvider, "")

		result := proxy.GetWindowState()

		assert.False(t, result.IsSuccess)
	})
}

func TestSettingsProxy_SaveWindowState(t *testing.T) {
	t.Run("successful save window state", func(t *testing.T) {
		provider := &MockSettingsProvider{}
		fontProvider := &MockFontProvider{}
		proxy := NewSettingsProxy(provider, fontProvider, "")

		result := proxy.SaveWindowState(models.WindowState{
			X: 100, Y: 100, Width: 1024, Height: 768,
		})

		assert.True(t, result.IsSuccess)
	})

	t.Run("save window state error", func(t *testing.T) {
		provider := &MockSettingsProvider{
			err: assert.AnError,
		}
		fontProvider := &MockFontProvider{}
		proxy := NewSettingsProxy(provider, fontProvider, "")

		result := proxy.SaveWindowState(models.WindowState{})

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestSettingsProxy_GetAppVersion(t *testing.T) {
	t.Run("get app version", func(t *testing.T) {
		provider := &MockSettingsProvider{}
		fontProvider := &MockFontProvider{}
		proxy := NewSettingsProxy(provider, fontProvider, "2026.04.2")

		result := proxy.GetAppVersion()

		assert.True(t, result.IsSuccess)
		assert.Equal(t, "2026.04.2", result.Data)
	})
}
