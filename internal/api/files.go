package api

import (
	"log/slog"

	"vervet/internal/models"
)

type FilesProvider interface {
	SelectFile(title string, filters []models.FileFilter) (string, error)
	SaveFile(title *string, name *string, filters []models.FileFilter) (string, error)
	ReadFile(path string) (string, error)
	WriteFile(path string, content string) error
}

type FilesProxy struct {
	log     *slog.Logger
	service FilesProvider
}

func NewFilesProxy(log *slog.Logger, service FilesProvider) *FilesProxy {
	return &FilesProxy{log: log, service: service}
}

func (fp *FilesProxy) SelectFile(title string, filters []models.FileFilter) Result[string] {
	path, err := fp.service.SelectFile(title, filters)
	if err != nil {
		logFail(fp.log, "SelectFile", err)
		return FailResult[string](err)
	}
	return SuccessResult(path)
}

func (fp *FilesProxy) SaveFile(title *string, defaultName *string, filters []models.FileFilter) Result[string] {
	path, err := fp.service.SaveFile(title, defaultName, filters)
	if err != nil {
		logFail(fp.log, "SaveFile", err)
		return FailResult[string](err)
	}
	return SuccessResult(path)
}

func (fp *FilesProxy) ReadFile(path string) Result[string] {
	content, err := fp.service.ReadFile(path)
	if err != nil {
		logFail(fp.log, "ReadFile", err)
		return FailResult[string](err)
	}
	return SuccessResult(content)
}

func (fp *FilesProxy) WriteFile(path string, content string) EmptyResult {
	err := fp.service.WriteFile(path, content)
	if err != nil {
		logFail(fp.log, "WriteFile", err)
		return Fail(err)
	}
	return Success()
}
