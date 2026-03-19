package api

import "context"

type FilesProvider interface {
	Init(ctx context.Context) error
	SelectFile(title string, extensions *[]string) (string, error)
	SaveFile(title *string, name *string, extensions *[]string) (string, error)
	ReadFile(path string) (string, error)
	WriteFile(path string, content string) error
}

type FilesProxy struct {
	service FilesProvider
}

func NewFilesProxy(service FilesProvider) *FilesProxy {
	return &FilesProxy{service: service}
}

func (fp *FilesProxy) SelectFile(title string, extensions *[]string) Result[string] {
	path, err := fp.service.SelectFile(title, extensions)
	if err != nil {
		return Result[string]{Error: err.Error()}
	}
	return Result[string]{IsSuccess: true, Data: path}
}

func (fp *FilesProxy) SaveFile(title *string, defaultName *string, extensions *[]string) Result[string] {
	path, err := fp.service.SaveFile(title, defaultName, extensions)
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
