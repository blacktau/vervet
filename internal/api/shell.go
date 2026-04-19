package api

import (
	"log/slog"

	"vervet/internal/models"
)

type ShellProxy struct {
	log      *slog.Logger
	provider ShellProvider
}

func NewShellProxy(log *slog.Logger, provider ShellProvider) *ShellProxy {
	return &ShellProxy{
		log:      log,
		provider: provider,
	}
}

func (sp *ShellProxy) ExecuteQuery(serverID string, dbName string, query string) Result[models.QueryResult] {
	result, err := sp.provider.ExecuteQuery(serverID, dbName, query)
	if err != nil {
		logFail(sp.log, "ExecuteQuery", err)
		return FailResult[models.QueryResult](err)
	}

	return SuccessResult(result)
}

func (sp *ShellProxy) CancelQuery(serverID string) EmptyResult {
	sp.provider.CancelQuery(serverID)
	return Success()
}

func (sp *ShellProxy) CheckMongosh() Result[bool] {
	return SuccessResult(sp.provider.CheckMongosh())
}
