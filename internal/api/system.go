package api

import "runtime"

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
}

func NewSystemProxy() *SystemProxy {
	return &SystemProxy{}
}

func (sp *SystemProxy) GetOs() Result[OperatingSystem] {
	os := Windows
	if runtime.GOOS == "windows" {
		os = Windows
	} else if runtime.GOOS == "darwin" {
		os = OSX
	} else if runtime.GOOS == "linux" {
		os = Linux
	}

	return Result[OperatingSystem]{
		IsSuccess: true,
		Data:      os,
	}
}
