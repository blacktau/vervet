package api

import (
	"log/slog"

	"vervet/internal/models"
)

type CollectionsProvider interface {
	GetStatistics(serverID string, dbName string, collectionName string) (map[string]any, error)
	GetServerStatistics(serverID string) (map[string]any, error)
	GetCollections(serverID string, dbName string) ([]string, error)
	GetViews(serverID string, dbName string) ([]string, error)
	GetCollectionSchema(serverID string, dbName string, collectionName string) (models.CollectionSchema, error)
	CreateCollection(serverID string, dbName string, collectionName string) error
	RenameCollection(serverID string, dbName string, oldName string, newName string) error
	DropCollection(serverID string, dbName string, collectionName string) error
}

type CollectionsProxy struct {
	log      *slog.Logger
	provider CollectionsProvider
}

func NewCollectionsProxy(log *slog.Logger, provider CollectionsProvider) *CollectionsProxy {
	return &CollectionsProxy{log: log, provider: provider}
}

func (cp *CollectionsProxy) GetServerStatistics(serverID string) Result[map[string]any] {
	result, err := cp.provider.GetServerStatistics(serverID)
	if err != nil {
		logFail(cp.log, "GetServerStatistics", err)
		return FailResult[map[string]any](err)
	}
	return SuccessResult(result)
}

func (cp *CollectionsProxy) GetStatistics(serverID string, dbName string, collectionName string) Result[map[string]any] {
	result, err := cp.provider.GetStatistics(serverID, dbName, collectionName)
	if err != nil {
		logFail(cp.log, "GetStatistics", err)
		return FailResult[map[string]any](err)
	}
	return SuccessResult(result)
}

func (cp *CollectionsProxy) GetCollections(serverID string, dbName string) Result[[]string] {
	result, err := cp.provider.GetCollections(serverID, dbName)
	if err != nil {
		logFail(cp.log, "GetCollections", err)
		return FailResult[[]string](err)
	}
	return SuccessResult(result)
}

func (cp *CollectionsProxy) GetViews(serverID string, dbName string) Result[[]string] {
	result, err := cp.provider.GetViews(serverID, dbName)
	if err != nil {
		logFail(cp.log, "GetViews", err)
		return FailResult[[]string](err)
	}
	return SuccessResult(result)
}

func (cp *CollectionsProxy) GetCollectionSchema(serverID string, dbName string, collectionName string) Result[models.CollectionSchema] {
	result, err := cp.provider.GetCollectionSchema(serverID, dbName, collectionName)
	if err != nil {
		logFail(cp.log, "GetCollectionSchema", err)
		return FailResult[models.CollectionSchema](err)
	}
	return SuccessResult(result)
}

func (cp *CollectionsProxy) CreateCollection(serverID string, dbName string, collectionName string) EmptyResult {
	err := cp.provider.CreateCollection(serverID, dbName, collectionName)
	if err != nil {
		logFail(cp.log, "CreateCollection", err)
		return Fail(err)
	}
	return Success()
}

func (cp *CollectionsProxy) RenameCollection(serverID string, dbName string, oldName string, newName string) EmptyResult {
	err := cp.provider.RenameCollection(serverID, dbName, oldName, newName)
	if err != nil {
		logFail(cp.log, "RenameCollection", err)
		return Fail(err)
	}
	return Success()
}

func (cp *CollectionsProxy) DropCollection(serverID string, dbName string, collectionName string) EmptyResult {
	err := cp.provider.DropCollection(serverID, dbName, collectionName)
	if err != nil {
		logFail(cp.log, "DropCollection", err)
		return Fail(err)
	}
	return Success()
}
