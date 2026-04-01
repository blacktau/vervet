package api

import (
	"fmt"
	"vervet/internal/models"
)

type ServersProvider interface {
	GetServers() ([]models.RegisteredServer, error)
	AddServer(parentID, name, uri, colour string) error
	UpdateServer(serverID, name, uri, parentID, colour string) error
	RemoveNode(id string) error
	GetURI(id string) (string, error)
	CreateGroup(parentID, name string) error
	UpdateGroup(groupID, name, parentID string) error
	GetServer(id string) (*models.RegisteredServer, error)
	AddServerWithConfig(parentID, name, colour string, cfg models.ConnectionConfig) error
	UpdateServerWithConfig(serverID, name, parentID, colour string, cfg models.ConnectionConfig) error
	GetConnectionConfig(serverID string) (models.ConnectionConfig, error)
	ExportServers(serverIDs []string, includeSensitiveData bool) ([]byte, error)
	ImportServers(data []byte) ([]models.RegisteredServer, error)
}

// ServersProxy exposes the ServerService to the UI
// the proxies serve as a place to handle the idiosyncrasies of the marshaling/unmarshalling to the UI
type ServersProxy struct {
	sm ServersProvider
}

func NewServersProxy(sm ServersProvider) *ServersProxy {
	return &ServersProxy{
		sm: sm,
	}
}

func (sp *ServersProxy) GetServers() Result[[]models.RegisteredServer] {
	registeredServers, err := sp.sm.GetServers()
	if err != nil {
		return FailResult[[]models.RegisteredServer](err)
	}

	return SuccessResult(registeredServers)
}

func (sp *ServersProxy) GetServer(id string) Result[models.RegisteredServer] {
	registerServer, err := sp.sm.GetServer(id)
	if err != nil {
		return FailResult[models.RegisteredServer](err)
	}

	if registerServer == nil {
		return FailResult[models.RegisteredServer](fmt.Errorf("server with id %s not found", id))
	}

	return SuccessResult(*registerServer)
}

func (sp *ServersProxy) CreateGroup(name, parentID string) EmptyResult {
	err := sp.sm.CreateGroup(parentID, name)
	if err != nil {
		return Fail(err)
	}

	return Success()
}

func (sp *ServersProxy) UpdateGroup(groupID, name, parentID string) EmptyResult {
	err := sp.sm.UpdateGroup(groupID, name, parentID)
	if err != nil {
		return Fail(err)
	}

	return Success()
}

// SaveServer saves a new server to the store
func (sp *ServersProxy) SaveServer(parentID, name, uri, colour string) EmptyResult {
	err := sp.sm.AddServer(parentID, name, uri, colour)
	if err != nil {
		return Fail(err)
	}

	return Success()
}

func (sp *ServersProxy) UpdateServer(serverID, name, uri, parentID, colour string) EmptyResult {
	err := sp.sm.UpdateServer(serverID, name, uri, parentID, colour)
	if err != nil {
		return Fail(err)
	}

	return Success()
}

func (sp *ServersProxy) RemoveNode(id string) EmptyResult {
	err := sp.sm.RemoveNode(id)
	if err != nil {
		return Fail(err)
	}
	return Success()
}

func (sp *ServersProxy) SaveServerWithConfig(parentID, name, colour string, cfg models.ConnectionConfig) EmptyResult {
	err := sp.sm.AddServerWithConfig(parentID, name, colour, cfg)
	if err != nil {
		return Fail(err)
	}
	return Success()
}

func (sp *ServersProxy) UpdateServerWithConfig(serverID, name, parentID, colour string, cfg models.ConnectionConfig) EmptyResult {
	err := sp.sm.UpdateServerWithConfig(serverID, name, parentID, colour, cfg)
	if err != nil {
		return Fail(err)
	}
	return Success()
}

func (sp *ServersProxy) GetConnectionConfig(id string) Result[models.ConnectionConfig] {
	cfg, err := sp.sm.GetConnectionConfig(id)
	if err != nil {
		return FailResult[models.ConnectionConfig](err)
	}
	// Strip refresh token — sensitive credential should not reach the frontend
	cfg.RefreshToken = ""
	return SuccessResult(cfg)
}

func (sp *ServersProxy) GetURI(id string) Result[string] {
	uri, err := sp.sm.GetURI(id)
	if err != nil {
		return FailResult[string](err)
	}

	return SuccessResult(uri)
}

func (sp *ServersProxy) ExportServers(serverIDs []string, includeSensitiveData bool) Result[string] {
	data, err := sp.sm.ExportServers(serverIDs, includeSensitiveData)
	if err != nil {
		return FailResult[string](err)
	}
	return SuccessResult(string(data))
}

func (sp *ServersProxy) ImportServers(jsonData string) Result[[]models.RegisteredServer] {
	imported, err := sp.sm.ImportServers([]byte(jsonData))
	if err != nil {
		return FailResult[[]models.RegisteredServer](err)
	}
	return SuccessResult(imported)
}
