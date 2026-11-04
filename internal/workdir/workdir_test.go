package workdir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGet(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	owd := t.TempDir()

	tests := []struct {
		name   string
		appDir string
		owd    string
		want   string
	}{
		{"outside an AppImage uses the process directory", "", owd, cwd},
		{"inside an AppImage uses the launch directory", "/tmp/.mount_x", owd, owd},
		{"falls back when OWD is unset", "/tmp/.mount_x", "", cwd},
		{"falls back when OWD no longer exists", "/tmp/.mount_x", filepath.Join(owd, "gone"), cwd},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APPDIR", tt.appDir)
			t.Setenv("OWD", tt.owd)

			got, err := Get()
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("Get() = %q, want %q", got, tt.want)
			}
		})
	}
}
