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

func TestClassifyError_DeadlineExceeded_WithServerSelection_IsConnectionFailed(t *testing.T) {
	wrapped := fmt.Errorf("connect failed: server selection error: server selection timeout, current topology: ...: %w", context.DeadlineExceeded)
	result := errcodes.ClassifyError(wrapped)
	assert.Equal(t, errcodes.ConnectionFailed, result.Code)
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

func TestClassifyError_MessageFallback_AuthFailed(t *testing.T) {
	err := errors.New("server selection error: server selection timeout, current topology: { Type: Unknown, Servers: [{ Addr: localhost:27017, Type: Unknown, Last error: connection() error occurred during connection handshake: auth error: sasl conversation error: Authentication failed. }] }")
	result := errcodes.ClassifyError(err)
	assert.Equal(t, errcodes.AuthFailed, result.Code)
}

func TestClassifyError_MessageFallback_ConnectionRefused(t *testing.T) {
	err := errors.New("server selection error: server selection timeout, current topology: { Type: Unknown, Servers: [{ Addr: localhost:27017, Type: Unknown, Last error: connection refused }] }")
	result := errcodes.ClassifyError(err)
	assert.Equal(t, errcodes.ConnectionFailed, result.Code)
}

func TestClassifyError_MessageFallback_ServerSelectionTimeout(t *testing.T) {
	err := errors.New("server selection error: server selection timeout, current topology: { Type: Unknown, Servers: [] }")
	result := errcodes.ClassifyError(err)
	assert.Equal(t, errcodes.ConnectionFailed, result.Code)
}

func TestClassifyError_MessageFallback_NotAuthorized(t *testing.T) {
	err := errors.New("server selection error: not authorized on admin to execute command")
	result := errcodes.ClassifyError(err)
	assert.Equal(t, errcodes.NotAuthorized, result.Code)
}

// timeoutError wraps a message and satisfies mongo.IsTimeout via Timeout() bool.
type timeoutError struct{ msg string }

func (e timeoutError) Error() string { return e.msg }
func (e timeoutError) Timeout() bool { return true }

func TestClassifyError_TimeoutError_WithServerSelection_IsConnectionFailed(t *testing.T) {
	err := timeoutError{msg: "server selection error: server selection timeout, current topology: { Type: Unknown }"}
	result := errcodes.ClassifyError(err)
	assert.Equal(t, errcodes.ConnectionFailed, result.Code)
}
