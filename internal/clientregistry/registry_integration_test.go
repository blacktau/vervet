//go:build integration

package clientregistry

import (
	"context"
	"log"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"

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

func newRegistry(t *testing.T) *ClientRegistry {
	t.Helper()
	// nil TokenManager is safe: no test exercises the AuthOIDC branch.
	r := NewClientRegistry(slog.Default(), nil)
	r.Init(context.Background())
	t.Cleanup(func() { r.DisconnectAll() })
	return r
}

func TestIntegration_Connect_RegistersAndPings(t *testing.T) {
	r := newRegistry(t)

	client, err := r.Connect("srv-1", "Primary", testURI)
	require.NoError(t, err)
	require.NotNil(t, client)

	assert.True(t, r.IsConnected("srv-1"))

	got, err := r.GetClient("srv-1")
	require.NoError(t, err)
	assert.Same(t, client, got, "GetClient must return the registered client")
}

func TestIntegration_Connect_DuplicateRejected(t *testing.T) {
	r := newRegistry(t)

	_, err := r.Connect("srv-dup", "A", testURI)
	require.NoError(t, err)

	_, err = r.Connect("srv-dup", "A again", testURI)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already connected")
}

func TestIntegration_Connect_BadURIFailsPing(t *testing.T) {
	r := newRegistry(t)

	// Port 1 has nothing listening; serverSelectionTimeoutMS keeps it quick.
	_, err := r.Connect("srv-bad", "Bad", "mongodb://127.0.0.1:1/?serverSelectionTimeoutMS=500")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ping failed")

	assert.False(t, r.IsConnected("srv-bad"), "a failed connect must not register a client")
}

func TestIntegration_ConnectWithConfig_NonOIDC(t *testing.T) {
	r := newRegistry(t)

	client, err := r.ConnectWithConfig("srv-cfg", "Cfg", models.ConnectionConfig{
		URI:        testURI,
		AuthMethod: models.AuthNone,
	})
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.True(t, r.IsConnected("srv-cfg"))
}

func TestIntegration_GetClient_UnknownServerErrors(t *testing.T) {
	r := newRegistry(t)

	_, err := r.GetClient("nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active connection")
}

func TestIntegration_GetAll_ReturnsEveryRegisteredClient(t *testing.T) {
	r := newRegistry(t)

	require.NoError(t, mustConnect(t, r, "a", "Alpha"))
	require.NoError(t, mustConnect(t, r, "b", "Beta"))

	all := r.GetAll()
	require.Len(t, all, 2)

	byID := map[string]string{}
	for _, c := range all {
		byID[c.ServerID] = c.Name
	}
	assert.Equal(t, "Alpha", byID["a"])
	assert.Equal(t, "Beta", byID["b"])
}

func TestIntegration_Disconnect_RemovesClient(t *testing.T) {
	r := newRegistry(t)
	require.NoError(t, mustConnect(t, r, "srv-dc", "X"))

	require.NoError(t, r.Disconnect("srv-dc"))
	assert.False(t, r.IsConnected("srv-dc"))

	_, err := r.GetClient("srv-dc")
	assert.Error(t, err)
}

func TestIntegration_Disconnect_UnknownIsIdempotent(t *testing.T) {
	r := newRegistry(t)
	// Documented behaviour: disconnecting an absent server is success, so the
	// frontend can retry against stale state without a warning.
	assert.NoError(t, r.Disconnect("never-connected"))
}

func TestIntegration_DisconnectAll_ClearsRegistry(t *testing.T) {
	r := newRegistry(t)
	require.NoError(t, mustConnect(t, r, "x", "X"))
	require.NoError(t, mustConnect(t, r, "y", "Y"))

	require.NoError(t, r.DisconnectAll())

	assert.Empty(t, r.GetAll())
	assert.False(t, r.IsConnected("x"))
	assert.False(t, r.IsConnected("y"))
}

func mustConnect(t *testing.T, r *ClientRegistry, id, name string) error {
	t.Helper()
	_, err := r.Connect(id, name, testURI)
	return err
}
