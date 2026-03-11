package queryengine

import (
	"testing"
	"time"
	"vervet/internal/shell"

	"github.com/stretchr/testify/assert"
)

func TestNewShellEngine(t *testing.T) {
	cfg := shell.Config{Timeout: 30 * time.Second}
	engine := NewShellEngine(cfg)
	assert.NotNil(t, engine)

	// Verify it satisfies the interface
	var _ QueryEngine = engine
}
