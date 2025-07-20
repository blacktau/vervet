package api

import (
	"fmt"
	"vervet/internal/connections"
)

type ConnectionsProxy struct {
	cm *connections.ConnectionManager
}

func NewConnectionsProxy(cm *connections.ConnectionManager) *ConnectionsProxy {
	return &ConnectionsProxy{
		cm: cm,
	}
}

func (cp *ConnectionsProxy) Connect(connectionID int) EmptyResult {
	err := cp.cm.Connect(connectionID)
	if err != nil {
		return Error(fmt.Sprintf("Error connecting to mongo instance: %v", err))
	}
	return Success()
}

func (cp *ConnectionsProxy) Disconnect(connectionID int) EmptyResult {
	err := cp.cm.Disconnect(connectionID)
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

func (cp *ConnectionsProxy) GetConnectionIDs() Result[[]int] {
	return Result[[]int]{
		IsSuccess: true,
		Data:      cp.cm.GetConnectedClientIDs(),
	}
}
