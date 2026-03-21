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
		return FailResult[models.Settings](err)
	}
	return SuccessResult(cfg)
}

func (sp *SettingsProxy) SetSettings(cfg models.Settings) EmptyResult {
	err := sp.settings.SetSettings(&cfg)
	if err != nil {
		return Fail(err)
	}
	return Success()
}

func (sp *SettingsProxy) ResetSettings() Result[*models.Settings] {
	cfg, err := sp.settings.RestoreSettings()
	if err != nil {
		return FailResult[*models.Settings](err)
	}

	return SuccessResult(cfg)
}

func (sp *SettingsProxy) GetAvailableFonts() Result[[]models.Font] {
	fonts := sp.fp.GetInstalledFonts()
	return SuccessResult(fonts)
}

func (sp *SettingsProxy) GetWindowState() Result[models.WindowState] {
	state, err := sp.settings.GetWindowState()
	if err != nil {
		return FailResult[models.WindowState](err)
	}

	return SuccessResult(state)
}

func (sp *SettingsProxy) SaveWindowState(state models.WindowState) EmptyResult {
	err := sp.settings.SaveWindowState(state)

	if err != nil {
		return Fail(err)
	}

	return Success()
}

func (sp *SettingsProxy) GetAppVersion() Result[string] {
	return SuccessResult("0.0.1")
}
