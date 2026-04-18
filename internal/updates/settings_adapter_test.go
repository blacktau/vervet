package updates

import (
	"testing"
	"vervet/internal/models"
)

type fakeSettingsService struct {
	settings models.Settings
	saved    *models.Settings
}

func (f *fakeSettingsService) GetSettings() (models.Settings, error) { return f.settings, nil }
func (f *fakeSettingsService) SetSettings(s *models.Settings) error {
	f.saved = s
	f.settings = *s
	return nil
}

func TestSettingsAdapter_RoundTrip(t *testing.T) {
	svc := &fakeSettingsService{settings: models.Settings{Updates: models.UpdatesSettings{Frequency: "weekly"}}}
	a := NewSettingsAdapter(svc)
	if a.GetUpdatesFrequency() != "weekly" {
		t.Fatalf("expected weekly")
	}
	if err := a.SetLastCheckedAt("2026-04-13T12:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if a.GetLastCheckedAt() != "2026-04-13T12:00:00Z" {
		t.Fatalf("not persisted")
	}
	if err := a.SetDismissedVersion("2026.05.1"); err != nil {
		t.Fatal(err)
	}
	if a.GetDismissedVersion() != "2026.05.1" {
		t.Fatalf("not persisted")
	}
}
