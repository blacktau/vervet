package api

import (
	"vervet/internal/models"
)

type FontProvider interface {
	GetInstalledFonts() []models.Font
}

type SettingsProvider interface {
	GetSettings() (models.Settings, error)
	SetSettings(settings *models.Settings) error
	RestoreSettings() (*models.Settings, error)
	GetWindowState() (models.WindowState, error)
	SaveWindowState(state models.WindowState) error
}

type SettingsProxy struct {
	settings SettingsProvider
	fp       FontProvider
}

func NewSettingsProxy(settings SettingsProvider, fp FontProvider) *SettingsProxy {
	return &SettingsProxy{
		settings: settings,
		fp:       fp,
	}
}

func (sp *SettingsProxy) GetSettings() Result[models.Settings] {
	cfg, err := sp.settings.GetSettings()
	if err != nil {
		return Result[models.Settings]{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}
	return Result[models.Settings]{
		IsSuccess: true,
		Data:      cfg,
	}
}

func (sp *SettingsProxy) SetSettings(cfg models.Settings) EmptyResult {
	err := sp.settings.SetSettings(&cfg)
	if err != nil {
		return EmptyResult{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}
	return Success()
}

func (sp *SettingsProxy) ResetSettings() Result[*models.Settings] {
	cfg, err := sp.settings.RestoreSettings()
	if err != nil {
		return Result[*models.Settings]{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}

	return Result[*models.Settings]{
		IsSuccess: true,
		Data:      cfg,
	}
}

func (sp *SettingsProxy) GetAvailableFonts() Result[[]models.Font] {
	fonts := sp.fp.GetInstalledFonts()
	return Result[[]models.Font]{
		IsSuccess: true,
		Data:      fonts,
	}
}

func (sp *SettingsProxy) GetWindowState() Result[models.WindowState] {
	state, err := sp.settings.GetWindowState()
	if err != nil {
		return Result[models.WindowState]{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}

	return Result[models.WindowState]{
		IsSuccess: true,
		Data:      state,
	}
}

func (sp *SettingsProxy) SaveWindowState(state models.WindowState) EmptyResult {
	err := sp.settings.SaveWindowState(state)

	if err != nil {
		return Error(err.Error())
	}

	return Success()
}

func (sp *SettingsProxy) GetAppVersion() Result[string] {
	return Result[string]{
		IsSuccess: true,
		Data:      "0.0.1",
	}
}
