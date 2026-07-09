package connections

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager_DefaultsEmitToWailsRuntime(t *testing.T) {
	cm := NewManager(slog.Default(), nil, nil, nil)
	require.NotNil(t, cm.emit, "emit must be defaulted so production keeps emitting events")
}

func TestNewManager_EmitIsOverridable(t *testing.T) {
	cm := NewManager(slog.Default(), nil, nil, nil)

	type captured struct {
		name string
		data []interface{}
	}
	var got []captured
	cm.emit = func(_ context.Context, name string, data ...interface{}) {
		got = append(got, captured{name: name, data: data})
	}

	cm.emit(context.Background(), ConnectedEvent, "server-1")

	require.Len(t, got, 1)
	assert.Equal(t, ConnectedEvent, got[0].name)
	assert.Equal(t, []interface{}{"server-1"}, got[0].data)
}
