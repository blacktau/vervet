package api

import (
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

type SystemProxy struct{}

func NewSystemProxy() *SystemProxy {
	return &SystemProxy{}
}

func (sp *SystemProxy) GetOs() Result[OperatingSystem] {
	os := Windows

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
