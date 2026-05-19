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

func (sp *ShellProxy) ExecuteQuery(serverID string, queryID string, dbName string, query string) Result[models.QueryResult] {
	result, err := sp.provider.ExecuteQuery(serverID, queryID, dbName, query)
	if err != nil {
		logFail(sp.log, "ExecuteQuery", err)
		return FailResult[models.QueryResult](err)
	}

	return SuccessResult(result)
}

func (sp *ShellProxy) CancelQuery(serverID string, queryID string) EmptyResult {
	sp.provider.CancelQuery(serverID, queryID)
	return Success()
}

func (sp *ShellProxy) CheckMongosh() Result[bool] {
	return SuccessResult(sp.provider.CheckMongosh())
}

// CountResponse is the payload returned by ShellProxy.CountForPage.
type CountResponse struct {
	Count     int64 `json:"count"`
	Estimated bool  `json:"estimated"`
}

// FetchPage fetches a single page of documents using a previously captured
// PageContext. The frontend calls this when the user navigates the pager.
func (sp *ShellProxy) FetchPage(serverID, dbName string, pc models.PageContext, page, pageSize int64) Result[models.QueryResult] {
	res, err := sp.provider.FetchPage(serverID, dbName, pc, page, pageSize)
	if err != nil {
		logFail(sp.log, "FetchPage", err)
		return FailResult[models.QueryResult](err)
	}
	return SuccessResult(res)
}

// CountForPage returns the total row count for a PageContext. Filter-empty
// queries return an estimated count; filtered queries return an exact count.
func (sp *ShellProxy) CountForPage(serverID, dbName string, pc models.PageContext) Result[CountResponse] {
	count, estimated, err := sp.provider.CountForPage(serverID, dbName, pc)
	if err != nil {
		logFail(sp.log, "CountForPage", err)
		return FailResult[CountResponse](err)
	}
	return SuccessResult(CountResponse{Count: count, Estimated: estimated})
}
