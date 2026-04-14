package updates

import "vervet/internal/models"

// settingsStore is the minimal subset of settings.Service used here.
type settingsStore interface {
	GetSettings() (models.Settings, error)
	SetSettings(settings *models.Settings) error
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
	s.Updates.LastCheckedAt = v
	return a.store.SetSettings(&s)
}

func (a *SettingsAdapter) SetDismissedVersion(v string) error {
	s, err := a.store.GetSettings()
	if err != nil {
		return err
	}
	s.Updates.DismissedVersion = v
	return a.store.SetSettings(&s)
}
