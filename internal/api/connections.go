package api

import (
	"context"
	"fmt"
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
	GetDatabases(serverID string) ([]string, error)
	GetCollections(serverID string, dbName string) ([]string, error)
	GetViews(serverID string, dbName string) ([]string, error)
	GetCollectionSchema(serverID string, dbName string, collectionName string) (models.CollectionSchema, error)
	CreateCollection(serverID string, dbName string, collectionName string) error
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
	err := cp.provider.Disconnect(serverID)
	if err != nil {
		return Error(fmt.Sprintf("Error disconnecting from mongo instance: %v", err))
	}

	return Success()
}

func (cp *ConnectionsProxy) DisconnectAll() EmptyResult {
	err := cp.provider.DisconnectAll()
	if err != nil {
		return Error(fmt.Sprintf("Error disconnecting from all connections: %v", err))
	}

	return Success()
}

func (cp *ConnectionsProxy) GetConnections() Result[[]models.Connection] {
	return Result[[]models.Connection]{
		IsSuccess: true,
		Data:      cp.provider.GetConnections(),
	}
}

func (cp *ConnectionsProxy) TestConnection(uri string) EmptyResult {
	_, err := cp.provider.TestConnection(uri)
	if err != nil {
		return Error(fmt.Sprintf("failed to connect to mongo server: %v", err))
	}

	return Success()
}

func (cp *ConnectionsProxy) GetDatabases(serverID string) Result[[]string] {
	result, err := cp.provider.GetDatabases(serverID)
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

func (cp *ConnectionsProxy) GetCollections(serverID string, dbName string) Result[[]string] {
	result, err := cp.provider.GetCollections(serverID, dbName)
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

func (cp *ConnectionsProxy) GetCollectionSchema(serverID string, dbName string, collectionName string) Result[models.CollectionSchema] {
	result, err := cp.provider.GetCollectionSchema(serverID, dbName, collectionName)
	if err != nil {
		return Result[models.CollectionSchema]{
			IsSuccess: false,
			Error:     err.Error(),
		}
	}

	return Result[models.CollectionSchema]{
		IsSuccess: true,
		Data:      result,
	}
}

func (cp *ConnectionsProxy) GetViews(serverID string, dbName string) Result[[]string] {
	result, err := cp.provider.GetViews(serverID, dbName)
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

func (cp *ConnectionsProxy) CreateCollection(serverID string, dbName string, collectionName string) EmptyResult {
	err := cp.provider.CreateCollection(serverID, dbName, collectionName)
	if err != nil {
		return Error(fmt.Sprintf("Error creating collection: %v", err))
	}

	return Success()
}
