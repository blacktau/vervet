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

// FailResultWith is FailResult that still hands data to the frontend, for
// failures that come with something worth showing (a script's printed output).
func FailResultWith[T any](data T, err error) Result[T] {
	r := FailResult[T](err)
	r.Data = data
	return r
}

// logFail logs an error from the proxy boundary at Error level.
// Call this once per failed operation before returning FailResult/Fail.
func logFail(log *slog.Logger, op string, err error) {
	if log == nil {
		return
	}
	log.Error(op+" failed", slog.Any("error", err))
}
