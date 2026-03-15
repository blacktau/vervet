package api

import (
	"errors"
	"testing"

	"vervet/internal/models"

	"github.com/stretchr/testify/assert"
)

type MockCollectionsProvider struct {
	statistics          map[string]any
	getStatisticsErr    error
	getDbStatisticsErr  error
	getServerStatsErr   error
	collections         []string
	getCollectionsErr   error
	views               []string
	getViewsErr         error
	schema              models.CollectionSchema
	getSchemaErr        error
	createCollectionErr error
	renameCollectionErr error
	dropCollectionErr   error
}

func (m *MockCollectionsProvider) GetStatistics(serverID, dbName, collectionName string) (map[string]any, error) {
	if m.getStatisticsErr != nil {
		return nil, m.getStatisticsErr
	}
	return m.statistics, nil
}

func (m *MockCollectionsProvider) GetDatabaseStatistics(serverID, dbName string) (map[string]any, error) {
	if m.getDbStatisticsErr != nil {
		return nil, m.getDbStatisticsErr
	}
	return m.statistics, nil
}

func (m *MockCollectionsProvider) GetServerStatistics(serverID string) (map[string]any, error) {
	if m.getServerStatsErr != nil {
		return nil, m.getServerStatsErr
	}
	return m.statistics, nil
}

func (m *MockCollectionsProvider) GetCollections(serverID, dbName string) ([]string, error) {
	if m.getCollectionsErr != nil {
		return nil, m.getCollectionsErr
	}
	return m.collections, nil
}

func (m *MockCollectionsProvider) GetViews(serverID, dbName string) ([]string, error) {
	if m.getViewsErr != nil {
		return nil, m.getViewsErr
	}
	return m.views, nil
}

func (m *MockCollectionsProvider) GetCollectionSchema(serverID, dbName, collectionName string) (models.CollectionSchema, error) {
	if m.getSchemaErr != nil {
		return models.CollectionSchema{}, m.getSchemaErr
	}
	return m.schema, nil
}

func (m *MockCollectionsProvider) CreateCollection(serverID, dbName, collectionName string) error {
	return m.createCollectionErr
}

func (m *MockCollectionsProvider) RenameCollection(serverID, dbName, oldName, newName string) error {
	return m.renameCollectionErr
}

func (m *MockCollectionsProvider) DropCollection(serverID, dbName, collectionName string) error {
	return m.dropCollectionErr
}

func TestCollectionsProxy_GetCollections(t *testing.T) {
	t.Run("successful get collections", func(t *testing.T) {
		provider := &MockCollectionsProvider{
			collections: []string{"coll1", "coll2"},
		}
		proxy := NewCollectionsProxy(provider)
		result := proxy.GetCollections("1", "db1")
		assert.True(t, result.IsSuccess)
		assert.Len(t, result.Data, 2)
	})

	t.Run("get collections error", func(t *testing.T) {
		provider := &MockCollectionsProvider{
			getCollectionsErr: errors.New("failed"),
		}
		proxy := NewCollectionsProxy(provider)
		result := proxy.GetCollections("1", "db1")
		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "failed")
	})
}

func TestCollectionsProxy_GetViews(t *testing.T) {
	t.Run("successful get views", func(t *testing.T) {
		provider := &MockCollectionsProvider{
			views: []string{"view1"},
		}
		proxy := NewCollectionsProxy(provider)
		result := proxy.GetViews("1", "db1")
		assert.True(t, result.IsSuccess)
		assert.Len(t, result.Data, 1)
	})

	t.Run("get views error", func(t *testing.T) {
		provider := &MockCollectionsProvider{
			getViewsErr: errors.New("failed"),
		}
		proxy := NewCollectionsProxy(provider)
		result := proxy.GetViews("1", "db1")
		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "failed")
	})
}

func TestCollectionsProxy_CreateCollection(t *testing.T) {
	t.Run("successful create", func(t *testing.T) {
		provider := &MockCollectionsProvider{}
		proxy := NewCollectionsProxy(provider)
		result := proxy.CreateCollection("1", "db1", "newcoll")
		assert.True(t, result.IsSuccess)
	})

	t.Run("create error", func(t *testing.T) {
		provider := &MockCollectionsProvider{
			createCollectionErr: errors.New("already exists"),
		}
		proxy := NewCollectionsProxy(provider)
		result := proxy.CreateCollection("1", "db1", "newcoll")
		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "already exists")
	})
}

func TestCollectionsProxy_RenameCollection(t *testing.T) {
	t.Run("successful rename", func(t *testing.T) {
		provider := &MockCollectionsProvider{}
		proxy := NewCollectionsProxy(provider)
		result := proxy.RenameCollection("1", "db1", "old", "new")
		assert.True(t, result.IsSuccess)
	})

	t.Run("rename error", func(t *testing.T) {
		provider := &MockCollectionsProvider{
			renameCollectionErr: errors.New("target exists"),
		}
		proxy := NewCollectionsProxy(provider)
		result := proxy.RenameCollection("1", "db1", "old", "new")
		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "target exists")
	})
}

func TestCollectionsProxy_DropCollection(t *testing.T) {
	t.Run("successful drop", func(t *testing.T) {
		provider := &MockCollectionsProvider{}
		proxy := NewCollectionsProxy(provider)
		result := proxy.DropCollection("1", "db1", "coll1")
		assert.True(t, result.IsSuccess)
	})

	t.Run("drop error", func(t *testing.T) {
		provider := &MockCollectionsProvider{
			dropCollectionErr: errors.New("ns not found"),
		}
		proxy := NewCollectionsProxy(provider)
		result := proxy.DropCollection("1", "db1", "coll1")
		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "ns not found")
	})
}

func TestCollectionsProxy_GetCollectionSchema(t *testing.T) {
	t.Run("successful get schema", func(t *testing.T) {
		provider := &MockCollectionsProvider{
			schema: models.CollectionSchema{},
		}
		proxy := NewCollectionsProxy(provider)
		result := proxy.GetCollectionSchema("1", "db1", "coll1")
		assert.True(t, result.IsSuccess)
	})

	t.Run("get schema error", func(t *testing.T) {
		provider := &MockCollectionsProvider{
			getSchemaErr: errors.New("failed"),
		}
		proxy := NewCollectionsProxy(provider)
		result := proxy.GetCollectionSchema("1", "db1", "coll1")
		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "failed")
	})
}
