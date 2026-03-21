package api

import (
	"context"
	"errors"
	"testing"

	"vervet/internal/models"

	"github.com/stretchr/testify/assert"
)

type MockConnectionsProvider struct {
	connection    models.Connection
	connectErr    error
	disconnectErr error
	testConnErr   error
	connections   []models.Connection
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

func TestConnectionsProxy_Connect(t *testing.T) {
	t.Run("successful connect", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			connection: models.Connection{ServerID: "1", Name: "Server 1"},
		}
		proxy := NewConnectionsProxy(provider)

		result := proxy.Connect("1")

		assert.True(t, result.IsSuccess)
		assert.Equal(t, "Server 1", result.Data.Name)
		assert.Empty(t, result.ErrorCode)
	})

	t.Run("connect error", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			connectErr: errors.New("connection failed"),
		}
		proxy := NewConnectionsProxy(provider)

		result := proxy.Connect("1")

		assert.False(t, result.IsSuccess)
		assert.Empty(t, result.Data)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestConnectionsProxy_Disconnect(t *testing.T) {
	t.Run("successful disconnect", func(t *testing.T) {
		provider := &MockConnectionsProvider{}
		proxy := NewConnectionsProxy(provider)

		result := proxy.Disconnect("1")

		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.ErrorCode)
	})

	t.Run("disconnect error", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			disconnectErr: errors.New("disconnect failed"),
		}
		proxy := NewConnectionsProxy(provider)

		result := proxy.Disconnect("1")

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}

func TestConnectionsProxy_DisconnectAll(t *testing.T) {
	t.Run("successful disconnect all", func(t *testing.T) {
		provider := &MockConnectionsProvider{}
		proxy := NewConnectionsProxy(provider)

		result := proxy.DisconnectAll()

		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.ErrorCode)
	})

	t.Run("disconnect all error", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			disconnectErr: errors.New("disconnect failed"),
		}
		proxy := NewConnectionsProxy(provider)

		result := proxy.DisconnectAll()

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
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
		proxy := NewConnectionsProxy(provider)

		result := proxy.GetConnections()

		assert.True(t, result.IsSuccess)
		assert.Len(t, result.Data, 2)
	})
}

func TestConnectionsProxy_TestConnection(t *testing.T) {
	t.Run("successful test connection", func(t *testing.T) {
		provider := &MockConnectionsProvider{}
		proxy := NewConnectionsProxy(provider)

		result := proxy.TestConnection("mongodb://localhost")

		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.ErrorCode)
	})

	t.Run("test connection error", func(t *testing.T) {
		provider := &MockConnectionsProvider{
			testConnErr: errors.New("connection failed"),
		}
		proxy := NewConnectionsProxy(provider)

		result := proxy.TestConnection("mongodb://localhost")

		assert.False(t, result.IsSuccess)
		assert.NotEmpty(t, result.ErrorCode)
	})
}
