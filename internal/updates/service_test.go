package updates

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

type fakeSettings struct {
	frequency        string
	lastCheckedAt    string
	dismissedVersion string
	saved            map[string]any
}

func (f *fakeSettings) GetUpdatesFrequency() string { return f.frequency }
func (f *fakeSettings) GetLastCheckedAt() string    { return f.lastCheckedAt }
func (f *fakeSettings) GetDismissedVersion() string { return f.dismissedVersion }
func (f *fakeSettings) SetLastCheckedAt(v string) error {
	f.lastCheckedAt = v
	return nil
}
func (f *fakeSettings) SetDismissedVersion(v string) error {
	f.dismissedVersion = v
	return nil
}

type fakeEmitter struct{ events []emitted }
type emitted struct {
	name string
	data any
}

func (f *fakeEmitter) EmitEvent(name string, data any) {
	f.events = append(f.events, emitted{name, data})
}

func newTestServer(t *testing.T, tag, htmlURL, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": tag,
			"html_url": htmlURL,
			"body":     body,
		})
	}))
}

func newService(t *testing.T, serverURL string, settings *fakeSettings, emitter *fakeEmitter, currentVersion string) *Service {
	t.Helper()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	return NewService(log, Config{
		CurrentVersion: currentVersion,
		ReleasesURL:    serverURL,
		HTTPClient:     &http.Client{Timeout: 2 * time.Second},
		Settings:       settings,
		Emitter:        emitter,
		Now:            func() time.Time { return time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC) },
	})
}

func TestCheckNow_ReturnsUpdateWhenNewer(t *testing.T) {
	ts := newTestServer(t, "v2026.05.1", "https://github.com/blacktau/vervet/releases/v2026.05.1", "notes")
	defer ts.Close()
	s := newService(t, ts.URL, &fakeSettings{frequency: "daily"}, &fakeEmitter{}, "2026.04.4")
	info, err := s.CheckNow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available || info.Version != "2026.05.1" {
		t.Fatalf("unexpected: %+v", info)
	}
}

func TestCheckNow_NoUpdateWhenSameVersion(t *testing.T) {
	ts := newTestServer(t, "v2026.04.4", "https://x", "")
	defer ts.Close()
	s := newService(t, ts.URL, &fakeSettings{frequency: "daily"}, &fakeEmitter{}, "2026.04.4")
	info, _ := s.CheckNow(context.Background())
	if info.Available {
		t.Fatalf("should not be available: %+v", info)
	}
}

func TestCheckIfDue_SkipsWhenFrequencyNever(t *testing.T) {
	emitter := &fakeEmitter{}
	s := newService(t, "http://unreachable.invalid", &fakeSettings{frequency: "never"}, emitter, "2026.04.4")
	if err := s.CheckIfDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(emitter.events) != 0 {
		t.Fatalf("expected no events, got %d", len(emitter.events))
	}
}

func TestCheckIfDue_SkipsWhenNotDue(t *testing.T) {
	emitter := &fakeEmitter{}
	recentlyChecked := time.Date(2026, 4, 13, 11, 30, 0, 0, time.UTC).Format(time.RFC3339)
	s := newService(t, "http://unreachable.invalid", &fakeSettings{frequency: "daily", lastCheckedAt: recentlyChecked}, emitter, "2026.04.4")
	if err := s.CheckIfDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(emitter.events) != 0 {
		t.Fatalf("expected no events")
	}
}

func TestCheckIfDue_EmitsEventWhenUpdateAvailable(t *testing.T) {
	ts := newTestServer(t, "v2026.05.1", "https://x", "notes")
	defer ts.Close()
	emitter := &fakeEmitter{}
	s := newService(t, ts.URL, &fakeSettings{frequency: "daily"}, emitter, "2026.04.4")
	if err := s.CheckIfDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(emitter.events) != 1 || emitter.events[0].name != "update-available" {
		t.Fatalf("unexpected events: %+v", emitter.events)
	}
}

func TestCheckIfDue_DismissedSuppressesEvent(t *testing.T) {
	ts := newTestServer(t, "v2026.05.1", "https://x", "")
	defer ts.Close()
	emitter := &fakeEmitter{}
	s := newService(t, ts.URL, &fakeSettings{frequency: "daily", dismissedVersion: "2026.05.1"}, emitter, "2026.04.4")
	_ = s.CheckIfDue(context.Background())
	if len(emitter.events) != 0 {
		t.Fatalf("dismissed version should suppress event")
	}
}

func TestCheckIfDue_SkipsWhenCurrentVersionIsDev(t *testing.T) {
	emitter := &fakeEmitter{}
	s := newService(t, "http://unreachable.invalid", &fakeSettings{frequency: "daily"}, emitter, "dev")
	if err := s.CheckIfDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(emitter.events) != 0 {
		t.Fatalf("dev version should skip check")
	}
}

func TestCheckIfDue_UpdatesLastCheckedOnError(t *testing.T) {
	settings := &fakeSettings{frequency: "daily"}
	s := newService(t, "http://127.0.0.1:1/unreachable", settings, &fakeEmitter{}, "2026.04.4")
	_ = s.CheckIfDue(context.Background())
	if settings.lastCheckedAt == "" {
		t.Fatalf("lastCheckedAt should be updated even on error")
	}
}
