package api

import (
	"errors"
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
	t.Run("successful get indexes", func(t *testing.T) {
		provider := &MockIndexesProvider{
			indexes: []models.Index{
				{Name: "_id_", Keys: []models.IndexKeyField{{Field: "_id", Direction: 1}}, Unique: true},
				{Name: "name_1", Keys: []models.IndexKeyField{{Field: "name", Direction: 1}}},
			},
		}
		proxy := NewIndexesProxy(provider)
		result := proxy.GetIndexes("1", "db1", "coll1")
		assert.True(t, result.IsSuccess)
		assert.Len(t, result.Data, 2)
		assert.Equal(t, "_id_", result.Data[0].Name)
	})

	t.Run("get indexes error", func(t *testing.T) {
		provider := &MockIndexesProvider{
			getIndexesErr: errors.New("failed to get indexes"),
		}
		proxy := NewIndexesProxy(provider)
		result := proxy.GetIndexes("1", "db1", "coll1")
		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "failed to get indexes")
	})
}

func TestIndexesProxy_CreateIndex(t *testing.T) {
	t.Run("successful create index", func(t *testing.T) {
		provider := &MockIndexesProvider{}
		proxy := NewIndexesProxy(provider)
		result := proxy.CreateIndex("1", "db1", "coll1", models.CreateIndexRequest{
			Keys:   []models.IndexKeyField{{Field: "email", Direction: 1}},
			Unique: true,
		})
		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.Error)
	})

	t.Run("create index error", func(t *testing.T) {
		provider := &MockIndexesProvider{
			createIndexErr: errors.New("duplicate key"),
		}
		proxy := NewIndexesProxy(provider)
		result := proxy.CreateIndex("1", "db1", "coll1", models.CreateIndexRequest{
			Keys: []models.IndexKeyField{{Field: "email", Direction: 1}},
		})
		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "duplicate key")
	})
}

func TestIndexesProxy_EditIndex(t *testing.T) {
	t.Run("successful edit index", func(t *testing.T) {
		provider := &MockIndexesProvider{}
		proxy := NewIndexesProxy(provider)
		result := proxy.EditIndex("1", "db1", "coll1", models.EditIndexRequest{
			OldName: "name_1",
			Keys:    []models.IndexKeyField{{Field: "name", Direction: -1}},
		})
		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.Error)
	})

	t.Run("edit index error", func(t *testing.T) {
		provider := &MockIndexesProvider{
			editIndexErr: errors.New("edit failed"),
		}
		proxy := NewIndexesProxy(provider)
		result := proxy.EditIndex("1", "db1", "coll1", models.EditIndexRequest{
			OldName: "name_1",
			Keys:    []models.IndexKeyField{{Field: "name", Direction: -1}},
		})
		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "edit failed")
	})
}

func TestIndexesProxy_DropIndex(t *testing.T) {
	t.Run("successful drop index", func(t *testing.T) {
		provider := &MockIndexesProvider{}
		proxy := NewIndexesProxy(provider)
		result := proxy.DropIndex("1", "db1", "coll1", "name_1")
		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.Error)
	})

	t.Run("drop index error", func(t *testing.T) {
		provider := &MockIndexesProvider{
			dropIndexErr: errors.New("index not found"),
		}
		proxy := NewIndexesProxy(provider)
		result := proxy.DropIndex("1", "db1", "coll1", "name_1")
		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "index not found")
	})
}
