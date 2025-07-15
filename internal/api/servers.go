package api

import (
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
	err := sp.sm.CreateGroup(name, parentID)
	if err != nil {
		return EmptyResult{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}

	return EmptyResult{
		IsSuccess: true,
	}
}

func (sp *ServersProxy) SaveServer(name string, parentID int, uri string) EmptyResult {
	err := sp.sm.SaveRegisterServer(name, parentID, uri)
	if err != nil {
		return EmptyResult{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}

	return EmptyResult{
		IsSuccess: true,
	}
}
