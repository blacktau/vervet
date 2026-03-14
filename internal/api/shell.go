package api

import (
	"fmt"
	"vervet/internal/models"
)

type ShellProxy struct {
	provider ShellProvider
}

func NewShellProxy(provider ShellProvider) *ShellProxy {
	return &ShellProxy{
		provider: provider,
	}
}

func (sp *ShellProxy) ExecuteQuery(serverID string, dbName string, query string) Result[models.QueryResult] {
	result, err := sp.provider.ExecuteQuery(serverID, dbName, query)
	if err != nil {
		return Result[models.QueryResult]{
			IsSuccess: false,
			Error:     fmt.Sprintf("Query execution failed: %v", err),
		}
	}

	return Result[models.QueryResult]{
		IsSuccess: true,
		Data:      result,
	}
}

func (sp *ShellProxy) CancelQuery(serverID string) EmptyResult {
	sp.provider.CancelQuery(serverID)
	return Success()
}

func (sp *ShellProxy) CheckMongosh() Result[bool] {
	return Result[bool]{
		IsSuccess: true,
		Data:      sp.provider.CheckMongosh(),
	}
}
