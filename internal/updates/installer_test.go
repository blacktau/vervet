package updates

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

type fakeInstaller struct {
	initCfg     *updater.Config
	initErr     error
	release     *updater.Release
	checkErr    error
	installed   bool
	restarted   bool
	downloadErr error
}

func (f *fakeInstaller) Init(cfg updater.Config) error {
	f.initCfg = &cfg
	return f.initErr
}

func (f *fakeInstaller) Check(context.Context) (*updater.Release, error) {
	return f.release, f.checkErr
}

func (f *fakeInstaller) DownloadAndInstall(context.Context) error {
	f.installed = true
	return f.downloadErr
}

func (f *fakeInstaller) Restart(context.Context) error {
	f.restarted = true
	return nil
}

func newInstallService(inst *fakeInstaller, install Install, version string) *Service {
	return NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), Config{
		CurrentVersion: version,
		Settings:       &fakeSettings{},
		Emit:           func(string, ...any) {},
		Installer:      inst,
		Install:        install,
	})
}

var selfUpdating = Install{Kind: InstallAppImage, Target: "/x.AppImage", CanSelfUpdate: true}

func TestInstallerConfiguredForSelfUpdatingRelease(t *testing.T) {
	inst := &fakeInstaller{}
	s := newInstallService(inst, selfUpdating, "2026.08.4")
	if !s.CanInstall() {
		t.Fatal("CanInstall = false")
	}
	if inst.initCfg == nil {
		t.Fatal("installer not initialised")
	}
	if inst.initCfg.CurrentVersion != "2026.8.4" {
		t.Errorf("CurrentVersion = %q, want semver form", inst.initCfg.CurrentVersion)
	}
	if len(inst.initCfg.PublicKey) != 32 {
		t.Errorf("PublicKey is %d bytes", len(inst.initCfg.PublicKey))
	}
	if inst.initCfg.Window != updater.WindowNone {
		t.Error("updater would open its own window")
	}
}

func TestInstallerOffWhenItCantSelfUpdate(t *testing.T) {
	cases := map[string]struct {
		install Install
		version string
		initErr error
	}{
		"package manager": {Install{Kind: InstallPackageManager}, "2026.08.4", nil},
		"read-only dir":   {Install{Kind: InstallWindows, CanSelfUpdate: false}, "2026.08.4", nil},
		"dev build":       {selfUpdating, "dev", nil},
		"init fails":      {selfUpdating, "2026.08.4", errors.New("boom")},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			inst := &fakeInstaller{initErr: c.initErr}
			s := newInstallService(inst, c.install, c.version)
			if s.CanInstall() {
				t.Error("CanInstall = true")
			}
			if err := s.Install(context.Background()); !errors.Is(err, ErrCannotInstall) {
				t.Errorf("Install err = %v", err)
			}
			if err := s.Restart(context.Background()); !errors.Is(err, ErrCannotInstall) {
				t.Errorf("Restart err = %v", err)
			}
		})
	}
}

func TestInstallDownloadsFoundRelease(t *testing.T) {
	inst := &fakeInstaller{release: &updater.Release{Version: "2026.11.2"}}
	s := newInstallService(inst, selfUpdating, "2026.08.4")
	if err := s.Install(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !inst.installed {
		t.Error("not downloaded")
	}
	if err := s.Restart(context.Background()); err != nil || !inst.restarted {
		t.Errorf("Restart = %v, restarted = %v", err, inst.restarted)
	}
}

func TestInstallWithoutSignedBuild(t *testing.T) {
	inst := &fakeInstaller{}
	s := newInstallService(inst, selfUpdating, "2026.08.4")
	if err := s.Install(context.Background()); !errors.Is(err, ErrNothingToInstall) {
		t.Errorf("err = %v", err)
	}
	if inst.installed {
		t.Error("downloaded with no release")
	}
}

func TestCheckNowReportsCanInstall(t *testing.T) {
	srv := newTestServer(t, "v2026.11.2", "https://example.test/r", "notes")
	defer srv.Close()
	s := NewService(slog.New(slog.NewTextHandler(os.Stderr, nil)), Config{
		CurrentVersion: "2026.08.4",
		ReleasesURL:    srv.URL,
		Settings:       &fakeSettings{},
		Emit:           func(string, ...any) {},
		Installer:      &fakeInstaller{},
		Install:        selfUpdating,
	})
	info, err := s.CheckNow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available || !info.CanInstall {
		t.Errorf("info = %+v", info)
	}
}
