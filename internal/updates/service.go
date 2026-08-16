package updates

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const (
	FrequencyNever   = "never"
	FrequencyStartup = "startup"
	FrequencyDaily   = "daily"
	FrequencyWeekly  = "weekly"

	EventUpdateAvailable = "update-available"

	defaultReleasesURL = "https://api.github.com/repos/blacktau/vervet/releases/latest"
)

// UpdateInfo is returned to callers and emitted as the event payload.
type UpdateInfo struct {
	Available    bool   `json:"available"`
	Version      string `json:"version"`
	URL          string `json:"url"`
	ReleaseNotes string `json:"releaseNotes"`
}

// SettingsAccessor is the narrow settings surface the service needs.
type SettingsAccessor interface {
	GetUpdatesFrequency() string
	GetLastCheckedAt() string
	GetDismissedVersion() string
	SetLastCheckedAt(v string) error
	SetDismissedVersion(v string) error
}

// EventEmitter matches wailsRuntime.EventsEmit's shape for testability.
type EventEmitter interface {
	EmitEvent(name string, data any)
}

// Config carries all dependencies so tests can substitute fakes.
type Config struct {
	CurrentVersion string
	ReleasesURL    string
	HTTPClient     *http.Client
	Settings       SettingsAccessor
	Emitter        EventEmitter
	Now            func() time.Time
}

type Service struct {
	log            *slog.Logger
	currentVersion string
	releasesURL    string
	http           *http.Client
	settings       SettingsAccessor
	emitter        EventEmitter
	now            func() time.Time
}

func NewService(log *slog.Logger, cfg Config) *Service {
	if cfg.ReleasesURL == "" {
		cfg.ReleasesURL = defaultReleasesURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{
		log:            log,
		currentVersion: cfg.CurrentVersion,
		releasesURL:    cfg.ReleasesURL,
		http:           cfg.HTTPClient,
		settings:       cfg.Settings,
		emitter:        cfg.Emitter,
		now:            cfg.Now,
	}
}

// CheckNow fetches the latest release and returns an UpdateInfo.
// It always updates lastCheckedAt, even on error.
func (s *Service) CheckNow(ctx context.Context) (UpdateInfo, error) {
	s.writeLastChecked()
	latest, url, body, err := s.fetchLatest(ctx)
	if err != nil {
		return UpdateInfo{}, err
	}
	if !IsReleaseVersion(s.currentVersion) {
		return UpdateInfo{Available: false, Version: latest, URL: url, ReleaseNotes: body}, nil
	}
	if CompareVersion(latest, s.currentVersion) <= 0 {
		return UpdateInfo{Available: false, Version: latest, URL: url, ReleaseNotes: body}, nil
	}
	return UpdateInfo{Available: true, Version: normalizeVersion(latest), URL: url, ReleaseNotes: body}, nil
}

// CheckIfDue runs CheckNow only if frequency and interval allow it.
// Emits "update-available" on discovery of a non-dismissed newer release.
func (s *Service) CheckIfDue(ctx context.Context) error {
	freq := s.settings.GetUpdatesFrequency()
	if freq == FrequencyNever {
		return nil
	}
	if !IsReleaseVersion(s.currentVersion) {
		return nil
	}
	if !s.isDue(freq) {
		return nil
	}
	info, err := s.CheckNow(ctx)
	if err != nil {
		s.log.Warn("update check failed", slog.Any("error", err))
		return nil
	}
	if !info.Available {
		return nil
	}
	if info.Version == s.settings.GetDismissedVersion() {
		return nil
	}
	s.emitter.EmitEvent(EventUpdateAvailable, info)
	return nil
}

func (s *Service) DismissVersion(version string) error {
	return s.settings.SetDismissedVersion(normalizeVersion(version))
}

func (s *Service) isDue(freq string) bool {
	last := s.settings.GetLastCheckedAt()
	if last == "" {
		return true
	}
	lastT, err := time.Parse(time.RFC3339, last)
	if err != nil {
		return true
	}
	var interval time.Duration
	switch freq {
	case FrequencyStartup:
		return true
	case FrequencyDaily:
		interval = 24 * time.Hour
	case FrequencyWeekly:
		interval = 7 * 24 * time.Hour
	default:
		return false
	}
	return s.now().Sub(lastT) >= interval
}

func (s *Service) writeLastChecked() {
	if err := s.settings.SetLastCheckedAt(s.now().UTC().Format(time.RFC3339)); err != nil {
		s.log.Warn("failed to persist lastCheckedAt", slog.Any("error", err))
	}
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

func (s *Service) fetchLatest(ctx context.Context) (tag, url, body string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.releasesURL, nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Vervet-UpdateCheck")
	resp, err := s.http.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", "", "", err
	}
	return rel.TagName, rel.HTMLURL, rel.Body, nil
}

func normalizeVersion(v string) string {
	if len(v) > 0 && (v[0] == 'v' || v[0] == 'V') {
		return v[1:]
	}
	return v
}
