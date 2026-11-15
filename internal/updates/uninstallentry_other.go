//go:build !windows

package updates

// SyncUninstallVersion only has work to do on Windows.
func SyncUninstallVersion(string) error { return nil }
