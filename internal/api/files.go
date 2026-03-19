package api

import "context"

type FileFilter struct {
	DisplayName string `json:"displayName"`
	Pattern     string `json:"pattern"`
}

type FilesProvider interface {
	Init(ctx context.Context) error
	SelectFile(title string, filters []FileFilter) (string, error)
	SaveFile(title *string, name *string, filters []FileFilter) (string, error)
	ReadFile(path string) (string, error)
	WriteFile(path string, content string) error
}

type FilesProxy struct {
	service FilesProvider
}

func NewFilesProxy(service FilesProvider) *FilesProxy {
	return &FilesProxy{service: service}
}

func (fp *FilesProxy) SelectFile(title string, filters []FileFilter) Result[string] {
	path, err := fp.service.SelectFile(title, filters)
	if err != nil {
		return Result[string]{Error: err.Error()}
	}
	return Result[string]{IsSuccess: true, Data: path}
}

func (fp *FilesProxy) SaveFile(title *string, defaultName *string, filters []FileFilter) Result[string] {
	path, err := fp.service.SaveFile(title, defaultName, filters)
	if err != nil {
		return Result[string]{Error: err.Error()}
	}
	return Result[string]{IsSuccess: true, Data: path}
}

func (fp *FilesProxy) ReadFile(path string) Result[string] {
	content, err := fp.service.ReadFile(path)
	if err != nil {
		return Result[string]{Error: err.Error()}
	}
	return Result[string]{IsSuccess: true, Data: content}
}

func (fp *FilesProxy) WriteFile(path string, content string) EmptyResult {
	err := fp.service.WriteFile(path, content)
	if err != nil {
		return Error(err.Error())
	}
	return Success()
}
