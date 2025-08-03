package logging

import (
	"log/slog"
)

const (
	SourceKey = "source"
)

type LogAdapter struct {
	*slog.Logger
}

func NewLogger(logger *slog.Logger) LogAdapter {
	return LogAdapter{
		logger,
	}
}

func (l LogAdapter) Print(message string) {
	l.Logger.Info(message)
}

func (l LogAdapter) Trace(message string) {
	l.Logger.Debug(message)
}

func (l LogAdapter) Debug(message string) {
	l.Logger.Debug(message)
}

func (l LogAdapter) Info(message string) {
	l.Logger.Info(message)
}

func (l LogAdapter) Warning(message string) {
	l.Logger.Warn(message)
}

func (l LogAdapter) Error(message string) {
	l.Logger.Error(message)
}

func (l LogAdapter) Fatal(message string) {
	l.Logger.Error(message)
	panic(message)
}