package system

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"vervet/internal/logging"
)

type Service struct {
	log *slog.Logger
	ctx context.Context
}

func NewSystemService(log *slog.Logger) *Service {
	return &Service{
		log: log.With(slog.String(logging.SourceKey, "Service")),
	}
}

func (s *Service) Init(ctx context.Context) {
	s.ctx = ctx
}

func (ss *Service) Log(level string, message string) {
	switch level {
	case "debug":
		ss.log.Debug(message, slog.String("origin", "UI"))
	case "info":
		ss.log.Info(message, slog.String("origin", "UI"))
	case "warn":
		ss.log.Warn(message, slog.String("origin", "UI"))
	case "error":
		ss.log.Error(message, slog.String("origin", "UI"))
	default:
		ss.log.Info(message, slog.String("origin", "UI"))
	}
}

func (s *Service) RevealLogsFolder() error {
	dir, err := logging.LogDir()
	if err != nil {
		return fmt.Errorf("resolve log dir: %w", err)
	}
	return openFolder(dir)
}

func openFolder(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}
