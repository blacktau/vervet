package api

import "fmt"

type CollectionsProvider interface {
	GetStatistics(serverID string, dbName string, collectionName string) (map[string]any, error)
	GetDatabaseStatistics(serverID string, dbName string) (map[string]any, error)
}

type CollectionsProxy struct {
	provider CollectionsProvider
}

func NewCollectionsProxy(provider CollectionsProvider) *CollectionsProxy {
	return &CollectionsProxy{provider: provider}
}

func (cp *CollectionsProxy) GetDatabaseStatistics(serverID string, dbName string) Result[map[string]any] {
	result, err := cp.provider.GetDatabaseStatistics(serverID, dbName)
	if err != nil {
		return Result[map[string]any]{
			IsSuccess: false,
			Error:     fmt.Sprintf("Error getting database statistics: %v", err),
		}
	}
	return Result[map[string]any]{
		IsSuccess: true,
		Data:      result,
	}
}

func (cp *CollectionsProxy) GetStatistics(serverID string, dbName string, collectionName string) Result[map[string]any] {
	result, err := cp.provider.GetStatistics(serverID, dbName, collectionName)
	if err != nil {
		return Result[map[string]any]{
			IsSuccess: false,
			Error:     fmt.Sprintf("Error getting collection statistics: %v", err),
		}
	}
	return Result[map[string]any]{
		IsSuccess: true,
		Data:      result,
	}
}
