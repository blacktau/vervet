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

func (m *MockShellProvider) ExecuteQuery(serverID, queryID, dbName, query string) (models.QueryResult, error) {
	if m.executeErr != nil {
		return models.QueryResult{}, m.executeErr
	}
	return m.queryResult, nil
}

func (m *MockShellProvider) CancelQuery(serverID, queryID string) {}

func (m *MockShellProvider) CheckMongosh() bool {
	return m.mongoshAvail
}

func (m *MockShellProvider) CloseAll() {}

func (m *MockShellProvider) FetchPage(serverID, dbName string, pc models.PageContext, page, pageSize int64) (models.QueryResult, error) {
	if m.executeErr != nil {
		return models.QueryResult{}, m.executeErr
	}
	return m.queryResult, nil
}

func (m *MockShellProvider) CountForPage(serverID, dbName string, pc models.PageContext) (int64, bool, error) {
	if m.executeErr != nil {
		return 0, false, m.executeErr
	}
	return 0, false, nil
}

func TestShellProxy_ExecuteQuery(t *testing.T) {
	t.Run("successful query", func(t *testing.T) {
		provider := &MockShellProvider{
			queryResult: models.QueryResult{RawOutput: "ok"},
		}
		proxy := NewShellProxy(testLogger(), provider)
		result := proxy.ExecuteQuery("1", "q1", "db1", "db.coll.find()")
		assert.True(t, result.IsSuccess)
		assert.Equal(t, "ok", result.Data.RawOutput)
	})

	t.Run("query error", func(t *testing.T) {
		provider := &MockShellProvider{
			executeErr: errors.New("query failed"),
		}
		proxy := NewShellProxy(testLogger(), provider)
		result := proxy.ExecuteQuery("1", "q1", "db1", "db.coll.find()")
		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestShellProxy_CancelQuery(t *testing.T) {
	t.Run("cancel query", func(t *testing.T) {
		provider := &MockShellProvider{}
		proxy := NewShellProxy(testLogger(), provider)
		result := proxy.CancelQuery("1", "q1")
		assert.True(t, result.IsSuccess)
	})
}

func TestShellProxy_FetchPage(t *testing.T) {
	t.Run("successful page", func(t *testing.T) {
		provider := &MockShellProvider{
			queryResult: models.QueryResult{Documents: []any{map[string]any{"x": int64(1)}}},
		}
		proxy := NewShellProxy(testLogger(), provider)
		result := proxy.FetchPage("1", "db1", models.PageContext{Collection: "c"}, 0, 25)
		assert.True(t, result.IsSuccess)
		assert.Len(t, result.Data.Documents, 1)
	})

	t.Run("error", func(t *testing.T) {
		provider := &MockShellProvider{executeErr: errors.New("boom")}
		proxy := NewShellProxy(testLogger(), provider)
		result := proxy.FetchPage("1", "db1", models.PageContext{Collection: "c"}, 0, 25)
		assert.False(t, result.IsSuccess)
	})
}

func TestShellProxy_CountForPage(t *testing.T) {
	t.Run("successful count", func(t *testing.T) {
		provider := &MockShellProvider{}
		proxy := NewShellProxy(testLogger(), provider)
		result := proxy.CountForPage("1", "db1", models.PageContext{Collection: "c"})
		assert.True(t, result.IsSuccess)
	})

	t.Run("error", func(t *testing.T) {
		provider := &MockShellProvider{executeErr: errors.New("boom")}
		proxy := NewShellProxy(testLogger(), provider)
		result := proxy.CountForPage("1", "db1", models.PageContext{Collection: "c"})
		assert.False(t, result.IsSuccess)
	})
}

func TestShellProxy_CheckMongosh(t *testing.T) {
	t.Run("mongosh available", func(t *testing.T) {
		provider := &MockShellProvider{mongoshAvail: true}
		proxy := NewShellProxy(testLogger(), provider)
		result := proxy.CheckMongosh()
		assert.True(t, result.IsSuccess)
		assert.True(t, result.Data)
	})

	t.Run("mongosh not available", func(t *testing.T) {
		provider := &MockShellProvider{mongoshAvail: false}
		proxy := NewShellProxy(testLogger(), provider)
		result := proxy.CheckMongosh()
		assert.True(t, result.IsSuccess)
		assert.False(t, result.Data)
	})
}
