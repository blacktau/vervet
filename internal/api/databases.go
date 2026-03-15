package api

import "fmt"

type DatabasesProvider interface {
	GetDatabases(serverID string) ([]string, error)
	GetDatabaseStatistics(serverID string, dbName string) (map[string]any, error)
	DropDatabase(serverID string, dbName string) error
}

type DatabasesProxy struct {
	provider DatabasesProvider
}

func NewDatabasesProxy(provider DatabasesProvider) *DatabasesProxy {
	return &DatabasesProxy{provider: provider}
}

func (dp *DatabasesProxy) GetDatabases(serverID string) Result[[]string] {
	result, err := dp.provider.GetDatabases(serverID)
	if err != nil {
		return Result[[]string]{
			IsSuccess: false,
			Error:     fmt.Sprintf("Error getting databases: %v", err),
		}
	}
	return Result[[]string]{
		IsSuccess: true,
		Data:      result,
	}
}

func (dp *DatabasesProxy) GetDatabaseStatistics(serverID string, dbName string) Result[map[string]any] {
	result, err := dp.provider.GetDatabaseStatistics(serverID, dbName)
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

func (dp *DatabasesProxy) DropDatabase(serverID string, dbName string) EmptyResult {
	err := dp.provider.DropDatabase(serverID, dbName)
	if err != nil {
		return Error(fmt.Sprintf("Error dropping database: %v", err))
	}
	return Success()
}
