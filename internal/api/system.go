package api

import (
	"runtime"
	"vervet/internal/system"
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
	service system.Service
}

func NewSystemProxy(ss system.Service) *SystemProxy {
	return &SystemProxy{
		service: ss,
	}
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

func (sp *SystemProxy) SelectFile(title string, extensions *[]string) Result[string] {
	path, err := sp.service.SelectFile(title, extensions)
	return Result[string]{
		IsSuccess: err == nil,
		Data:      path,
		Error:     err.Error(),
	}
}

func (sp *SystemProxy) SaveFile(title, defaultName *string, extensions *[]string) Result[string] {
	path, err := sp.service.SaveFile(title, defaultName, extensions)
	return Result[string]{
		IsSuccess: err == nil,
		Data:      path,
		Error:     err.Error(),
	}
}
