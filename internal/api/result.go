// Package api is the main interface to the frontend
package api

type Result[T any] struct {
	IsSuccess bool   `json:"isSuccess"`
	Data      T      `json:"data"`
	Error     string `json:"error"`
}

type EmptyResult struct {
	IsSuccess bool   `json:"isSuccess"`
	Error     string `json:"error"`
}

func Success() EmptyResult {
	return EmptyResult{
		IsSuccess: true,
	}
}

func Error(message string) EmptyResult {
	return EmptyResult{
		IsSuccess: false,
		Error:     message,
	}
}
