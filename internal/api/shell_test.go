package api

import (
	"errors"
	"testing"

	"vervet/internal/models"

	"github.com/stretchr/testify/assert"
)

type MockShellProvider struct {
	executeErr   error
	queryResult  models.QueryResult
	mongoshAvail bool
}

func (m *MockShellProvider) ExecuteQuery(serverID, dbName, query string) (models.QueryResult, error) {
	if m.executeErr != nil {
		return models.QueryResult{}, m.executeErr
	}
	return m.queryResult, nil
}

func (m *MockShellProvider) CancelQuery(serverID string) {}

func (m *MockShellProvider) CheckMongosh() bool {
	return m.mongoshAvail
}

func (m *MockShellProvider) CloseAll() {}

func TestShellProxy_ExecuteQuery(t *testing.T) {
	t.Run("successful query", func(t *testing.T) {
		provider := &MockShellProvider{
			queryResult: models.QueryResult{RawOutput: "ok"},
		}
		proxy := NewShellProxy(provider)
		result := proxy.ExecuteQuery("1", "db1", "db.coll.find()")
		assert.True(t, result.IsSuccess)
		assert.Equal(t, "ok", result.Data.RawOutput)
	})

	t.Run("query error", func(t *testing.T) {
		provider := &MockShellProvider{
			executeErr: errors.New("query failed"),
		}
		proxy := NewShellProxy(provider)
		result := proxy.ExecuteQuery("1", "db1", "db.coll.find()")
		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "query failed")
	})
}

func TestShellProxy_CancelQuery(t *testing.T) {
	t.Run("cancel query", func(t *testing.T) {
		provider := &MockShellProvider{}
		proxy := NewShellProxy(provider)
		result := proxy.CancelQuery("1")
		assert.True(t, result.IsSuccess)
	})
}

func TestShellProxy_CheckMongosh(t *testing.T) {
	t.Run("mongosh available", func(t *testing.T) {
		provider := &MockShellProvider{mongoshAvail: true}
		proxy := NewShellProxy(provider)
		result := proxy.CheckMongosh()
		assert.True(t, result.IsSuccess)
		assert.True(t, result.Data)
	})

	t.Run("mongosh not available", func(t *testing.T) {
		provider := &MockShellProvider{mongoshAvail: false}
		proxy := NewShellProxy(provider)
		result := proxy.CheckMongosh()
		assert.True(t, result.IsSuccess)
		assert.False(t, result.Data)
	})
}
