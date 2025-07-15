// Package common holds share types
package common

type Result[T any] struct {
	IsSuccess bool   `json:"isSuccess"`
	Data      T      `json:"data"`
	Error     string `json:"error"`
}

type EmptyResult struct {
	IsSuccess bool   `json:"isSuccess"`
	Error     string `json:"error"`
}
