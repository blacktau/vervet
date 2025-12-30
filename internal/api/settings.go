package api

import "vervet/internal/settings"

type SettingsProxy struct {
	cm settings.Manager
}

func NewSettingsProxy(cm settings.Manager) *SettingsProxy {
	return &SettingsProxy{
		cm: cm,
	}
}

func (cp *SettingsProxy) GetSettings() Result[settings.Settings] {
	cfg, err := cp.cm.GetSettings()
	if err != nil {
		return Result[settings.Settings]{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}
	return Result[settings.Settings]{
		IsSuccess: true,
		Data:      cfg,
	}
}

func (cp *SettingsProxy) SetSettings(cfg settings.Settings) EmptyResult {
	err := cp.cm.SetSettings(&cfg)
	return EmptyResult{
		IsSuccess: err == nil,
		Error:     err.Error(),
	}
}

func (cp *SettingsProxy) ResetSettings() Result[*settings.Settings] {
	cfg, err := cp.cm.RestoreSettings()
	return Result[*settings.Settings]{
		IsSuccess: err == nil,
		Data:      cfg,
		Error:     err.Error(),
	}
}

func (cp *SettingsProxy) GetAvailableFonts() Result[[]settings.Font] {
	fonts, err := cp.cm.GetFonts()
	return Result[[]settings.Font]{
		IsSuccess: err == nil,
		Data:      fonts,
		Error:     err.Error(),
	}
}

func (cp *SettingsProxy) GetWindowState() Result[settings.WindowState] {
	state, err := cp.cm.GetWindowState()
	return Result[settings.WindowState]{
		IsSuccess: err == nil,
		Data:      state,
		Error:     err.Error(),
	}
}

func (cp *SettingsProxy) SaveWindowState(state settings.WindowState) EmptyResult {
	err := cp.cm.SaveWindowState(state)

	return EmptyResult{
		IsSuccess: err == nil,
		Error:     err.Error(),
	}
}

func (cp *SettingsProxy) GetAppVersion() Result[string] {
	return Result[string]{
		IsSuccess: true,
		Data:      "0.0.1",
	}
}
