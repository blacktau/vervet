package api

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
		return FailResult[[]string](err)
	}
	return SuccessResult(result)
}

func (dp *DatabasesProxy) GetDatabaseStatistics(serverID string, dbName string) Result[map[string]any] {
	result, err := dp.provider.GetDatabaseStatistics(serverID, dbName)
	if err != nil {
		return FailResult[map[string]any](err)
	}
	return SuccessResult(result)
}

func (dp *DatabasesProxy) DropDatabase(serverID string, dbName string) EmptyResult {
	err := dp.provider.DropDatabase(serverID, dbName)
	if err != nil {
		return Fail(err)
	}
	return Success()
}
