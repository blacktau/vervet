package clientregistry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClientRegistry(t *testing.T) {
	t.Run("creates registry with empty clients map", func(t *testing.T) {
		reg := NewClientRegistry(nil)
		assert.NotNil(t, reg)
		assert.False(t, reg.IsConnected("nonexistent"))
	})
}

func TestClientRegistry_Connect_GetClient(t *testing.T) {
	t.Run("GetClient returns error for unknown serverID", func(t *testing.T) {
		reg := NewClientRegistry(nil)
		reg.Init(context.Background())

		_, err := reg.GetClient("unknown")
		assert.Error(t, err)
	})
}

func TestClientRegistry_IsConnected(t *testing.T) {
	t.Run("returns false for unknown serverID", func(t *testing.T) {
		reg := NewClientRegistry(nil)
		assert.False(t, reg.IsConnected("unknown"))
	})
}
