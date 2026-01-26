package api

import (
	"context"
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
	service SystemProvider
}

type SystemProvider interface {
	Init(ctx context.Context) error
	SelectFile(title string, extensions *[]string) (string, error)
	SaveFile(title *string, name *string, extensions *[]string) (string, error)
}

func NewSystemProxy(ss SystemProvider) *SystemProxy {
	return &SystemProxy{
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

func (sp *SystemProxy) SelectFile(title string, extensions *[]string) Result[string] {
	path, err := sp.service.SelectFile(title, extensions)

	if err != nil {
		return Result[string]{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}

	return Result[string]{
		IsSuccess: true,
		Data:      path,
	}
}

func (sp *SystemProxy) SaveFile(title, defaultName *string, extensions *[]string) Result[string] {
	path, err := sp.service.SaveFile(title, defaultName, extensions)
	if err != nil {
		return Result[string]{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}
	return Result[string]{
		IsSuccess: true,
		Data:      path,
	}
}