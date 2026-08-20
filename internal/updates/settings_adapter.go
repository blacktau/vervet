package updates

import "vervet/internal/models"

// settingsStore is the minimal subset of settings.Service used here.
type settingsStore interface {
	GetSettings() (models.Settings, error)
	SetUpdatesState(lastCheckedAt string, dismissedVersion string) error
}

type SettingsAdapter struct {
	store settingsStore
}

func NewSettingsAdapter(store settingsStore) *SettingsAdapter {
	return &SettingsAdapter{store: store}
}

func (a *SettingsAdapter) GetUpdatesFrequency() string {
	s, err := a.store.GetSettings()
	if err != nil || s.Updates.Frequency == "" {
		return FrequencyDaily
	}
	return s.Updates.Frequency
}

func (a *SettingsAdapter) GetLastCheckedAt() string {
	s, _ := a.store.GetSettings()
	return s.Updates.LastCheckedAt
}

func (a *SettingsAdapter) GetDismissedVersion() string {
	s, _ := a.store.GetSettings()
	return s.Updates.DismissedVersion
}

func (a *SettingsAdapter) SetLastCheckedAt(v string) error {
	s, err := a.store.GetSettings()
	if err != nil {
		return err
	}
	return a.store.SetUpdatesState(v, s.Updates.DismissedVersion)
}

func (a *SettingsAdapter) SetDismissedVersion(v string) error {
	s, err := a.store.GetSettings()
	if err != nil {
		return err
	}
	return a.store.SetUpdatesState(s.Updates.LastCheckedAt, v)
}
