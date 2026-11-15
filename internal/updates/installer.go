package updates

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

// Installer is the part of the Wails updater the service drives.
// *updater.Updater satisfies it.
type Installer interface {
	Init(cfg updater.Config) error
	Check(ctx context.Context) (*updater.Release, error)
	DownloadAndInstall(ctx context.Context) error
	Restart(ctx context.Context) error
}

// ErrCannotInstall means this copy of Vervet can't install updates itself;
// the user installs them from the release page.
var ErrCannotInstall = errors.New("this copy of Vervet can't install updates itself")

// ErrNothingToInstall means the release has no signed build for this
// platform, e.g. one published before in-app updates existed.
var ErrNothingToInstall = errors.New("no installable update was found for this platform")

// NewInstaller returns a Wails updater that emits its events (the
// wails:updater:* names) through emit, and quits the app through quit once
// the restart helper is waiting. It never opens a window of its own.
func NewInstaller(emit Emit, quit func()) *updater.Updater {
	return updater.New(installerHost{emit: emit, quit: quit})
}

type installerHost struct {
	emit Emit
	quit func()
}

func (h installerHost) Emit(name string, data ...any) bool {
	if h.emit != nil {
		h.emit(name, data...)
	}
	return true
}

func (installerHost) OnEvent(string, func(any)) func() { return func() {} }

// OpenWindow is never called: the updater is configured with WindowNone.
func (installerHost) OpenWindow(updater.WindowOptions) updater.WindowHandle { return nil }

func (h installerHost) Quit() {
	if h.quit != nil {
		h.quit()
	}
}

func (s *Service) initInstaller(cfg Config) {
	if cfg.Installer == nil || !cfg.Install.CanSelfUpdate || !IsReleaseVersion(cfg.CurrentVersion) {
		return
	}
	if err := configureInstaller(cfg); err != nil {
		s.log.Warn("in-app updates are off", slog.Any("error", err))
		return
	}
	s.installer = cfg.Installer
	s.canInstall = true
}

func configureInstaller(cfg Config) error {
	key, err := base64.StdEncoding.DecodeString(updatePublicKey)
	if err != nil {
		return fmt.Errorf("update public key: %w", err)
	}
	url := cfg.ManifestURL
	if url == "" {
		url = defaultManifestURL()
	}
	provider, err := endpoint.New(endpoint.Config{URL: url, HTTPClient: cfg.HTTPClient})
	if err != nil {
		return err
	}
	return cfg.Installer.Init(updater.Config{
		CurrentVersion: SemverVersion(cfg.CurrentVersion),
		Providers:      []updater.Provider{provider},
		PublicKey:      key,
		Window:         updater.WindowNone,
	})
}

// CanInstall reports whether this copy can install updates itself.
func (s *Service) CanInstall() bool {
	return s.canInstall
}

// Install downloads, verifies and stages the newest release. Progress is
// reported through the wails:updater:* events. Call Restart afterwards.
func (s *Service) Install(ctx context.Context) error {
	if !s.canInstall {
		return ErrCannotInstall
	}
	rel, err := s.installer.Check(ctx)
	if err != nil {
		return err
	}
	if rel == nil {
		return ErrNothingToInstall
	}
	return s.installer.DownloadAndInstall(ctx)
}

// Restart replaces the running copy with the staged update and relaunches
// it. It quits the app, so callers must deal with unsaved work first.
func (s *Service) Restart(ctx context.Context) error {
	if !s.canInstall {
		return ErrCannotInstall
	}
	return s.installer.Restart(ctx)
}
