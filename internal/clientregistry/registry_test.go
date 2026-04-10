package clientregistry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClientRegistry(t *testing.T) {
	t.Run("creates registry with empty clients map", func(t *testing.T) {
		reg := NewClientRegistry(nil, nil)
		assert.NotNil(t, reg)
		assert.False(t, reg.IsConnected("nonexistent"))
	})
}

func TestClientRegistry_Connect_GetClient(t *testing.T) {
	t.Run("GetClient returns error for unknown serverID", func(t *testing.T) {
		reg := NewClientRegistry(nil, nil)
		reg.Init(context.Background())

		_, err := reg.GetClient("unknown")
		assert.Error(t, err)
	})
}

func TestClientRegistry_IsConnected(t *testing.T) {
	t.Run("returns false for unknown serverID", func(t *testing.T) {
		reg := NewClientRegistry(nil, nil)
		assert.False(t, reg.IsConnected("unknown"))
	})
}

func TestClientRegistry_Disconnect(t *testing.T) {
	t.Run("returns error for unknown serverID", func(t *testing.T) {
		reg := NewClientRegistry(nil, nil)
		reg.Init(context.Background())

		err := reg.Disconnect("unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no active connection")
	})

	t.Run("removes client from map even when client is nil", func(t *testing.T) {
		reg := NewClientRegistry(nil, nil)
		reg.Init(context.Background())

		// Manually inject a client to simulate a connected state
		reg.mu.Lock()
		reg.clients["test-server"] = registeredClient{
			client:   nil,
			serverID: "test-server",
			name:     "Test",
		}
		reg.mu.Unlock()

		assert.True(t, reg.IsConnected("test-server"))

		// Disconnect handles nil client gracefully and should still clean up
		_ = reg.Disconnect("test-server")

		assert.False(t, reg.IsConnected("test-server"))
	})
}
