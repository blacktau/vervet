package errcodes_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"

	"vervet/internal/errcodes"
	"vervet/internal/shell"
)

func TestClassifyError_Timeout(t *testing.T) {
	result := errcodes.ClassifyError(context.DeadlineExceeded)
	assert.Equal(t, errcodes.QueryTimeout, result.Code)
}

func TestClassifyError_Cancelled(t *testing.T) {
	result := errcodes.ClassifyError(context.Canceled)
	assert.Equal(t, errcodes.QueryCancelled, result.Code)
}

func TestClassifyError_WrappedTimeout(t *testing.T) {
	wrapped := fmt.Errorf("outer: %w", context.DeadlineExceeded)
	result := errcodes.ClassifyError(wrapped)
	assert.Equal(t, errcodes.QueryTimeout, result.Code)
}

func TestClassifyError_MongoCommandError_Auth(t *testing.T) {
	err := mongo.CommandError{Code: 18, Message: "Authentication failed"}
	result := errcodes.ClassifyError(err)
	assert.Equal(t, errcodes.AuthFailed, result.Code)
}

func TestClassifyError_MongoCommandError_NotAuthorized(t *testing.T) {
	err := mongo.CommandError{Code: 13, Message: "not authorized on db"}
	result := errcodes.ClassifyError(err)
	assert.Equal(t, errcodes.NotAuthorized, result.Code)
}

func TestClassifyError_MongoCommandError_NamespaceNotFound(t *testing.T) {
	err := mongo.CommandError{Code: 26, Message: "ns not found"}
	result := errcodes.ClassifyError(err)
	assert.Equal(t, errcodes.NamespaceNotFound, result.Code)
}

func TestClassifyError_UnknownError(t *testing.T) {
	err := errors.New("something unexpected")
	result := errcodes.ClassifyError(err)
	assert.Equal(t, errcodes.UnknownError, result.Code)
	assert.Equal(t, "something unexpected", result.Detail)
}

func TestClassifyError_NilError(t *testing.T) {
	result := errcodes.ClassifyError(nil)
	assert.Equal(t, errcodes.UnknownError, result.Code)
	assert.Equal(t, "", result.Detail)
}

func TestClassifyError_ShellNotFound(t *testing.T) {
	result := errcodes.ClassifyError(shell.ErrShellNotFound)
	assert.Equal(t, errcodes.ShellNotFound, result.Code)
}

func TestClassifyError_ShellQueryTimeout(t *testing.T) {
	result := errcodes.ClassifyError(shell.ErrQueryTimeout)
	assert.Equal(t, errcodes.QueryTimeout, result.Code)
}

func TestClassifyError_WrappedShellNotFound(t *testing.T) {
	wrapped := fmt.Errorf("execution failed: %w", shell.ErrShellNotFound)
	result := errcodes.ClassifyError(wrapped)
	assert.Equal(t, errcodes.ShellNotFound, result.Code)
}
