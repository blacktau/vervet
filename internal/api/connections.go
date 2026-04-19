package api

import (
	"context"
	"log/slog"

	"vervet/internal/models"
)

type ConnectionsProxy struct {
	log      *slog.Logger
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

func NewConnectionsProxy(log *slog.Logger, provider ConnectionsProvider) *ConnectionsProxy {
	return &ConnectionsProxy{
		log:      log,
		provider: provider,
	}
}

func (cp *ConnectionsProxy) Connect(serverID string) Result[models.Connection] {
	connection, err := cp.provider.Connect(serverID)
	if err != nil {
		logFail(cp.log, "Connect", err)
		return FailResult[models.Connection](err)
	}

	return SuccessResult(connection)
}

func (cp *ConnectionsProxy) Disconnect(serverID string) EmptyResult {
	err := cp.provider.Disconnect(serverID)
	if err != nil {
		logFail(cp.log, "Disconnect", err)
		return Fail(err)
	}

	return Success()
}

func (cp *ConnectionsProxy) DisconnectAll() EmptyResult {
	err := cp.provider.DisconnectAll()
	if err != nil {
		logFail(cp.log, "DisconnectAll", err)
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
		logFail(cp.log, "TestConnection", err)
		return Fail(err)
	}

	return Success()
}
