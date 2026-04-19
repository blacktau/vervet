package api

import (
	"errors"
	"testing"

	"vervet/internal/models"
	"vervet/internal/servers"

	"github.com/stretchr/testify/assert"
)

type MockServersProvider struct {
	err     error
	server  *models.RegisteredServer
	servers []models.RegisteredServer
}

func (m *MockServersProvider) GetServers() ([]models.RegisteredServer, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.servers, nil
}

func (m *MockServersProvider) GetServer(id string) (*models.RegisteredServer, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.server, nil
}

func (m *MockServersProvider) AddServer(parentID, name, uri, colour string) error {
	return m.err
}

func (m *MockServersProvider) UpdateServer(serverID, name, uri, parentID, colour string) error {
	return m.err
}

func (m *MockServersProvider) RemoveNode(id string) error {
	return m.err
}

func (m *MockServersProvider) GetURI(id string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "mongodb://localhost", nil
}

func (m *MockServersProvider) CreateGroup(parentID, name string) error {
	return m.err
}

func (m *MockServersProvider) UpdateGroup(groupID, name, parentID string) error {
	return m.err
}

func (m *MockServersProvider) AddServerWithConfig(parentID, name, colour string, cfg models.ConnectionConfig) error {
	return m.err
}

func (m *MockServersProvider) UpdateServerWithConfig(serverID, name, parentID, colour string, cfg models.ConnectionConfig) error {
	return m.err
}

func (m *MockServersProvider) GetConnectionConfig(serverID string) (models.ConnectionConfig, error) {
	if m.err != nil {
		return models.ConnectionConfig{}, m.err
	}
	return models.ConnectionConfig{URI: "mongodb://localhost", AuthMethod: models.AuthPassword}, nil
}

func (m *MockServersProvider) ExportServers(serverIDs []string, includeSensitiveData bool) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []byte(`{"version":1,"servers":[]}`), nil
}

func (m *MockServersProvider) ImportServers(data []byte) (*servers.ImportResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &servers.ImportResult{Created: []models.RegisteredServer{}}, nil
}

func TestServersProxy_GetServers(t *testing.T) {
	t.Run("successful get servers", func(t *testing.T) {
		provider := &MockServersProvider{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Server 1"},
				{ID: "2", Name: "Server 2"},
			},
		}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.GetServers()

		assert.True(t, result.IsSuccess)
		assert.Len(t, result.Data, 2)
	})

	t.Run("get servers error", func(t *testing.T) {
		provider := &MockServersProvider{
			err: errors.New("failed to get servers"),
		}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.GetServers()

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestServersProxy_GetServer(t *testing.T) {
	t.Run("successful get server", func(t *testing.T) {
		provider := &MockServersProvider{
			server: &models.RegisteredServer{ID: "1", Name: "Server 1"},
		}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.GetServer("1")

		assert.True(t, result.IsSuccess)
		assert.Equal(t, "Server 1", result.Data.Name)
	})

	t.Run("server not found", func(t *testing.T) {
		provider := &MockServersProvider{
			server: nil,
		}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.GetServer("1")

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
		assert.Contains(t, result.ErrorDetail, "not found")
	})

	t.Run("get server error", func(t *testing.T) {
		provider := &MockServersProvider{
			err: errors.New("failed to get server"),
		}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.GetServer("1")

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestServersProxy_CreateGroup(t *testing.T) {
	t.Run("successful create group", func(t *testing.T) {
		provider := &MockServersProvider{}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.CreateGroup("parent", "New Group")

		assert.True(t, result.IsSuccess)
	})

	t.Run("create group error", func(t *testing.T) {
		provider := &MockServersProvider{
			err: errors.New("failed to create group"),
		}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.CreateGroup("parent", "New Group")

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestServersProxy_UpdateGroup(t *testing.T) {
	t.Run("successful update group", func(t *testing.T) {
		provider := &MockServersProvider{}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.UpdateGroup("1", "Updated Group", "")

		assert.True(t, result.IsSuccess)
	})

	t.Run("update group error", func(t *testing.T) {
		provider := &MockServersProvider{
			err: errors.New("failed to update group"),
		}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.UpdateGroup("1", "Updated Group", "")

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestServersProxy_SaveServer(t *testing.T) {
	t.Run("successful save server", func(t *testing.T) {
		provider := &MockServersProvider{}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.SaveServer("parent", "New Server", "mongodb://localhost", "")

		assert.True(t, result.IsSuccess)
	})

	t.Run("save server error", func(t *testing.T) {
		provider := &MockServersProvider{
			err: errors.New("failed to save server"),
		}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.SaveServer("parent", "New Server", "mongodb://localhost", "")

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestServersProxy_UpdateServer(t *testing.T) {
	t.Run("successful update server", func(t *testing.T) {
		provider := &MockServersProvider{}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.UpdateServer("1", "Updated Server", "mongodb://localhost", "", "")

		assert.True(t, result.IsSuccess)
	})

	t.Run("update server error", func(t *testing.T) {
		provider := &MockServersProvider{
			err: errors.New("failed to update server"),
		}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.UpdateServer("1", "Updated Server", "mongodb://localhost", "", "")

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestServersProxy_RemoveNode(t *testing.T) {
	t.Run("successful remove node", func(t *testing.T) {
		provider := &MockServersProvider{}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.RemoveNode("1")

		assert.True(t, result.IsSuccess)
	})

	t.Run("remove node error", func(t *testing.T) {
		provider := &MockServersProvider{
			err: errors.New("failed to remove node"),
		}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.RemoveNode("1")

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestServersProxy_GetURI(t *testing.T) {
	t.Run("successful get URI", func(t *testing.T) {
		provider := &MockServersProvider{}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.GetURI("1")

		assert.True(t, result.IsSuccess)
		assert.Equal(t, "mongodb://localhost", result.Data)
	})

	t.Run("get URI error", func(t *testing.T) {
		provider := &MockServersProvider{
			err: errors.New("failed to get URI"),
		}
		proxy := NewServersProxy(testLogger(), provider)

		result := proxy.GetURI("1")

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}
