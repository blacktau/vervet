package api

import (
	"context"
	"fmt"
	"vervet/internal/models"
)

type ConnectionsProxy struct {
	cm ConnectionsProvider
}

type ConnectionsProvider interface {
	Init(ctx context.Context) error
	Connect(serverID string) (models.Connection, error)
	TestConnection(uri string) (bool, error)
	Disconnect(serverID string) error
	DisconnectAll() error
	GetConnectedClientIDs() []string
	GetDatabases(serverID string) ([]string, error)
}

func NewConnectionsProxy(cm ConnectionsProvider) *ConnectionsProxy {
	return &ConnectionsProxy{
		cm: cm,
	}
}

func (cp *ConnectionsProxy) Connect(serverID string) Result[models.Connection] {
	connection, err := cp.cm.Connect(serverID)
	if err != nil {
		return Result[models.Connection]{
			IsSuccess: false,
			Error:     fmt.Sprintf("Error connecting to mongo instance: %v", err),
		}
	}

	return Result[models.Connection]{
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
	_, err := cp.cm.TestConnection(uri)
	if err != nil {
		return Error(fmt.Sprintf("failed to connect to mongo server: %v", err))
	}

	return Success()
}

func (cp *ConnectionsProxy) GetDatabases(serverID string) Result[[]string] {
	result, err := cp.cm.GetDatabases(serverID)
	if err != nil {
		return Result[[]string]{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}

	return Result[[]string]{
		IsSuccess: true,
		Data:      result,
	}
}
