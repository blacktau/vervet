package api

import (
	"context"
	"fmt"
	"log/slog"

	"vervet/internal/models"
	"vervet/internal/workspaces"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type WorkspacesProxy struct {
	log              *slog.Logger
	ctx              context.Context
	service          *workspaces.WorkspaceService
	settingsProvider SettingsProvider
}

func NewWorkspacesProxy(log *slog.Logger, service *workspaces.WorkspaceService, settingsProvider SettingsProvider) *WorkspacesProxy {
	return &WorkspacesProxy{log: log, service: service, settingsProvider: settingsProvider}
}

func (p *WorkspacesProxy) Init(ctx context.Context) {
	p.ctx = ctx
}

func (p *WorkspacesProxy) GetWorkspaces() Result[models.WorkspaceData] {
	data, err := p.service.GetWorkspaces()
	if err != nil {
		logFail(p.log, "GetWorkspaces", err)
		return FailResult[models.WorkspaceData](err)
	}
	return SuccessResult(data)
}

func (p *WorkspacesProxy) CreateWorkspace(name string) Result[models.Workspace] {
	ws, err := p.service.CreateWorkspace(name)
	if err != nil {
		logFail(p.log, "CreateWorkspace", err)
		return FailResult[models.Workspace](err)
	}
	return SuccessResult(ws)
}

func (p *WorkspacesProxy) RenameWorkspace(id, name string) EmptyResult {
	err := p.service.RenameWorkspace(id, name)
	if err != nil {
		logFail(p.log, "RenameWorkspace", err)
		return Fail(err)
	}
	return Success()
}

func (p *WorkspacesProxy) DeleteWorkspace(id string) EmptyResult {
	err := p.service.DeleteWorkspace(id)
	if err != nil {
		logFail(p.log, "DeleteWorkspace", err)
		return Fail(err)
	}
	return Success()
}

func (p *WorkspacesProxy) SetActiveWorkspace(id string) EmptyResult {
	err := p.service.SetActiveWorkspace(id)
	if err != nil {
		logFail(p.log, "SetActiveWorkspace", err)
		return Fail(err)
	}
	return Success()
}

func (p *WorkspacesProxy) AddFolder(workspaceID string) Result[string] {
	path, err := wailsRuntime.OpenDirectoryDialog(p.ctx, wailsRuntime.OpenDialogOptions{})
	if err != nil {
		logFail(p.log, "AddFolder", err)
		return FailResult[string](err)
	}
	if path == "" {
		return FailResult[string](fmt.Errorf("no folder selected"))
	}
	if err := p.service.AddFolder(workspaceID, path); err != nil {
		logFail(p.log, "AddFolder", err)
		return FailResult[string](err)
	}
	return SuccessResult(path)
}

func (p *WorkspacesProxy) RemoveFolder(workspaceID, path string) EmptyResult {
	err := p.service.RemoveFolder(workspaceID, path)
	if err != nil {
		logFail(p.log, "RemoveFolder", err)
		return Fail(err)
	}
	return Success()
}

func (p *WorkspacesProxy) ReadDirectory(path string) Result[[]models.DirectoryEntry] {
	settings, err := p.settingsProvider.GetSettings()
	if err != nil {
		logFail(p.log, "ReadDirectory", err)
		return FailResult[[]models.DirectoryEntry](err)
	}
	entries, err := p.service.ReadDirectory(path, settings.Workspaces.FileExtensions)
	if err != nil {
		logFail(p.log, "ReadDirectory", err)
		return FailResult[[]models.DirectoryEntry](err)
	}
	return SuccessResult(entries)
}

func (p *WorkspacesProxy) CreateFolder(dirPath, name string) Result[string] {
	fullPath, err := p.service.CreateFolder(dirPath, name)
	if err != nil {
		logFail(p.log, "CreateFolder", err)
		return FailResult[string](err)
	}
	return SuccessResult(fullPath)
}

func (p *WorkspacesProxy) CreateFile(dirPath, name string) Result[string] {
	fullPath, err := p.service.CreateFile(dirPath, name)
	if err != nil {
		logFail(p.log, "CreateFile", err)
		return FailResult[string](err)
	}
	return SuccessResult(fullPath)
}

func (p *WorkspacesProxy) RenameFile(oldPath, newPath string) EmptyResult {
	err := p.service.RenameFile(oldPath, newPath)
	if err != nil {
		logFail(p.log, "RenameFile", err)
		return Fail(err)
	}
	return Success()
}

func (p *WorkspacesProxy) DeleteFile(path string) EmptyResult {
	err := p.service.DeleteFile(path)
	if err != nil {
		logFail(p.log, "DeleteFile", err)
		return Fail(err)
	}
	return Success()
}
