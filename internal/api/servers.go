package api

import (
	"fmt"
	"vervet/internal/configuration"
	"vervet/internal/servers"
)

// ServersProxy exposes the REgisteredServerManagaer to the UI
// the proxies serve as a place to handle the idosyncrasies of the mashalling/unmarshalling to the UI
type ServersProxy struct {
	sm *servers.ServerManager
}

func NewServersProxy(sm *servers.ServerManager) *ServersProxy {
	return &ServersProxy{
		sm: sm,
	}
}

func (sp *ServersProxy) GetServers() Result[[]configuration.RegisteredServer] {
	registeredServers, err := sp.sm.GetRegisteredServers()
	if err != nil {
		return Result[[]configuration.RegisteredServer]{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}

	return Result[[]configuration.RegisteredServer]{
		IsSuccess: true,
		Data:      registeredServers,
	}
}

func (sp *ServersProxy) CreateGroup(name string, parentID int) EmptyResult {
	err := sp.sm.CreateGroup(parentID, name)
	if err != nil {
		return Error(err.Error())
	}

	return Success()
}

func (sp *ServersProxy) UpdateGroup(groupID, parentID int, name string) EmptyResult {
	err := sp.sm.UpdateGroup(groupID, parentID, name)
	if err != nil {
		return Error(err.Error())
	}

	return Success()
}

func (sp *ServersProxy) SaveServer(parentID int, name, uri string) EmptyResult {
	err := sp.sm.AddRegisterServer(parentID, name, uri)
	if err != nil {
		return Error(err.Error())
	}

	return Success()
}

func (sp *ServersProxy) UpdateServer(serverID, parentID int, name, uri string) EmptyResult {
	err := sp.sm.UpdateRegisterServer(serverID, parentID, name, uri)
	if err != nil {
		return Error(err.Error())
	}

	return Success()
}

func (sp *ServersProxy) RemoveNode(id int, isFolder bool) EmptyResult {
	err := sp.sm.RemoveNode(id, isFolder)
	if err != nil {
		return Error(fmt.Sprintf("Error removing node: %v", err))
	}
	return Success()
}

func (sp *ServersProxy) GetURI(id int) Result[string] {
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
