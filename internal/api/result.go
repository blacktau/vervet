package api

import (
	"log/slog"

	"vervet/internal/errcodes"
)

type Result[T any] struct {
	IsSuccess   bool   `json:"isSuccess"`
	Data        T      `json:"data"`
	ErrorCode   string `json:"errorCode,omitempty"`
	ErrorDetail string `json:"errorDetail,omitempty"`
}

type EmptyResult struct {
	IsSuccess   bool   `json:"isSuccess"`
	ErrorCode   string `json:"errorCode,omitempty"`
	ErrorDetail string `json:"errorDetail,omitempty"`
}

func Success() EmptyResult {
	return EmptyResult{IsSuccess: true}
}

func Fail(err error) EmptyResult {
	classified := errcodes.ClassifyError(err)
	return EmptyResult{
		IsSuccess:   false,
		ErrorCode:   classified.Code,
		ErrorDetail: classified.Detail,
	}
}

func SuccessResult[T any](data T) Result[T] {
	return Result[T]{IsSuccess: true, Data: data}
}

func FailResult[T any](err error) Result[T] {
	classified := errcodes.ClassifyError(err)
	return Result[T]{
		IsSuccess:   false,
		ErrorCode:   classified.Code,
		ErrorDetail: classified.Detail,
	}
}

// logFail logs an error from the proxy boundary at Error level.
// Call this once per failed operation before returning FailResult/Fail.
func logFail(log *slog.Logger, op string, err error) {
	if log == nil {
		return
	}
	log.Error(op+" failed", slog.Any("error", err))
}
