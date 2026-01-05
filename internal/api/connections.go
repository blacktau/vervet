package api

import (
	"fmt"
	"vervet/internal/connections"
)

type ConnectionsProxy struct {
	cm connections.Manager
}

func NewConnectionsProxy(cm connections.Manager) *ConnectionsProxy {
	return &ConnectionsProxy{
		cm: cm,
	}
}

func (cp *ConnectionsProxy) Connect(serverID string) Result[connections.Connection] {
	connection, err := cp.cm.Connect(serverID)
	if err != nil {
		return Result[connections.Connection]{
			IsSuccess: false,
			Error:     fmt.Sprintf("Error connecting to mongo instance: %v", err),
		}
	}

	return Result[connections.Connection]{
		IsSuccess: true,
		Data:      connection,
	}
}

func (cp *ConnectionsProxy) Disconnect(serverID string) EmptyResult {
	err := cp.cm.Disconnect(serverID)
	if err != nil {
		return Error(fmt.Sprintf("Error disconnecting from mongo instance: %v", err))
	}

	return Success()
}

func (cp *ConnectionsProxy) DisconnectAll() EmptyResult {
	err := cp.cm.DisconnectAll()
	if err != nil {
		return Error(fmt.Sprintf("Error disconnecting from all connections: %v", err))
	}

	return Success()
}

func (cp *ConnectionsProxy) GetConnectionIDs() Result[[]string] {
	return Result[[]string]{
		IsSuccess: true,
		Data:      cp.cm.GetConnectedClientIDs(),
	}
}

func (cp *ConnectionsProxy) TestConnection(uri string) EmptyResult {
	result, err := cp.cm.TestConnection(uri)
	return EmptyResult{
		IsSuccess: result,
		Error:     fmt.Sprintf("failed to connect to mongo server: %v", err),
	}
}

func (cp *ConnectionsProxy) GetDatabases(serverID string) Result[[]string] {
	result, err := cp.cm.GetDatabases(serverID)
	return Result[[]string]{
		IsSuccess: err == nil,
		Data:      result,
		Error:     err.Error(),
	}
}
