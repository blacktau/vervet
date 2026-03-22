package api

import (
	"vervet/internal/models"
)

type IndexesProvider interface {
	GetIndexes(serverID string, dbName string, collectionName string) ([]models.Index, error)
	CreateIndex(serverID string, dbName string, collectionName string, request models.CreateIndexRequest) error
	EditIndex(serverID string, dbName string, collectionName string, request models.EditIndexRequest) error
	DropIndex(serverID string, dbName string, collectionName string, indexName string) error
}

type IndexesProxy struct {
	provider IndexesProvider
}

func NewIndexesProxy(provider IndexesProvider) *IndexesProxy {
	return &IndexesProxy{provider: provider}
}

func (ip *IndexesProxy) GetIndexes(serverID string, dbName string, collectionName string) Result[[]models.Index] {
	result, err := ip.provider.GetIndexes(serverID, dbName, collectionName)
	if err != nil {
		return FailResult[[]models.Index](err)
	}
	return SuccessResult(result)
}

func (ip *IndexesProxy) CreateIndex(serverID string, dbName string, collectionName string, request models.CreateIndexRequest) EmptyResult {
	err := ip.provider.CreateIndex(serverID, dbName, collectionName, request)
	if err != nil {
		return Fail(err)
	}
	return Success()
}

func (ip *IndexesProxy) EditIndex(serverID string, dbName string, collectionName string, request models.EditIndexRequest) EmptyResult {
	err := ip.provider.EditIndex(serverID, dbName, collectionName, request)
	if err != nil {
		return Fail(err)
	}
	return Success()
}

func (ip *IndexesProxy) DropIndex(serverID string, dbName string, collectionName string, indexName string) EmptyResult {
	err := ip.provider.DropIndex(serverID, dbName, collectionName, indexName)
	if err != nil {
		return Fail(err)
	}
	return Success()
}
