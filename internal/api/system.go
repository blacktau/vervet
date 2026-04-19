package api

import (
	"log/slog"
	"runtime"
)

type OperatingSystem string

const (
	Windows OperatingSystem = "windows"
	Linux   OperatingSystem = "linux"
	OSX     OperatingSystem = "darwin"
)

var AllOperatingSystems = []struct {
	Value  OperatingSystem
	TSName string
}{
	{Windows, "WINDOWS"},
	{Linux, "LINUX"},
	{OSX, "OSX"},
}

type SystemProxy struct {
	log     *slog.Logger
	service SystemProvider
}

type SystemProvider interface {
	Log(level string, message string)
	RevealLogsFolder() error
}

func NewSystemProxy(log *slog.Logger, ss SystemProvider) *SystemProxy {
	return &SystemProxy{
		log:     log,
		service: ss,
	}
}

func (sp *SystemProxy) GetOs() Result[OperatingSystem] {
	var os OperatingSystem
	switch runtime.GOOS {
	case "windows":
		os = Windows
	case "darwin":
		os = OSX
	case "linux":
		os = Linux
	default:
		os = Windows
	}

	return Result[OperatingSystem]{
		IsSuccess: true,
		Data:      os,
	}
}

func (sp *SystemProxy) Log(level string, message string) {
	sp.service.Log(level, message)
}

func (sp *SystemProxy) RevealLogsFolder() EmptyResult {
	if err := sp.service.RevealLogsFolder(); err != nil {
		logFail(sp.log, "RevealLogsFolder", err)
		return Fail(err)
	}
	return Success()
}
