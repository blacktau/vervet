package api

import "log/slog"

type DatabasesProvider interface {
	GetDatabases(serverID string) ([]string, error)
	GetDatabaseStatistics(serverID string, dbName string) (map[string]any, error)
	DropDatabase(serverID string, dbName string) error
}

type DatabasesProxy struct {
	log      *slog.Logger
	provider DatabasesProvider
}

func NewDatabasesProxy(log *slog.Logger, provider DatabasesProvider) *DatabasesProxy {
	return &DatabasesProxy{log: log, provider: provider}
}

func (dp *DatabasesProxy) GetDatabases(serverID string) Result[[]string] {
	result, err := dp.provider.GetDatabases(serverID)
	if err != nil {
		logFail(dp.log, "GetDatabases", err)
		return FailResult[[]string](err)
	}
	return SuccessResult(result)
}

func (dp *DatabasesProxy) GetDatabaseStatistics(serverID string, dbName string) Result[map[string]any] {
	result, err := dp.provider.GetDatabaseStatistics(serverID, dbName)
	if err != nil {
		logFail(dp.log, "GetDatabaseStatistics", err)
		return FailResult[map[string]any](err)
	}
	return SuccessResult(result)
}

func (dp *DatabasesProxy) DropDatabase(serverID string, dbName string) EmptyResult {
	err := dp.provider.DropDatabase(serverID, dbName)
	if err != nil {
		logFail(dp.log, "DropDatabase", err)
		return Fail(err)
	}
	return Success()
}
