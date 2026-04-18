package logging_test

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"vervet/internal/logging"
	"vervet/internal/models"
)

func setupLogHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	switch runtime.GOOS {
	case "linux":
		t.Setenv("XDG_STATE_HOME", tmp)
	case "darwin":
		t.Setenv("HOME", tmp)
	case "windows":
		t.Setenv("LOCALAPPDATA", tmp)
	}
	return tmp
}

func TestInit_WritesToFile(t *testing.T) {
	setupLogHome(t)
	t.Cleanup(func() { _ = logging.Close() })

	cfg := models.LoggingSettings{
		Level:          "debug",
		ConsoleEnabled: false,
		FileEnabled:    true,
		MaxSizeMB:      1,
		MaxBackups:     1,
	}
	logger, err := logging.Init(cfg, false)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	logger.Info("file-target", slog.String("k", "v"))

	dir, err := logging.LogDir()
	if err != nil {
		t.Fatalf("LogDir: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "vervet.log"))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(b), "file-target") {
		t.Errorf("log file missing entry: %q", string(b))
	}
}

func TestInit_ConsoleOnly(t *testing.T) {
	setupLogHome(t)
	t.Cleanup(func() { _ = logging.Close() })

	cfg := models.LoggingSettings{
		Level:          "info",
		ConsoleEnabled: true,
		FileEnabled:    false,
	}
	logger, err := logging.Init(cfg, false)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	logger.Info("console-only")

	dir, err := logging.LogDir()
	if err != nil {
		t.Fatalf("LogDir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "vervet.log")); !os.IsNotExist(err) {
		t.Errorf("expected no log file, stat err = %v", err)
	}
}

func TestInit_NoHandlersFallsBackToStderr(t *testing.T) {
	setupLogHome(t)
	t.Cleanup(func() { _ = logging.Close() })

	orig := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	cfg := models.LoggingSettings{
		Level:          "info",
		ConsoleEnabled: false,
		FileEnabled:    false,
	}
	logger, err := logging.Init(cfg, false)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	logger.Info("fallback-msg")
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "fallback-msg") {
		t.Errorf("expected stderr fallback to contain message, got %q", buf.String())
	}
}

func TestSetLevel_HotSwap(t *testing.T) {
	setupLogHome(t)
	t.Cleanup(func() { _ = logging.Close() })

	cfg := models.LoggingSettings{
		Level:          "info",
		ConsoleEnabled: true,
		FileEnabled:    false,
	}
	if _, err := logging.Init(cfg, false); err != nil {
		t.Fatalf("Init: %v", err)
	}

	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	logger, _ := logging.Init(cfg, false)
	logger.Debug("should-not-appear")
	logging.SetLevel(slog.LevelDebug)
	logger.Debug("should-appear")
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()
	if strings.Contains(out, "should-not-appear") {
		t.Errorf("debug leaked at info level: %q", out)
	}
	if !strings.Contains(out, "should-appear") {
		t.Errorf("debug missing after SetLevel: %q", out)
	}
}
