package updates

import (
	"os"
	"path/filepath"
	"strings"
)

// InstallKind is how this copy of Vervet was installed, which decides whether
// it can replace itself.
type InstallKind string

const (
	InstallAppImage       InstallKind = "appimage"
	InstallWindows        InstallKind = "windows"        // portable zip or the installer
	InstallMacApp         InstallKind = "macos-app"      // a .app bundle
	InstallPackageManager InstallKind = "packagemanager" // deb, rpm, COPR
	InstallOther          InstallKind = "other"          // a bare binary, e.g. wails3 task build
)

// Install describes the running copy.
type Install struct {
	Kind InstallKind
	// Target is what an update replaces: the AppImage file, the exe or the .app.
	Target string
	// CanSelfUpdate is true when Vervet may download an update and swap Target.
	CanSelfUpdate bool
}

// installEnv is what DetectInstall reads from the system; tests replace it.
type installEnv struct {
	goos     string
	exe      string
	getenv   func(string) string
	writable func(dir string) bool
}

// DetectInstall works out how the running copy was installed.
func DetectInstall(goos string) Install {
	exe, err := installedExe()
	if err != nil {
		return Install{Kind: InstallOther}
	}
	return detectInstall(installEnv{goos: goos, exe: exe, getenv: os.Getenv, writable: dirWritable})
}

func detectInstall(env installEnv) Install {
	inst := classify(env)
	switch inst.Kind {
	case InstallAppImage, InstallWindows, InstallMacApp:
		// An all-users Windows install, or a .app in an /Applications owned by
		// another admin, can't be replaced without elevation; those update
		// from the release page instead.
		inst.CanSelfUpdate = env.writable(filepath.Dir(inst.Target))
	}
	return inst
}

func classify(env installEnv) Install {
	if appImage := env.getenv("APPIMAGE"); appImage != "" && env.goos == "linux" {
		return Install{Kind: InstallAppImage, Target: appImage}
	}
	switch env.goos {
	case "windows":
		return Install{Kind: InstallWindows, Target: env.exe}
	case "darwin":
		if bundle, ok := enclosingApp(env.exe); ok {
			return Install{Kind: InstallMacApp, Target: bundle}
		}
	case "linux":
		if strings.HasPrefix(env.exe, "/usr/") || strings.HasPrefix(env.exe, "/opt/") {
			return Install{Kind: InstallPackageManager, Target: env.exe}
		}
	}
	// A bare Linux or macOS binary isn't something a release asset can
	// replace: the Linux artifact is an AppImage and the macOS one a .app.
	return Install{Kind: InstallOther, Target: env.exe}
}

// enclosingApp returns the .app bundle that contains exe, if any.
func enclosingApp(exe string) (string, bool) {
	for dir := filepath.Dir(exe); dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if strings.HasSuffix(dir, ".app") {
			return dir, true
		}
	}
	return "", false
}

// dirWritable reports whether a file can be created in dir, which is what
// swapping the update in needs. Checking permission bits would miss ACLs,
// read-only mounts and UAC virtualisation.
func dirWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".vervet-update-check-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

func installedExe() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Clean(exe), nil
}
