package connections

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager_DefaultEmitIsANoOp(t *testing.T) {
	cm := NewManager(slog.Default(), nil, nil, nil)
	require.NotNil(t, cm.emit)
	assert.NotPanics(t, func() { cm.emit(ConnectedEvent, "server-1") })
}

func TestSetEmitter_ReplacesEmit(t *testing.T) {
	cm := NewManager(slog.Default(), nil, nil, nil)

	type captured struct {
		name string
		data []any
	}
	var got []captured
	cm.SetEmitter(func(name string, data ...any) {
		got = append(got, captured{name: name, data: data})
	})

	cm.emit(ConnectedEvent, "server-1")

	require.Len(t, got, 1)
	assert.Equal(t, ConnectedEvent, got[0].name)
	assert.Equal(t, []any{"server-1"}, got[0].data)
}
