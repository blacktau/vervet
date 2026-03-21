package api

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuccess(t *testing.T) {
	result := Success()
	assert.True(t, result.IsSuccess)
	assert.Empty(t, result.ErrorCode)
	assert.Empty(t, result.ErrorDetail)
}

func TestFail(t *testing.T) {
	err := fmt.Errorf("some error")
	result := Fail(err)
	assert.False(t, result.IsSuccess)
	assert.NotEmpty(t, result.ErrorCode)
	assert.NotEmpty(t, result.ErrorDetail)
}

func TestFailResult(t *testing.T) {
	err := fmt.Errorf("bad thing")
	result := FailResult[string](err)
	assert.False(t, result.IsSuccess)
	assert.NotEmpty(t, result.ErrorCode)
	assert.Equal(t, "bad thing", result.ErrorDetail)
	assert.Empty(t, result.Data)
}

func TestSuccessResult(t *testing.T) {
	result := SuccessResult("hello")
	assert.True(t, result.IsSuccess)
	assert.Equal(t, "hello", result.Data)
	assert.Empty(t, result.ErrorCode)
}
