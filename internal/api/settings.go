package api

import (
	"vervet/internal/models"
	"vervet/internal/settings"
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
	cm settings.Manager
	fp FontProvider
}

func NewSettingsProxy(cm settings.Manager, fp FontProvider) *SettingsProxy {
	return &SettingsProxy{
		cm: cm,
		fp: fp,
	}
}

func (cp *SettingsProxy) GetSettings() Result[models.Settings] {
	cfg, err := cp.cm.GetSettings()
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

func (cp *SettingsProxy) SetSettings(cfg models.Settings) EmptyResult {
	err := cp.cm.SetSettings(&cfg)
	if err != nil {
		return EmptyResult{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}
	return Success()
}

func (cp *SettingsProxy) ResetSettings() Result[*models.Settings] {
	cfg, err := cp.cm.RestoreSettings()
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

func (cp *SettingsProxy) GetAvailableFonts() Result[[]models.Font] {
	fonts := cp.fp.GetInstalledFonts()
	return Result[[]models.Font]{
		IsSuccess: true,
		Data:      fonts,
	}
}

func (cp *SettingsProxy) GetWindowState() Result[models.WindowState] {
	state, err := cp.cm.GetWindowState()
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

func (cp *SettingsProxy) SaveWindowState(state models.WindowState) EmptyResult {
	err := cp.cm.SaveWindowState(state)

	if err != nil {
		return Error(err.Error())
	}

	return Success()
}

func (cp *SettingsProxy) GetAppVersion() Result[string] {
	return Result[string]{
		IsSuccess: true,
		Data:      "0.0.1",
	}
}
