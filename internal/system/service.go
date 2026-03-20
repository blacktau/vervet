package system

import (
	"log/slog"
	"vervet/internal/logging"
)

type Service struct {
	log *slog.Logger
}

func NewSystemService(log *slog.Logger) *Service {
	return &Service{
		log: log.With(slog.String(logging.SourceKey, "Service")),
	}
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
