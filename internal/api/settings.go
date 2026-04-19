package api

import (
	"log/slog"

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
	log      *slog.Logger
	settings SettingsProvider
	fp       FontProvider
	version  string
}

func NewSettingsProxy(log *slog.Logger, settings SettingsProvider, fp FontProvider, version string) *SettingsProxy {
	return &SettingsProxy{
		log:      log,
		settings: settings,
		fp:       fp,
		version:  version,
	}
}

func (sp *SettingsProxy) GetSettings() Result[models.Settings] {
	cfg, err := sp.settings.GetSettings()
	if err != nil {
		logFail(sp.log, "GetSettings", err)
		return FailResult[models.Settings](err)
	}
	return SuccessResult(cfg)
}

func (sp *SettingsProxy) SetSettings(cfg models.Settings) EmptyResult {
	err := sp.settings.SetSettings(&cfg)
	if err != nil {
		logFail(sp.log, "SetSettings", err)
		return Fail(err)
	}
	return Success()
}

func (sp *SettingsProxy) ResetSettings() Result[*models.Settings] {
	cfg, err := sp.settings.RestoreSettings()
	if err != nil {
		logFail(sp.log, "ResetSettings", err)
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
		logFail(sp.log, "GetWindowState", err)
		return FailResult[models.WindowState](err)
	}

	return SuccessResult(state)
}

func (sp *SettingsProxy) SaveWindowState(state models.WindowState) EmptyResult {
	err := sp.settings.SaveWindowState(state)

	if err != nil {
		logFail(sp.log, "SaveWindowState", err)
		return Fail(err)
	}

	return Success()
}

func (sp *SettingsProxy) GetAppVersion() Result[string] {
	return SuccessResult(sp.version)
}
