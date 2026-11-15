//go:build windows

package updates

import (
	"errors"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// uninstallKey is the installer's uninstall entry: UNINST_KEY in
// build/windows/nsis (company name + product name).
const uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\BlacktauVervet`

// SyncUninstallVersion corrects the version Windows shows under Installed
// apps after an in-app update, which replaces Vervet.exe without the
// installer. Only a per-user install's entry is touched, and only when it
// belongs to the running copy.
func SyncUninstallVersion(version string) error {
	if !IsReleaseVersion(version) {
		return nil
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, uninstallKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer k.Close()

	location, _, err := k.GetStringValue("InstallLocation")
	if err != nil {
		return nil
	}
	exe, err := installedExe()
	if err != nil {
		return err
	}
	if !strings.EqualFold(filepath.Clean(location), filepath.Dir(exe)) {
		return nil
	}
	current, _, _ := k.GetStringValue("DisplayVersion")
	if current == version {
		return nil
	}
	return k.SetStringValue("DisplayVersion", version)
}
