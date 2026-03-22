package api

import (
	"context"
	"vervet/internal/models"
)

type ConnectionsProxy struct {
	provider ConnectionsProvider
}

type ConnectionsProvider interface {
	Init(ctx context.Context) error
	Connect(serverID string) (models.Connection, error)
	TestConnection(uri string) (bool, error)
	Disconnect(serverID string) error
	DisconnectAll() error
	GetConnections() []models.Connection
}

type ShellProvider interface {
	ExecuteQuery(serverID, dbName, query string) (models.QueryResult, error)
	CancelQuery(serverID string)
	CheckMongosh() bool
	CloseAll()
}

func NewConnectionsProxy(provider ConnectionsProvider) *ConnectionsProxy {
	return &ConnectionsProxy{
		provider: provider,
	}
}

func (cp *ConnectionsProxy) Connect(serverID string) Result[models.Connection] {
	connection, err := cp.provider.Connect(serverID)
	if err != nil {
		return FailResult[models.Connection](err)
	}

	return SuccessResult(connection)
}

func (cp *ConnectionsProxy) Disconnect(serverID string) EmptyResult {
	err := cp.provider.Disconnect(serverID)
	if err != nil {
		return Fail(err)
	}

	return Success()
}

func (cp *ConnectionsProxy) DisconnectAll() EmptyResult {
	err := cp.provider.DisconnectAll()
	if err != nil {
		return Fail(err)
	}

	return Success()
}

func (cp *ConnectionsProxy) GetConnections() Result[[]models.Connection] {
	return SuccessResult(cp.provider.GetConnections())
}

func (cp *ConnectionsProxy) TestConnection(uri string) EmptyResult {
	_, err := cp.provider.TestConnection(uri)
	if err != nil {
		return Fail(err)
	}

	return Success()
}
