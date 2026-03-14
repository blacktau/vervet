package api

import (
	"context"
	"errors"
	"testing"

	"vervet/internal/models"

	"github.com/stretchr/testify/assert"
)

type MockConnectionsProvider struct {
	connection        models.Connection
	connectErr        error
	disconnectErr     error
	testConnErr       error
	getDatabasesErr   error
	getCollectionsErr error
	getViewsErr       error
	connections       []models.Connection
	databases         []string
	collections       []string
	views             []string
	getIndexesErr     error
	indexes           []models.Index
	createIndexErr    error
	dropIndexErr      error
}

func (m *MockConnectionsProvider) Init(ctx context.Context) error {
	return nil
}

func (m *MockConnectionsProvider) Connect(serverID string) (models.Connection, error) {
	if m.connectErr != nil {
		return models.Connection{}, m.connectErr
	}
	return m.connection, nil
}

func (m *MockConnectionsProvider) TestConnection(uri string) (bool, error) {
	if m.testConnErr != nil {
		return false, m.testConnErr
	}
	return true, nil
}

func (m *MockConnectionsProvider) Disconnect(serverID string) error {
	return m.disconnectErr
}

func (m *MockConnectionsProvider) DisconnectAll() error {
	return m.disconnectErr
}

func (m *MockConnectionsProvider) GetConnections() []models.Connection {
	return m.connections
}

func (m *MockConnectionsProvider) GetDatabases(serverID string) ([]string, error) {
	if m.getDatabasesErr != nil {
		return nil, m.getDatabasesErr
	}
	return m.databases, nil
}

func (m *MockConnectionsProvider) GetCollections(serverID string, dbName string) ([]string, error) {
	if m.getCollectionsErr != nil {
		return nil, m.getCollectionsErr
	}
	return m.collections, nil
}

func (m *MockConnectionsProvider) GetViews(serverID string, dbName string) ([]string, error) {
	if m.getViewsErr != nil {
		return nil, m.getViewsErr
	}
	return m.views, nil
}

func (m *MockConnectionsProvider) GetCollectionSchema(serverID string, dbName string, collectionName string) (models.CollectionSchema, error) {
	return models.CollectionSchema{}, nil
}

func (m *MockConnectionsProvider) CreateCollection(serverID string, dbName string, collectionName string) error {
	return nil
}

func (m *MockConnectionsProvider) GetIndexes(serverID string, dbName string, collectionName string) ([]models.Index, error) {
	if m.getIndexesErr != nil {
		return nil, m.getIndexesErr
	}
	return m.indexes, nil
}

func (m *MockConnectionsProvider) CreateIndex(serverID string, dbName string, collectionName string, request models.CreateIndexRequest) error {
	return m.createIndexErr
}

func (m *MockConnectionsProvider) DropIndex(serverID string, dbName string, collectionName string, indexName string) error {
	return m.dropIndexErr
}

func TestConnectionsProxy_Connect(t *testing.T) {
	t.Run("successful connect", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			connection: models.Connection{ServerID: "1", Name: "Server 1"},
		}
		proxy := NewConnectionsProxy(provider, nil)

		result := proxy.Connect("1")

		assert.True(t, result.IsSuccess)
		assert.Equal(t, "Server 1", result.Data.Name)
		assert.Empty(t, result.Error)
	})

	t.Run("connect error", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			connectErr: errors.New("connection failed"),
		}
		proxy := NewConnectionsProxy(provider, nil)

		result := proxy.Connect("1")

		assert.False(t, result.IsSuccess)
		assert.Empty(t, result.Data)
		assert.Contains(t, result.Error, "connection failed")
	})
}

func TestConnectionsProxy_Disconnect(t *testing.T) {
	t.Run("successful disconnect", func(t *testing.T) {
		provider := &MockConnectionsProvider{}
		proxy := NewConnectionsProxy(provider, nil)

		result := proxy.Disconnect("1")

		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.Error)
	})

	t.Run("disconnect error", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			disconnectErr: errors.New("disconnect failed"),
		}
		proxy := NewConnectionsProxy(provider, nil)

		result := proxy.Disconnect("1")

		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "disconnect failed")
	})
}

func TestConnectionsProxy_DisconnectAll(t *testing.T) {
	t.Run("successful disconnect all", func(t *testing.T) {
		provider := &MockConnectionsProvider{}
		proxy := NewConnectionsProxy(provider, nil)

		result := proxy.DisconnectAll()

		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.Error)
	})

	t.Run("disconnect all error", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			disconnectErr: errors.New("disconnect failed"),
		}
		proxy := NewConnectionsProxy(provider, nil)

		result := proxy.DisconnectAll()

		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "disconnect failed")
	})
}

func TestConnectionsProxy_GetConnections(t *testing.T) {
	t.Run("get connections", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			connections: []models.Connection{
				{ServerID: "1", Name: "Server 1"},
				{ServerID: "2", Name: "Server 2"},
			},
		}
		proxy := NewConnectionsProxy(provider, nil)

		result := proxy.GetConnections()

		assert.True(t, result.IsSuccess)
		assert.Len(t, result.Data, 2)
	})
}

func TestConnectionsProxy_TestConnection(t *testing.T) {
	t.Run("successful test connection", func(t *testing.T) {
		provider := &MockConnectionsProvider{}
		proxy := NewConnectionsProxy(provider, nil)

		result := proxy.TestConnection("mongodb://localhost")

		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.Error)
	})

	t.Run("test connection error", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			testConnErr: errors.New("connection failed"),
		}
		proxy := NewConnectionsProxy(provider, nil)

		result := proxy.TestConnection("mongodb://localhost")

		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "connection failed")
	})
}

func TestConnectionsProxy_GetDatabases(t *testing.T) {
	t.Run("successful get databases", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			databases: []string{"db1", "db2"},
		}
		proxy := NewConnectionsProxy(provider, nil)

		result := proxy.GetDatabases("1")

		assert.True(t, result.IsSuccess)
		assert.Len(t, result.Data, 2)
	})

	t.Run("get databases error", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			getDatabasesErr: errors.New("failed to get databases"),
		}
		proxy := NewConnectionsProxy(provider, nil)

		result := proxy.GetDatabases("1")

		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "failed to get databases")
	})
}

func TestConnectionsProxy_GetCollections(t *testing.T) {
	t.Run("successful get collections", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			collections: []string{"coll1", "coll2"},
		}
		proxy := NewConnectionsProxy(provider, nil)

		result := proxy.GetCollections("1", "db1")

		assert.True(t, result.IsSuccess)
		assert.Len(t, result.Data, 2)
	})

	t.Run("get collections error", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			getCollectionsErr: errors.New("failed to get collections"),
		}
		proxy := NewConnectionsProxy(provider, nil)

		result := proxy.GetCollections("1", "db1")

		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "failed to get collections")
	})
}

func TestConnectionsProxy_GetIndexes(t *testing.T) {
	t.Run("successful get indexes", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			indexes: []models.Index{
				{Name: "_id_", Keys: []models.IndexKeyField{{Field: "_id", Direction: 1}}, Unique: true},
				{Name: "name_1", Keys: []models.IndexKeyField{{Field: "name", Direction: 1}}},
			},
		}
		proxy := NewConnectionsProxy(provider, nil)
		result := proxy.GetIndexes("1", "db1", "coll1")
		assert.True(t, result.IsSuccess)
		assert.Len(t, result.Data, 2)
		assert.Equal(t, "_id_", result.Data[0].Name)
	})

	t.Run("get indexes error", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			getIndexesErr: errors.New("failed to get indexes"),
		}
		proxy := NewConnectionsProxy(provider, nil)
		result := proxy.GetIndexes("1", "db1", "coll1")
		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "failed to get indexes")
	})
}

func TestConnectionsProxy_CreateIndex(t *testing.T) {
	t.Run("successful create index", func(t *testing.T) {
		provider := &MockConnectionsProvider{}
		proxy := NewConnectionsProxy(provider, nil)
		result := proxy.CreateIndex("1", "db1", "coll1", models.CreateIndexRequest{
			Keys:   []models.IndexKeyField{{Field: "email", Direction: 1}},
			Unique: true,
		})
		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.Error)
	})

	t.Run("create index error", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			createIndexErr: errors.New("duplicate key"),
		}
		proxy := NewConnectionsProxy(provider, nil)
		result := proxy.CreateIndex("1", "db1", "coll1", models.CreateIndexRequest{
			Keys: []models.IndexKeyField{{Field: "email", Direction: 1}},
		})
		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "duplicate key")
	})
}

func TestConnectionsProxy_DropIndex(t *testing.T) {
	t.Run("successful drop index", func(t *testing.T) {
		provider := &MockConnectionsProvider{}
		proxy := NewConnectionsProxy(provider, nil)
		result := proxy.DropIndex("1", "db1", "coll1", "name_1")
		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.Error)
	})

	t.Run("drop index error", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			dropIndexErr: errors.New("index not found"),
		}
		proxy := NewConnectionsProxy(provider, nil)
		result := proxy.DropIndex("1", "db1", "coll1", "name_1")
		assert.False(t, result.IsSuccess)
		assert.Contains(t, result.Error, "index not found")
	})
}
