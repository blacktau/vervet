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

type SystemProxy struct {
	log     *slog.Logger
	service SystemProvider
	browser Browser
}

// Browser opens web links. It refuses anything but http(s), so a link in a
// document can't launch file://, smb:// or a custom URL handler.
type Browser interface {
	OpenURL(url string) error
}

type SystemProvider interface {
	Log(level string, message string)
	RevealLogsFolder() error
}

func NewSystemProxy(log *slog.Logger, ss SystemProvider, browser Browser) *SystemProxy {
	return &SystemProxy{
		log:     log,
		service: ss,
		browser: browser,
	}
}

// OpenURL opens an http(s) link in the user's browser. The frontend uses this
// rather than the Wails runtime's Browser.OpenURL, which allows other schemes.
func (sp *SystemProxy) OpenURL(url string) EmptyResult {
	if err := sp.browser.OpenURL(url); err != nil {
		logFail(sp.log, "OpenURL", err)
		return Fail(err)
	}
	return Success()
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
