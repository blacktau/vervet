//go:build integration

package connections

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"

	"vervet/internal/clientregistry"
	"vervet/internal/models"
)

var testURI string

func TestMain(m *testing.M) {
	ctx := context.Background()
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	container, err := mongodb.Run(ctx, "mongo:7")
	if err != nil {
		log.Fatalf("start container: %v", err)
	}
	defer func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			log.Printf("terminate: %v", err)
		}
	}()

	testURI, err = container.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("conn string: %v", err)
	}

	os.Exit(m.Run())
}

// fakeServerProvider satisfies ServerProvider.
type fakeServerProvider struct {
	servers map[string]*models.RegisteredServer
}

func (f fakeServerProvider) GetServer(id string) (*models.RegisteredServer, error) {
	s, ok := f.servers[id]
	if !ok {
		return nil, fmt.Errorf("no server %s", id)
	}
	return s, nil
}

// fakeStore satisfies connectionStrings.Store. Only GetConnectionConfig is
// exercised; the rest are present to satisfy the interface.
type fakeStore struct {
	configs map[string]models.ConnectionConfig
}

func (f fakeStore) StoreRegisteredServerURI(string, string) error               { return nil }
func (f fakeStore) GetRegisteredServerURI(string) (string, error)               { return "", nil }
func (f fakeStore) DeleteRegisteredServerURI(string) error                      { return nil }
func (f fakeStore) StoreConnectionConfig(string, models.ConnectionConfig) error { return nil }
func (f fakeStore) UpdateRefreshToken(string, string) error                     { return nil }

func (f fakeStore) GetConnectionConfig(id string) (models.ConnectionConfig, error) {
	cfg, ok := f.configs[id]
	if !ok {
		return models.ConnectionConfig{}, fmt.Errorf("no config for %s", id)
	}
	return cfg, nil
}

// recorder captures emitted Wails events in place of runtime.EventsEmit.
type recorder struct {
	mu     sync.Mutex
	events []event
}

type event struct {
	name string
	data []interface{}
}

func (r *recorder) emit(_ context.Context, name string, data ...interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event{name: name, data: data})
}

func (r *recorder) names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.events))
	for _, e := range r.events {
		out = append(out, e.name)
	}
	return out
}

// newManager wires a real ClientRegistry to the container, a fake store and a
// fake server provider, and replaces the Wails emitter with a recorder.
func newManager(t *testing.T, serverID, name string, cfg models.ConnectionConfig) (*ConnectionManager, *recorder) {
	t.Helper()

	registry := clientregistry.NewClientRegistry(slog.Default(), nil)
	registry.Init(context.Background())
	t.Cleanup(func() { registry.DisconnectAll() })

	provider := fakeServerProvider{servers: map[string]*models.RegisteredServer{
		serverID: {ID: serverID, Name: name},
	}}
	store := fakeStore{configs: map[string]models.ConnectionConfig{serverID: cfg}}

	cm := NewManager(slog.Default(), registry, store, provider)
	require.NoError(t, cm.Init(context.Background()))

	rec := &recorder{}
	cm.emit = rec.emit
	return cm, rec
}

func TestIntegration_Connect_EmitsConnectedEvent(t *testing.T) {
	cm, rec := newManager(t, "srv-1", "Primary", models.ConnectionConfig{
		URI: testURI, AuthMethod: models.AuthNone,
	})

	conn, err := cm.Connect("srv-1")
	require.NoError(t, err)

	assert.Equal(t, "srv-1", conn.ServerID)
	assert.Equal(t, "Primary", conn.Name)
	assert.Equal(t, []string{ConnectedEvent}, rec.names())
}

func TestIntegration_Connect_DuplicateRejectedAndEmitsNothing(t *testing.T) {
	cm, rec := newManager(t, "srv-dup", "Dup", models.ConnectionConfig{
		URI: testURI, AuthMethod: models.AuthNone,
	})

	_, err := cm.Connect("srv-dup")
	require.NoError(t, err)

	_, err = cm.Connect("srv-dup")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already connected")

	assert.Equal(t, []string{ConnectedEvent}, rec.names(), "the rejected connect must not emit")
}

func TestIntegration_Connect_UnknownServerErrors(t *testing.T) {
	cm, rec := newManager(t, "srv-known", "K", models.ConnectionConfig{
		URI: testURI, AuthMethod: models.AuthNone,
	})

	_, err := cm.Connect("srv-unknown")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error retrieving server")
	assert.Empty(t, rec.names())
}

func TestIntegration_GetConnections_ReflectsRegistry(t *testing.T) {
	cm, _ := newManager(t, "srv-list", "Listed", models.ConnectionConfig{
		URI: testURI, AuthMethod: models.AuthNone,
	})

	assert.Empty(t, cm.GetConnections())

	_, err := cm.Connect("srv-list")
	require.NoError(t, err)

	conns := cm.GetConnections()
	require.Len(t, conns, 1)
	assert.Equal(t, "srv-list", conns[0].ServerID)
	assert.Equal(t, "Listed", conns[0].Name)
}

func TestIntegration_Disconnect_EmitsDisconnectedEvent(t *testing.T) {
	cm, rec := newManager(t, "srv-dc", "D", models.ConnectionConfig{
		URI: testURI, AuthMethod: models.AuthNone,
	})

	_, err := cm.Connect("srv-dc")
	require.NoError(t, err)
	require.NoError(t, cm.Disconnect("srv-dc"))

	assert.Equal(t, []string{ConnectedEvent, DisconnectedEvent}, rec.names())
	assert.Empty(t, cm.GetConnections())
}

// Disconnecting an unknown server still emits, because the registry treats it
// as success and the frontend must converge on "disconnected".
func TestIntegration_Disconnect_UnknownStillEmits(t *testing.T) {
	cm, rec := newManager(t, "srv-any", "A", models.ConnectionConfig{
		URI: testURI, AuthMethod: models.AuthNone,
	})

	require.NoError(t, cm.Disconnect("never-connected"))
	assert.Equal(t, []string{DisconnectedEvent}, rec.names())
}

func TestIntegration_DisconnectAll_EmitsPerServer(t *testing.T) {
	cm, rec := newManager(t, "srv-1", "One", models.ConnectionConfig{
		URI: testURI, AuthMethod: models.AuthNone,
	})

	_, err := cm.Connect("srv-1")
	require.NoError(t, err)
	require.NoError(t, cm.DisconnectAll())

	assert.Equal(t, []string{ConnectedEvent, DisconnectedEvent}, rec.names())
	assert.Empty(t, cm.GetConnections())
}

func TestIntegration_TestConnection_Succeeds(t *testing.T) {
	cm, _ := newManager(t, "srv-t", "T", models.ConnectionConfig{
		URI: testURI, AuthMethod: models.AuthNone,
	})

	ok, err := cm.TestConnection(testURI)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestIntegration_TestConnection_BadURIFails(t *testing.T) {
	cm, _ := newManager(t, "srv-t", "T", models.ConnectionConfig{
		URI: testURI, AuthMethod: models.AuthNone,
	})

	ok, err := cm.TestConnection("mongodb://127.0.0.1:1/?serverSelectionTimeoutMS=500")
	require.Error(t, err)
	assert.False(t, ok)
}

func TestIntegration_TestConnectionWithConfig_Succeeds(t *testing.T) {
	cm, _ := newManager(t, "srv-t", "T", models.ConnectionConfig{
		URI: testURI, AuthMethod: models.AuthNone,
	})

	ok, err := cm.TestConnectionWithConfig(context.Background(), models.ConnectionConfig{
		URI: testURI, AuthMethod: models.AuthNone,
	})
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestIntegration_TestConnectionWithConfig_RejectsOIDC(t *testing.T) {
	cm, _ := newManager(t, "srv-t", "T", models.ConnectionConfig{
		URI: testURI, AuthMethod: models.AuthNone,
	})

	ok, err := cm.TestConnectionWithConfig(context.Background(), models.ConnectionConfig{
		URI: testURI, AuthMethod: models.AuthOIDC,
	})
	require.Error(t, err)
	assert.False(t, ok)
	assert.Contains(t, err.Error(), "not supported for OIDC")
}
