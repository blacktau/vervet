package updates

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

func TestArtifactTarget(t *testing.T) {
	cases := []struct {
		name, platform, arch string
		ok                   bool
	}{
		{"Vervet-linux-amd64.AppImage", "linux", "amd64", true},
		{"Vervet-windows-arm64-portable.zip", "windows", "arm64", true},
		{"Vervet-macos-arm64.app.zip", "darwin", "arm64", true},
		{"Vervet-windows-amd64-installer.exe", "", "", false},
		{"Vervet-linux-amd64.deb", "", "", false},
		{"Vervet-linux-amd64.rpm", "", "", false},
		{"Vervet-macos-arm64.dmg", "", "", false},
	}
	for _, c := range cases {
		p, a, ok := ArtifactTarget(c.name)
		if p != c.platform || a != c.arch || ok != c.ok {
			t.Errorf("ArtifactTarget(%q) = %q, %q, %v; want %q, %q, %v", c.name, p, a, ok, c.platform, c.arch, c.ok)
		}
	}
}

func TestBuildManifestSkipsUninstallableAndNormalisesVersion(t *testing.T) {
	priv, _ := mustKey(t)
	dir := t.TempDir()
	paths := writeFiles(t, dir, map[string]string{
		"Vervet-linux-amd64.AppImage":        "appimage",
		"Vervet-windows-amd64-installer.exe": "installer",
		"Vervet-linux-amd64.deb":             "deb",
	})

	m, err := BuildManifest("v2026.08.4", "https://example.test/dl/", priv, paths, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if m.Version != "2026.8.4" || m.Name != "2026.08.4" {
		t.Errorf("version = %q, name = %q", m.Version, m.Name)
	}
	if len(m.Artifacts) != 1 || m.Artifacts[0].URL != "https://example.test/dl/Vervet-linux-amd64.AppImage" {
		t.Errorf("artifacts = %+v", m.Artifacts)
	}
}

func TestBuildManifestRejectsNoArtifactsAndDevVersion(t *testing.T) {
	priv, _ := mustKey(t)
	paths := writeFiles(t, t.TempDir(), map[string]string{"Vervet-linux-amd64.deb": "deb"})
	if _, err := BuildManifest("2026.11.2", "https://x", priv, paths, time.Now()); err == nil {
		t.Error("expected error with no installable artifacts")
	}
	if _, err := BuildManifest("dev", "https://x", priv, paths, time.Now()); err == nil {
		t.Error("expected error for dev version")
	}
}

// The tests below run the real Wails updater against a manifest built here,
// so a change in what the updater expects shows up as a failure.

func TestManifestInstallsWithWailsUpdater(t *testing.T) {
	priv, pub := mustKey(t)
	u := serveRelease(t, priv, pub, "new binary", nil)

	if err := u.DownloadAndInstall(context.Background()); err != nil {
		t.Fatalf("DownloadAndInstall: %v", err)
	}
	got, err := os.ReadFile(u.DownloadedPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new binary" {
		t.Errorf("downloaded %q", got)
	}
}

func TestManifestRejectsTamperedArtifact(t *testing.T) {
	priv, pub := mustKey(t)
	u := serveRelease(t, priv, pub, "new binary", []byte("new binarY"))
	if err := u.DownloadAndInstall(context.Background()); err == nil {
		t.Fatal("tampered artifact installed")
	}
}

func TestManifestRejectsOtherKey(t *testing.T) {
	priv, _ := mustKey(t)
	_, otherPub := mustKey(t)
	u := serveRelease(t, priv, otherPub, "new binary", nil)
	if err := u.DownloadAndInstall(context.Background()); err == nil {
		t.Fatal("artifact signed with another key installed")
	}
}

// serveRelease signs content into a manifest for the running platform, serves
// it (or replacement bytes in place of the signed content), and returns an
// updater that has found the release.
func serveRelease(t *testing.T, key ed25519.PrivateKey, pub ed25519.PublicKey, content string, replacement []byte) *updater.Updater {
	t.Helper()
	name := assetNameFor(runtime.GOOS, runtime.GOARCH)
	if name == "" {
		t.Skipf("no release asset for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	// A plain file for every platform keeps the test about verification;
	// the updater only unpacks names ending in .zip.
	name += ".bin"
	dir := t.TempDir()
	paths := writeFiles(t, dir, map[string]string{name: content})

	served := []byte(content)
	if replacement != nil {
		served = replacement
	}
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	m, err := buildManifestUnchecked("2099.01.1", srv.URL, key, paths)
	if err != nil {
		t.Fatal(err)
	}
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(m)
	})
	mux.HandleFunc("/"+name, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(served)
	})

	provider, err := endpoint.New(endpoint.Config{URL: srv.URL + "/manifest.json"})
	if err != nil {
		t.Fatal(err)
	}
	u := updater.New(nopHost{})
	if err = u.Init(updater.Config{
		CurrentVersion: SemverVersion("2026.08.4"),
		Providers:      []updater.Provider{provider},
		PublicKey:      pub,
		Window:         updater.WindowNone,
	}); err != nil {
		t.Fatal(err)
	}
	rel, err := u.Check(context.Background())
	if err != nil || rel == nil {
		t.Fatalf("Check = %v, %v", rel, err)
	}
	return u
}

// buildManifestUnchecked is BuildManifest without the asset-name filter, so
// the test can serve a plain file under any platform.
func buildManifestUnchecked(version, baseURL string, key ed25519.PrivateKey, paths []string) (Manifest, error) {
	m := Manifest{SchemaVersion: 1, Version: SemverVersion(version), Name: version}
	for _, p := range paths {
		art, err := signFile(key, p)
		if err != nil {
			return Manifest{}, err
		}
		art.URL = baseURL + "/" + filepath.Base(p)
		art.Filename = filepath.Base(p)
		art.Platform = runtime.GOOS
		art.Arch = runtime.GOARCH
		m.Artifacts = append(m.Artifacts, art)
	}
	return m, nil
}

func assetNameFor(goos, goarch string) string {
	switch goos {
	case "linux":
		return "Vervet-linux-" + goarch
	case "windows":
		return "Vervet-windows-" + goarch
	case "darwin":
		return "Vervet-macos-" + goarch
	}
	return ""
}

func mustKey(t *testing.T) (ed25519.PrivateKey, ed25519.PublicKey) {
	t.Helper()
	privB64, pubB64, err := GenerateSigningKey()
	if err != nil {
		t.Fatal(err)
	}
	key, err := ParseSigningKey(privB64)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil {
		t.Fatal(err)
	}
	return key, ed25519.PublicKey(pub)
}

func writeFiles(t *testing.T, dir string, files map[string]string) []string {
	t.Helper()
	var paths []string
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	return paths
}

type nopHost struct{}

func (nopHost) Emit(string, ...any) bool                              { return true }
func (nopHost) OnEvent(string, func(any)) func()                      { return func() {} }
func (nopHost) OpenWindow(updater.WindowOptions) updater.WindowHandle { return nil }
func (nopHost) Quit()                                                 {}
