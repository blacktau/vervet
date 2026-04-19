package api

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockDatabasesProvider struct {
	databases           []string
	getDatabasesErr     error
	dbStats             map[string]any
	getDatabaseStatsErr error
	dropDatabaseErr     error
}

func (m *MockDatabasesProvider) GetDatabases(serverID string) ([]string, error) {
	if m.getDatabasesErr != nil {
		return nil, m.getDatabasesErr
	}
	return m.databases, nil
}

func (m *MockDatabasesProvider) GetDatabaseStatistics(serverID string, dbName string) (map[string]any, error) {
	if m.getDatabaseStatsErr != nil {
		return nil, m.getDatabaseStatsErr
	}
	return m.dbStats, nil
}

func (m *MockDatabasesProvider) DropDatabase(serverID string, dbName string) error {
	return m.dropDatabaseErr
}

func TestDatabasesProxy_GetDatabases(t *testing.T) {
	log := slog.Default()
	t.Run("successful get databases", func(t *testing.T) {
		provider := &MockDatabasesProvider{
			databases: []string{"db1", "db2"},
		}
		proxy := NewDatabasesProxy(log, provider)

		result := proxy.GetDatabases("1")

		assert.True(t, result.IsSuccess)
		assert.Len(t, result.Data, 2)
	})

	t.Run("get databases error", func(t *testing.T) {
		provider := &MockDatabasesProvider{
			getDatabasesErr: errors.New("failed to get databases"),
		}
		proxy := NewDatabasesProxy(log, provider)

		result := proxy.GetDatabases("1")

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestDatabasesProxy_GetDatabaseStatistics(t *testing.T) {
	log := slog.Default()
	t.Run("successful get database statistics", func(t *testing.T) {
		provider := &MockDatabasesProvider{
			dbStats: map[string]any{"db": "testdb", "collections": 3},
		}
		proxy := NewDatabasesProxy(log, provider)

		result := proxy.GetDatabaseStatistics("1", "testdb")

		assert.True(t, result.IsSuccess)
		assert.Equal(t, "testdb", result.Data["db"])
	})

	t.Run("get database statistics error", func(t *testing.T) {
		provider := &MockDatabasesProvider{
			getDatabaseStatsErr: errors.New("stats error"),
		}
		proxy := NewDatabasesProxy(log, provider)

		result := proxy.GetDatabaseStatistics("1", "testdb")

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestDatabasesProxy_DropDatabase(t *testing.T) {
	log := slog.Default()
	t.Run("successful drop database", func(t *testing.T) {
		provider := &MockDatabasesProvider{}
		proxy := NewDatabasesProxy(log, provider)

		result := proxy.DropDatabase("1", "testdb")

		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.ErrorCode)
	})

	t.Run("drop database error", func(t *testing.T) {
		provider := &MockDatabasesProvider{
			dropDatabaseErr: errors.New("drop failed"),
		}
		proxy := NewDatabasesProxy(log, provider)

		result := proxy.DropDatabase("1", "testdb")

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}
