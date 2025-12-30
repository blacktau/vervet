package api

import (
	"fmt"
	"vervet/internal/servers"
)

// ServersProxy exposes the ServerManagerImpl to the UI
// the proxies serve as a place to handle the idiosyncrasies of the marshaling/unmarshalling to the UI
type ServersProxy struct {
	sm servers.Manager
}

func NewServersProxy(sm servers.Manager) *ServersProxy {
	return &ServersProxy{
		sm: sm,
	}
}

func (sp *ServersProxy) GetServers() Result[[]servers.RegisteredServer] {
	registeredServers, err := sp.sm.GetServers()
	if err != nil {
		return Result[[]servers.RegisteredServer]{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}

	return Result[[]servers.RegisteredServer]{
		IsSuccess: true,
		Data:      registeredServers,
	}
}

func (sp *ServersProxy) GetServer(id string) Result[servers.RegisteredServer] {
	registerServer, err := sp.sm.GetServer(id)
	if err != nil {
		return Result[servers.RegisteredServer]{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}

	if registerServer == nil {
		return Result[servers.RegisteredServer]{
			IsSuccess: false,
			Error:     fmt.Sprintf("Server with id %s not found", id),
		}
	}

	return Result[servers.RegisteredServer]{
		IsSuccess: true,
		Data:      *registerServer,
	}
}

func (sp *ServersProxy) CreateGroup(name, parentID string) EmptyResult {
	err := sp.sm.CreateGroup(parentID, name)
	if err != nil {
		return Error(err.Error())
	}

	return Success()
}

func (sp *ServersProxy) UpdateGroup(groupID, name string) EmptyResult {
	err := sp.sm.UpdateGroup(groupID, name)
	if err != nil {
		return Error(err.Error())
	}

	return Success()
}

// SaveServer saves a new server to the store
func (sp *ServersProxy) SaveServer(parentID, name, uri string) EmptyResult {
	err := sp.sm.AddServer(parentID, name, uri)
	if err != nil {
		return Error(err.Error())
	}

	return Success()
}

func (sp *ServersProxy) UpdateServer(serverID, name, uri, parentID string) EmptyResult {
	err := sp.sm.UpdateServer(serverID, name, uri, parentID)
	if err != nil {
		return Error(err.Error())
	}

	return Success()
}

func (sp *ServersProxy) RemoveNode(id string) EmptyResult {
	err := sp.sm.RemoveNode(id)
	if err != nil {
		return Error(fmt.Sprintf("Error removing node: %v", err))
	}
	return Success()
}

func (sp *ServersProxy) GetURI(id string) Result[string] {
	uri, err := sp.sm.GetURI(id)
	if err != nil {
		return Result[string]{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}

	return Result[string]{
		IsSuccess: true,
		Data:      uri,
	}
}
