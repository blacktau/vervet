package servers

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"vervet/internal/connectionStrings"

	"github.com/stretchr/testify/assert"
)

// MockServerStore is a mock implementation of ServerStore for testing.
type MockServerStore struct {
	servers []RegisteredServer
	err     error
}

func (m *MockServerStore) LoadServers() ([]RegisteredServer, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.servers, nil
}

func (m *MockServerStore) SaveServers(servers []RegisteredServer) error {
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

func newTestServerManager(serverStore ServerStore, connectionStringsStore connectionStrings.Store) *ServerManager {
	log := slog.Default()
	return &ServerManager{
		log:               log,
		mu:                sync.RWMutex{},
		connectionStrings: connectionStringsStore,
		ctx:               context.Background(),
		store:             serverStore,
	}
}

func TestGetServers(t *testing.T) {
	// Test case 1: Successful get servers
	t.Run("successful get servers", func(t *testing.T) {
		mockStore := &MockServerStore{
			servers: []RegisteredServer{
				{ID: "1", Name: "Server 1"},
				{ID: "2", Name: "Server 2"},
			},
		}
		sm := newTestServerManager(mockStore, &MockConnectionStringsStore{})

		servers, err := sm.GetServers()

		assert.NoError(t, err)
		assert.Len(t, servers, 2)
	})

	// Test case 2: Store returns an error
	t.Run("store error", func(t *testing.T) {
		mockStore := &MockServerStore{
			err: errors.New("store error"),
		}
		sm := newTestServerManager(mockStore, &MockConnectionStringsStore{})

		servers, err := sm.GetServers()

		assert.Error(t, err)
		assert.Nil(t, servers)
	})
}

func TestAddServer(t *testing.T) {
	t.Run("successful add", func(t *testing.T) {
		mockStore := &MockServerStore{
			servers: []RegisteredServer{
				{ID: "parent", Name: "Parent Group", IsGroup: true},
			},
		}
		mockCSStore := &MockConnectionStringsStore{
			uris: make(map[string]string),
		}
		sm := newTestServerManager(mockStore, mockCSStore)

		err := sm.AddServer("parent", "New Server", "mongodb://localhost", "")

		assert.NoError(t, err)
		assert.Len(t, mockStore.servers, 2)
		assert.Len(t, mockCSStore.uris, 1)
	})
}

func TestUpdateServer(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		mockStore := &MockServerStore{
			servers: []RegisteredServer{
				{ID: "1", Name: "Server 1"},
			},
		}
		mockCSStore := &MockConnectionStringsStore{
			uris: map[string]string{"1": "mongodb://localhost"},
		}
		sm := newTestServerManager(mockStore, mockCSStore)

		err := sm.UpdateServer("1", "Updated Server", "mongodb://newhost", "a", "")

		assert.NoError(t, err)
		assert.Equal(t, "Updated Server", mockStore.servers[0].Name)
		assert.Equal(t, "mongodb://newhost", mockCSStore.uris["1"])
	})

	t.Run("server not found", func(t *testing.T) {
		mockStore := &MockServerStore{
			servers: []RegisteredServer{},
		}
		sm := newTestServerManager(mockStore, &MockConnectionStringsStore{})

		err := sm.UpdateServer("1", "Updated Server", "mongodb://newhost", "", "")

		assert.Error(t, err)
	})
}

func TestRemoveNode(t *testing.T) {
	t.Run("successful remove server", func(t *testing.T) {
		mockStore := &MockServerStore{
			servers: []RegisteredServer{
				{ID: "1", Name: "Server 1"},
			},
		}
		mockCSStore := &MockConnectionStringsStore{
			uris: map[string]string{"1": "mongodb://localhost"},
		}
		sm := newTestServerManager(mockStore, mockCSStore)

		err := sm.RemoveNode("1")

		assert.NoError(t, err)
		assert.Len(t, mockStore.servers, 0)
		assert.Len(t, mockCSStore.uris, 0)
	})

	t.Run("successful remove group", func(t *testing.T) {
		mockStore := &MockServerStore{
			servers: []RegisteredServer{
				{ID: "1", Name: "Group 1", IsGroup: true},
			},
		}
		sm := newTestServerManager(mockStore, &MockConnectionStringsStore{})

		err := sm.RemoveNode("1")

		assert.NoError(t, err)
		assert.Len(t, mockStore.servers, 0)
	})

	t.Run("node not found", func(t *testing.T) {
		mockStore := &MockServerStore{
			servers: []RegisteredServer{},
		}
		sm := newTestServerManager(mockStore, &MockConnectionStringsStore{})

		err := sm.RemoveNode("1")

		assert.Error(t, err)
	})

	t.Run("cannot remove group with children", func(t *testing.T) {
		mockStore := &MockServerStore{
			servers: []RegisteredServer{
				{ID: "1", Name: "Group 1", IsGroup: true},
				{ID: "2", Name: "Server 2", ParentID: "1"},
			},
		}
		sm := newTestServerManager(mockStore, &MockConnectionStringsStore{})

		err := sm.RemoveNode("1")

		assert.Error(t, err)
	})
}

func TestGetURI(t *testing.T) {
	t.Run("successful get uri", func(t *testing.T) {
		mockCSStore := &MockConnectionStringsStore{
			uris: map[string]string{"1": "mongodb://localhost"},
		}
		sm := newTestServerManager(&MockServerStore{}, mockCSStore)

		uri, err := sm.GetURI("1")

		assert.NoError(t, err)
		assert.Equal(t, "mongodb://localhost", uri)
	})

	t.Run("uri not found", func(t *testing.T) {
		mockCSStore := &MockConnectionStringsStore{
			uris: make(map[string]string),
		}
		sm := newTestServerManager(&MockServerStore{}, mockCSStore)

		uri, err := sm.GetURI("1")

		assert.Error(t, err)
		assert.Empty(t, uri)
	})
}