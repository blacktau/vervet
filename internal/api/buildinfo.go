package api

import (
	"log/slog"

	"vervet/internal/buildinfo"
)

// BuildInfoProxy exposes build-time metadata to the frontend.
type BuildInfoProxy struct {
	log *slog.Logger
}

func NewBuildInfoProxy(log *slog.Logger) *BuildInfoProxy {
	return &BuildInfoProxy{log: log}
}

// GetChannel returns the distribution channel of the running binary,
// either "github" or "msstore".
func (p *BuildInfoProxy) GetChannel() Result[string] {
	return SuccessResult(string(buildinfo.Channel()))
}
