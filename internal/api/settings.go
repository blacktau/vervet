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
	if err != nil {
		return EmptyResult{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}
	return Success()
}

func (cp *SettingsProxy) ResetSettings() Result[*settings.Settings] {
	cfg, err := cp.cm.RestoreSettings()
	if err != nil {
		return Result[*settings.Settings]{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}

	return Result[*settings.Settings]{
		IsSuccess: true,
		Data:      cfg,
	}
}

func (cp *SettingsProxy) GetAvailableFonts() Result[[]settings.Font] {
	fonts, err := cp.cm.GetFonts()
	if err != nil {
		return Result[[]settings.Font]{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}
	return Result[[]settings.Font]{
		IsSuccess: true,
		Data:      fonts,
	}
}

func (cp *SettingsProxy) GetWindowState() Result[settings.WindowState] {
	state, err := cp.cm.GetWindowState()
	if err != nil {
		return Result[settings.WindowState]{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}

	return Result[settings.WindowState]{
		IsSuccess: true,
		Data:      state,
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
