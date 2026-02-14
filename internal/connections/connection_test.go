package connections

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewActiveConnection(t *testing.T) {
	t.Run("creates connection with correct fields", func(t *testing.T) {
		conn := newActiveConnection("server1", "My Server")

		assert.Equal(t, "server1", conn.serverID)
		assert.Equal(t, "My Server", conn.name)
		assert.Nil(t, conn.client)
	})
}
