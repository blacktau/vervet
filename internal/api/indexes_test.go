package api

import (
	"errors"
	"log/slog"
	"testing"

	"vervet/internal/models"

	"github.com/stretchr/testify/assert"
)

type MockIndexesProvider struct {
	getIndexesErr  error
	indexes        []models.Index
	createIndexErr error
	editIndexErr   error
	dropIndexErr   error
}

func (m *MockIndexesProvider) GetIndexes(serverID string, dbName string, collectionName string) ([]models.Index, error) {
	if m.getIndexesErr != nil {
		return nil, m.getIndexesErr
	}
	return m.indexes, nil
}

func (m *MockIndexesProvider) CreateIndex(serverID string, dbName string, collectionName string, request models.CreateIndexRequest) error {
	return m.createIndexErr
}

func (m *MockIndexesProvider) EditIndex(serverID string, dbName string, collectionName string, request models.EditIndexRequest) error {
	return m.editIndexErr
}

func (m *MockIndexesProvider) DropIndex(serverID string, dbName string, collectionName string, indexName string) error {
	return m.dropIndexErr
}

func TestIndexesProxy_GetIndexes(t *testing.T) {
	log := slog.New(slog.Default().Handler())
	t.Run("successful get indexes", func(t *testing.T) {
		provider := &MockIndexesProvider{
			indexes: []models.Index{
				{Name: "_id_", Keys: []models.IndexKeyField{{Field: "_id", Direction: 1}}, Unique: true},
				{Name: "name_1", Keys: []models.IndexKeyField{{Field: "name", Direction: 1}}},
			},
		}
		proxy := NewIndexesProxy(log, provider)
		result := proxy.GetIndexes("1", "db1", "coll1")
		assert.True(t, result.IsSuccess)
		assert.Len(t, result.Data, 2)
		assert.Equal(t, "_id_", result.Data[0].Name)
	})

	t.Run("get indexes error", func(t *testing.T) {
		provider := &MockIndexesProvider{
			getIndexesErr: errors.New("failed to get indexes"),
		}
		proxy := NewIndexesProxy(log, provider)
		result := proxy.GetIndexes("1", "db1", "coll1")
		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestIndexesProxy_CreateIndex(t *testing.T) {
	log := slog.New(slog.Default().Handler())
	t.Run("successful create index", func(t *testing.T) {
		provider := &MockIndexesProvider{}
		proxy := NewIndexesProxy(log, provider)
		result := proxy.CreateIndex("1", "db1", "coll1", models.CreateIndexRequest{
			Keys:   []models.IndexKeyField{{Field: "email", Direction: 1}},
			Unique: true,
		})
		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.ErrorCode)
	})

	t.Run("create index error", func(t *testing.T) {
		provider := &MockIndexesProvider{
			createIndexErr: errors.New("duplicate key"),
		}
		proxy := NewIndexesProxy(log, provider)
		result := proxy.CreateIndex("1", "db1", "coll1", models.CreateIndexRequest{
			Keys: []models.IndexKeyField{{Field: "email", Direction: 1}},
		})
		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestIndexesProxy_EditIndex(t *testing.T) {
	log := slog.New(slog.Default().Handler())
	t.Run("successful edit index", func(t *testing.T) {
		provider := &MockIndexesProvider{}
		proxy := NewIndexesProxy(log, provider)
		result := proxy.EditIndex("1", "db1", "coll1", models.EditIndexRequest{
			OldName: "name_1",
			Keys:    []models.IndexKeyField{{Field: "name", Direction: -1}},
		})
		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.ErrorCode)
	})

	t.Run("edit index error", func(t *testing.T) {
		provider := &MockIndexesProvider{
			editIndexErr: errors.New("edit failed"),
		}
		proxy := NewIndexesProxy(log, provider)
		result := proxy.EditIndex("1", "db1", "coll1", models.EditIndexRequest{
			OldName: "name_1",
			Keys:    []models.IndexKeyField{{Field: "name", Direction: -1}},
		})
		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestIndexesProxy_DropIndex(t *testing.T) {
	log := slog.New(slog.Default().Handler())
	t.Run("successful drop index", func(t *testing.T) {
		provider := &MockIndexesProvider{}
		proxy := NewIndexesProxy(log, provider)
		result := proxy.DropIndex("1", "db1", "coll1", "name_1")
		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.ErrorCode)
	})

	t.Run("drop index error", func(t *testing.T) {
		provider := &MockIndexesProvider{
			dropIndexErr: errors.New("index not found"),
		}
		proxy := NewIndexesProxy(log, provider)
		result := proxy.DropIndex("1", "db1", "coll1", "name_1")
		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}
