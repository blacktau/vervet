package logging_test

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"vervet/internal/logging"
)

func TestLogDir_CreatesDirectory(t *testing.T) {
	tmp := t.TempDir()
	switch runtime.GOOS {
	case "linux":
		t.Setenv("XDG_STATE_HOME", tmp)
	case "darwin":
		t.Setenv("HOME", tmp)
	case "windows":
		t.Setenv("LOCALAPPDATA", tmp)
	}

	dir, err := logging.LogDir()
	if err != nil {
		t.Fatalf("LogDir() error = %v", err)
	}
	info, statErr := os.Stat(dir)
	if statErr != nil {
		t.Fatalf("stat(%s) = %v", dir, statErr)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", dir)
	}
	if !strings.HasPrefix(dir, tmp) {
		t.Fatalf("expected dir under %s, got %s", tmp, dir)
	}
}
