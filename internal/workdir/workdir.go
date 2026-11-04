// Package workdir resolves the directory the user launched the app from.
package workdir

import "os"

// Get returns the user's working directory. The AppImage launcher has to start
// the app inside its read-only mount for WebKit to work, so there the process
// working directory means nothing to the user, and the AppImage runtime's $OWD
// holds the real one.
func Get() (string, error) {
	if os.Getenv("APPDIR") != "" {
		owd := os.Getenv("OWD")
		if info, err := os.Stat(owd); err == nil && info.IsDir() {
			return owd, nil
		}
	}
	return os.Getwd()
}
