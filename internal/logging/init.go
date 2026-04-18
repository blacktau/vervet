package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"vervet/internal/models"

	"gopkg.in/natefinch/lumberjack.v2"
)

const logFileName = "vervet.log"

var (
	levelVar = new(slog.LevelVar)
	mu       sync.Mutex
	lumber   *lumberjack.Logger
)

func Init(cfg models.LoggingSettings, isDev bool) (*slog.Logger, error) {
	mu.Lock()
	defer mu.Unlock()

	levelVar.Set(ParseLevel(cfg.Level))
	opts := &slog.HandlerOptions{Level: levelVar}

	if lumber != nil {
		_ = lumber.Close()
		lumber = nil
	}

	var handlers []slog.Handler
	if cfg.ConsoleEnabled {
		handlers = append(handlers, slog.NewTextHandler(os.Stdout, opts))
	}
	if cfg.FileEnabled {
		dir, err := LogDir()
		if err != nil {
			return slog.New(slog.NewTextHandler(os.Stderr, opts)), fmt.Errorf("log dir: %w", err)
		}
		lumber = &lumberjack.Logger{
			Filename:   filepath.Join(dir, logFileName),
			MaxSize:    maxSizeMB(cfg.MaxSizeMB),
			MaxBackups: maxBackups(cfg.MaxBackups),
			Compress:   false,
		}
		handlers = append(handlers, slog.NewTextHandler(lumber, opts))
	}

	if len(handlers) == 0 {
		return slog.New(slog.NewTextHandler(os.Stderr, opts)), nil
	}
	return slog.New(NewMultiHandler(handlers...)), nil
}

func SetLevel(level slog.Level) {
	levelVar.Set(level)
}

func Close() error {
	mu.Lock()
	defer mu.Unlock()
	if lumber == nil {
		return nil
	}
	err := lumber.Close()
	lumber = nil
	return err
}

func maxSizeMB(v int) int {
	if v <= 0 {
		return 10
	}
	return v
}

func maxBackups(v int) int {
	if v < 0 {
		return 0
	}
	if v == 0 {
		return 5
	}
	return v
}

var _ io.Writer = (*lumberjack.Logger)(nil)
