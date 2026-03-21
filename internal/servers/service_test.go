package servers

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"vervet/internal/connectionStrings"
	"vervet/internal/models"

	"github.com/stretchr/testify/assert"
)

// mockServerStore is a mock implementation of serverStore for testing.
type mockServerStore struct {
	servers []models.RegisteredServer
	err     error
}

func (m *mockServerStore) LoadServers() ([]models.RegisteredServer, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.servers, nil
}

func (m *mockServerStore) SaveServers(servers []models.RegisteredServer) error {
	if m.err != nil {
		return m.err
	}
	m.servers = servers
	return nil
}

// MockConnectionStringsStore is a mock implementation of ConnectionStringsStore for testing.
type MockConnectionStringsStore struct {
	uris map[string]string
	err  error
}

func (m *MockConnectionStringsStore) StoreRegisteredServerURI(id, uri string) error {
	if m.err != nil {
		return m.err
	}
	m.uris[id] = uri
	return nil
}

func (m *MockConnectionStringsStore) GetRegisteredServerURI(id string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	uri, ok := m.uris[id]
	if !ok {
		return "", errors.New("URI not found")
	}
	return uri, nil
}

func (m *MockConnectionStringsStore) DeleteRegisteredServerURI(id string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.uris, id)
	return nil
}

func (m *MockConnectionStringsStore) StoreConnectionConfig(id string, cfg models.ConnectionConfig) error {
	if m.err != nil {
		return m.err
	}
	m.uris[id] = cfg.URI
	return nil
}

func (m *MockConnectionStringsStore) GetConnectionConfig(id string) (models.ConnectionConfig, error) {
	if m.err != nil {
		return models.ConnectionConfig{}, m.err
	}
	uri, ok := m.uris[id]
	if !ok {
		return models.ConnectionConfig{}, errors.New("config not found")
	}
	return models.ConnectionConfig{URI: uri, AuthMethod: models.AuthPassword}, nil
}

func (m *MockConnectionStringsStore) UpdateRefreshToken(id string, refreshToken string) error {
	return m.err
}

func newTestServerService(store ServerStore, connectionStringsStore connectionStrings.Store) *ServerService {
	log := slog.Default()
	return &ServerService{
		log:               log,
		mu:                sync.RWMutex{},
		connectionStrings: connectionStringsStore,
		ctx:               context.Background(),
		store:             store,
	}
}

func TestGetServers(t *testing.T) {
	// Test case 1: Successful get servers
	t.Run("successful get servers", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Server 1"},
				{ID: "2", Name: "Server 2"},
			},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		servers, err := sm.GetServers()

		assert.NoError(t, err)
		assert.Len(t, servers, 2)
	})

	// Test case 2: Store returns an error
	t.Run("store error", func(t *testing.T) {
		mockStore := &mockServerStore{
			err: errors.New("store error"),
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		servers, err := sm.GetServers()

		assert.Error(t, err)
		assert.Nil(t, servers)
	})
}

func TestAddServer(t *testing.T) {
	t.Run("successful add", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "parent", Name: "Parent Group", IsGroup: true},
			},
		}
		mockCSStore := &MockConnectionStringsStore{
			uris: make(map[string]string),
		}
		sm := newTestServerService(mockStore, mockCSStore)

		err := sm.AddServer("parent", "New Server", "mongodb://localhost", "")

		assert.NoError(t, err)
		assert.Len(t, mockStore.servers, 2)
		assert.Len(t, mockCSStore.uris, 1)
	})
}

func TestUpdateServer(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Server 1"},
			},
		}
		mockCSStore := &MockConnectionStringsStore{
			uris: map[string]string{"1": "mongodb://localhost"},
		}
		sm := newTestServerService(mockStore, mockCSStore)

		err := sm.UpdateServer("1", "Updated Server", "mongodb://newhost", "a", "")

		assert.NoError(t, err)
		assert.Equal(t, "Updated Server", mockStore.servers[0].Name)
		assert.Equal(t, "mongodb://newhost", mockCSStore.uris["1"])
	})

	t.Run("server not found", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		err := sm.UpdateServer("1", "Updated Server", "mongodb://newhost", "", "")

		assert.Error(t, err)
	})
}

func TestRemoveNode(t *testing.T) {
	t.Run("successful remove server", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Server 1"},
			},
		}
		mockCSStore := &MockConnectionStringsStore{
			uris: map[string]string{"1": "mongodb://localhost"},
		}
		sm := newTestServerService(mockStore, mockCSStore)

		err := sm.RemoveNode("1")

		assert.NoError(t, err)
		assert.Len(t, mockStore.servers, 0)
		assert.Len(t, mockCSStore.uris, 0)
	})

	t.Run("successful remove group", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Group 1", IsGroup: true},
			},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		err := sm.RemoveNode("1")

		assert.NoError(t, err)
		assert.Len(t, mockStore.servers, 0)
	})

	t.Run("node not found", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		err := sm.RemoveNode("1")

		assert.Error(t, err)
	})

	t.Run("cannot remove group with children", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Group 1", IsGroup: true},
				{ID: "2", Name: "Server 2", ParentID: "1"},
			},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		err := sm.RemoveNode("1")

		assert.Error(t, err)
	})
}

func TestGetURI(t *testing.T) {
	t.Run("successful get uri", func(t *testing.T) {
		mockCSStore := &MockConnectionStringsStore{
			uris: map[string]string{"1": "mongodb://localhost"},
		}
		sm := newTestServerService(&mockServerStore{}, mockCSStore)

		uri, err := sm.GetURI("1")

		assert.NoError(t, err)
		assert.Equal(t, "mongodb://localhost", uri)
	})

	t.Run("uri not found", func(t *testing.T) {
		mockCSStore := &MockConnectionStringsStore{
			uris: make(map[string]string),
		}
		sm := newTestServerService(&mockServerStore{}, mockCSStore)

		uri, err := sm.GetURI("1")

		assert.Error(t, err)
		assert.Empty(t, uri)
	})
}
