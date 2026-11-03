package servers

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"sync"
	"testing"
	"vervet/internal/connectionStrings"
	"vervet/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockServerStore is a mock implementation of serverStore for testing.
type mockServerStore struct {
	servers []models.RegisteredServer
	err     error
}

func (m *mockServerStore) LoadServers() ([]models.RegisteredServer, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.servers, nil
}

func (m *mockServerStore) SaveServers(servers []models.RegisteredServer) error {
	if m.err != nil {
		return m.err
	}
	m.servers = servers
	return nil
}

// MockConnectionStringsStore is a mock implementation of ConnectionStringsStore for testing.
type MockConnectionStringsStore struct {
	uris    map[string]string
	configs map[string]models.ConnectionConfig
	err     error
}

func (m *MockConnectionStringsStore) StoreRegisteredServerURI(id, uri string) error {
	if m.err != nil {
		return m.err
	}
	m.uris[id] = uri
	return nil
}

func (m *MockConnectionStringsStore) GetRegisteredServerURI(id string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	uri, ok := m.uris[id]
	if !ok {
		return "", errors.New("URI not found")
	}
	return uri, nil
}

func (m *MockConnectionStringsStore) DeleteRegisteredServerURI(id string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.uris, id)
	return nil
}

func (m *MockConnectionStringsStore) StoreConnectionConfig(id string, cfg models.ConnectionConfig) error {
	if m.err != nil {
		return m.err
	}
	if m.configs == nil {
		m.configs = map[string]models.ConnectionConfig{}
	}
	m.configs[id] = cfg
	m.uris[id] = cfg.URI
	return nil
}

func (m *MockConnectionStringsStore) GetConnectionConfig(id string) (models.ConnectionConfig, error) {
	if m.err != nil {
		return models.ConnectionConfig{}, m.err
	}
	if cfg, ok := m.configs[id]; ok {
		return cfg, nil
	}
	uri, ok := m.uris[id]
	if !ok {
		return models.ConnectionConfig{}, errors.New("config not found")
	}
	return models.ConnectionConfig{URI: uri, AuthMethod: models.AuthPassword}, nil
}

func (m *MockConnectionStringsStore) UpdateRefreshToken(id, refreshToken, binding string) error {
	return m.err
}

func newTestServerService(store ServerStore, connectionStringsStore connectionStrings.Store) *ServerService {
	log := slog.Default()
	return &ServerService{
		log:               log,
		mu:                sync.RWMutex{},
		connectionStrings: connectionStringsStore,
		ctx:               context.Background(),
		store:             store,
	}
}

func TestGetServers(t *testing.T) {
	// Test case 1: Successful get servers
	t.Run("successful get servers", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Server 1"},
				{ID: "2", Name: "Server 2"},
			},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		servers, err := sm.GetServers()

		assert.NoError(t, err)
		assert.Len(t, servers, 2)
	})

	// Test case 2: Store returns an error
	t.Run("store error", func(t *testing.T) {
		mockStore := &mockServerStore{
			err: errors.New("store error"),
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		servers, err := sm.GetServers()

		assert.Error(t, err)
		assert.Nil(t, servers)
	})
}

func TestAddServer(t *testing.T) {
	t.Run("successful add", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "parent", Name: "Parent Group", IsGroup: true},
			},
		}
		mockCSStore := &MockConnectionStringsStore{
			uris: make(map[string]string),
		}
		sm := newTestServerService(mockStore, mockCSStore)

		err := sm.AddServer("parent", "New Server", "mongodb://localhost", "")

		assert.NoError(t, err)
		assert.Len(t, mockStore.servers, 2)
		assert.Len(t, mockCSStore.uris, 1)
	})
}

func TestUpdateServer(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Server 1"},
			},
		}
		mockCSStore := &MockConnectionStringsStore{
			uris: map[string]string{"1": "mongodb://localhost"},
		}
		sm := newTestServerService(mockStore, mockCSStore)

		err := sm.UpdateServer("1", "Updated Server", "mongodb://newhost", "a", "")

		assert.NoError(t, err)
		assert.Equal(t, "Updated Server", mockStore.servers[0].Name)
		assert.Equal(t, "mongodb://newhost", mockCSStore.uris["1"])
	})

	t.Run("server not found", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		err := sm.UpdateServer("1", "Updated Server", "mongodb://newhost", "", "")

		assert.Error(t, err)
	})
}

func TestRemoveNode(t *testing.T) {
	t.Run("successful remove server", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Server 1"},
			},
		}
		mockCSStore := &MockConnectionStringsStore{
			uris: map[string]string{"1": "mongodb://localhost"},
		}
		sm := newTestServerService(mockStore, mockCSStore)

		err := sm.RemoveNode("1")

		assert.NoError(t, err)
		assert.Len(t, mockStore.servers, 0)
		assert.Len(t, mockCSStore.uris, 0)
	})

	t.Run("successful remove group", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Group 1", IsGroup: true},
			},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		err := sm.RemoveNode("1")

		assert.NoError(t, err)
		assert.Len(t, mockStore.servers, 0)
	})

	t.Run("node not found", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		err := sm.RemoveNode("1")

		assert.Error(t, err)
	})

	t.Run("remove group with children recursively", func(t *testing.T) {
		mockCSStore := &MockConnectionStringsStore{
			uris: map[string]string{"2": "mongodb://localhost"},
		}
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Group 1", IsGroup: true},
				{ID: "2", Name: "Server 2", ParentID: "1"},
			},
		}
		sm := newTestServerService(mockStore, mockCSStore)

		err := sm.RemoveNode("1")

		assert.NoError(t, err)
		assert.Len(t, mockStore.servers, 0)
		assert.Len(t, mockCSStore.uris, 0)
	})
}

func TestUpdateServer_KeyringFailureDoesNotPersistMetadata(t *testing.T) {
	t.Run("metadata unchanged when keyring store fails", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Original Name"},
			},
		}
		mockCSStore := &MockConnectionStringsStore{
			uris: map[string]string{"1": "mongodb://localhost"},
			err:  errors.New("keyring locked"),
		}
		sm := newTestServerService(mockStore, mockCSStore)

		err := sm.UpdateServer("1", "New Name", "mongodb://newhost", "", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "keyring locked")
		// Metadata should NOT have been updated since keyring failed
		assert.Equal(t, "Original Name", mockStore.servers[0].Name)
	})
}

// countingConnectionStringsStore fails StoreConnectionConfig at a specific call index.
type countingConnectionStringsStore struct {
	inner      connectionStrings.Store
	failAtCall int
	callCount  int
}

func (m *countingConnectionStringsStore) StoreConnectionConfig(id string, cfg models.ConnectionConfig) error {
	idx := m.callCount
	m.callCount++
	if idx == m.failAtCall {
		return errors.New("simulated keyring failure")
	}
	return m.inner.StoreConnectionConfig(id, cfg)
}

func (m *countingConnectionStringsStore) StoreRegisteredServerURI(id, uri string) error {
	return m.inner.StoreRegisteredServerURI(id, uri)
}

func (m *countingConnectionStringsStore) GetRegisteredServerURI(id string) (string, error) {
	return m.inner.GetRegisteredServerURI(id)
}

func (m *countingConnectionStringsStore) DeleteRegisteredServerURI(id string) error {
	return m.inner.DeleteRegisteredServerURI(id)
}

func (m *countingConnectionStringsStore) GetConnectionConfig(id string) (models.ConnectionConfig, error) {
	return m.inner.GetConnectionConfig(id)
}

func (m *countingConnectionStringsStore) UpdateRefreshToken(id, refreshToken, binding string) error {
	return m.inner.UpdateRefreshToken(id, refreshToken, binding)
}

func TestGetURI(t *testing.T) {
	t.Run("successful get uri", func(t *testing.T) {
		mockCSStore := &MockConnectionStringsStore{
			uris: map[string]string{"1": "mongodb://localhost"},
		}
		sm := newTestServerService(&mockServerStore{}, mockCSStore)

		uri, err := sm.GetURI("1")

		assert.NoError(t, err)
		assert.Equal(t, "mongodb://localhost", uri)
	})

	t.Run("uri not found", func(t *testing.T) {
		mockCSStore := &MockConnectionStringsStore{
			uris: make(map[string]string),
		}
		sm := newTestServerService(&mockServerStore{}, mockCSStore)

		uri, err := sm.GetURI("1")

		assert.Error(t, err)
		assert.Empty(t, uri)
	})
}

func TestBuildFullConnectionString_VerbatimAuthMethods(t *testing.T) {
	cases := []struct {
		name   string
		method models.AuthMethod
		uri    string
	}{
		{"none", models.AuthNone, "mongodb://example.com:27017/?retryWrites=true"},
		{"password", models.AuthPassword, "mongodb://user:pass@example.com:27017/admin"},
		{"x509", models.AuthX509, "mongodb://example.com/?authMechanism=MONGODB-X509&tlsCertificateKeyFile=/c.pem"},
		{"aws", models.AuthAWS, "mongodb://example.com/?authMechanism=MONGODB-AWS"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cs := &MockConnectionStringsStore{uris: map[string]string{}}
			require.NoError(t, cs.StoreConnectionConfig("s1", models.ConnectionConfig{
				URI:        tc.uri,
				AuthMethod: tc.method,
			}))
			store := &mockServerStore{servers: []models.RegisteredServer{{ID: "s1", Name: "S1"}}}
			sm := newTestServerService(store, cs)

			got, err := sm.BuildFullConnectionString("s1")
			require.NoError(t, err)
			assert.Equal(t, tc.uri, got)
		})
	}
}

func TestBuildFullConnectionString_ServerNotFound(t *testing.T) {
	cs := &MockConnectionStringsStore{uris: map[string]string{}}
	store := &mockServerStore{servers: []models.RegisteredServer{}}
	sm := newTestServerService(store, cs)

	_, err := sm.BuildFullConnectionString("missing")
	assert.Error(t, err)
}

func TestBuildFullConnectionString_KeyringError(t *testing.T) {
	cs := &MockConnectionStringsStore{uris: map[string]string{}, err: errors.New("keyring locked")}
	store := &mockServerStore{servers: []models.RegisteredServer{{ID: "s1", Name: "S1"}}}
	sm := newTestServerService(store, cs)

	_, err := sm.BuildFullConnectionString("s1")
	assert.Error(t, err)
}

func TestBuildFullConnectionString_StoreError(t *testing.T) {
	cs := &MockConnectionStringsStore{uris: map[string]string{}}
	store := &mockServerStore{err: errors.New("yaml parse error")}
	sm := newTestServerService(store, cs)

	_, err := sm.BuildFullConnectionString("s1")
	assert.Error(t, err)
}

func TestBuildFullConnectionString_OIDC_AddsParams(t *testing.T) {
	cs := &MockConnectionStringsStore{uris: map[string]string{}}
	require.NoError(t, cs.StoreConnectionConfig("s1", models.ConnectionConfig{
		URI:        "mongodb://example.com/",
		AuthMethod: models.AuthOIDC,
	}))
	store := &mockServerStore{servers: []models.RegisteredServer{{ID: "s1", Name: "S1"}}}
	sm := newTestServerService(store, cs)

	got, err := sm.BuildFullConnectionString("s1")
	require.NoError(t, err)

	u, err := url.Parse(got)
	require.NoError(t, err)
	assert.Equal(t, "MONGODB-OIDC", u.Query().Get("authMechanism"))
	assert.Equal(t, "$external", u.Query().Get("authSource"))
	assert.NotContains(t, got, "ALLOWED_HOSTS")
}

func TestBuildFullConnectionString_OIDC_PreservesUserParams(t *testing.T) {
	in := "mongodb://example.com/?authMechanism=MONGODB-OIDC&authMechanismProperties=ENVIRONMENT:azure"
	cs := &MockConnectionStringsStore{uris: map[string]string{}}
	require.NoError(t, cs.StoreConnectionConfig("s1", models.ConnectionConfig{
		URI:        in,
		AuthMethod: models.AuthOIDC,
	}))
	store := &mockServerStore{servers: []models.RegisteredServer{{ID: "s1", Name: "S1"}}}
	sm := newTestServerService(store, cs)

	got, err := sm.BuildFullConnectionString("s1")
	require.NoError(t, err)
	u, err := url.Parse(got)
	require.NoError(t, err)
	assert.Equal(t, "MONGODB-OIDC", u.Query().Get("authMechanism"))
	assert.Equal(t, "ENVIRONMENT:azure", u.Query().Get("authMechanismProperties"))
}

// recordingDisconnector records disconnect calls and the order relative to
// keyring deletion, so the test can assert we disconnect before wiping secrets.
type recordingDisconnector struct {
	disconnected []string
	onDisconnect func(serverID string)
}

func (d *recordingDisconnector) Disconnect(serverID string) error {
	d.disconnected = append(d.disconnected, serverID)
	if d.onDisconnect != nil {
		d.onDisconnect(serverID)
	}
	return nil
}

// Deleting a group must disconnect its servers. Otherwise their clients stay
// live in the registry with no name left to close them by, and their OIDC
// tokens are cleared before the eventual disconnect — which makes the driver's
// disconnect-time re-auth pop a browser login.
func TestRemoveNode_DisconnectsRemovedServers(t *testing.T) {
	t.Run("group children are disconnected before their secrets are dropped", func(t *testing.T) {
		mockCSStore := &MockConnectionStringsStore{
			uris: map[string]string{"2": "mongodb://localhost"},
		}
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Group 1", IsGroup: true},
				{ID: "2", Name: "Server 2", ParentID: "1"},
			},
		}
		sm := newTestServerService(mockStore, mockCSStore)

		uriStillPresent := false
		disconnector := &recordingDisconnector{
			onDisconnect: func(serverID string) {
				_, uriStillPresent = mockCSStore.uris[serverID]
			},
		}
		sm.SetDisconnector(disconnector)

		err := sm.RemoveNode("1")

		assert.NoError(t, err)
		assert.Equal(t, []string{"2"}, disconnector.disconnected, "group child was not disconnected")
		assert.True(t, uriStillPresent, "disconnect ran after the keyring entry was deleted")
	})

	t.Run("groups themselves are not disconnected", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Group 1", IsGroup: true},
			},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})
		disconnector := &recordingDisconnector{}
		sm.SetDisconnector(disconnector)

		err := sm.RemoveNode("1")

		assert.NoError(t, err)
		assert.Empty(t, disconnector.disconnected)
	})
}

type fakeOIDCSessions struct {
	forgotten []string
}

func (f *fakeOIDCSessions) ForgetSession(serverID string) {
	f.forgotten = append(f.forgotten, serverID)
}

func TestUpdateServerWithConfig_OIDCSession(t *testing.T) {
	stored := models.ConnectionConfig{
		URI:          "mongodb://host",
		AuthMethod:   models.AuthOIDC,
		OIDCConfig:   &models.OIDCConfig{ProviderURL: "https://idp", ClientID: "vervet"},
		RefreshToken: "stored-token",
	}
	// What the frontend sends: the details, never the token.
	edited := func() models.ConnectionConfig {
		cfg := stored
		cfg.RefreshToken = ""
		oidcCfg := *stored.OIDCConfig
		cfg.OIDCConfig = &oidcCfg
		return cfg
	}
	newService := func(csStore *MockConnectionStringsStore) (*ServerService, *fakeOIDCSessions) {
		sessions := &fakeOIDCSessions{}
		sm := newTestServerService(&mockServerStore{servers: []models.RegisteredServer{{ID: "1", Name: "Server"}}}, csStore)
		sm.oidcSessions = sessions
		return sm, sessions
	}
	storeWithToken := func() *MockConnectionStringsStore {
		return &MockConnectionStringsStore{
			uris:    map[string]string{},
			configs: map[string]models.ConnectionConfig{"1": stored},
		}
	}

	t.Run("a rename keeps the session", func(t *testing.T) {
		csStore := storeWithToken()
		sm, sessions := newService(csStore)

		assert.NoError(t, sm.UpdateServerWithConfig("1", "Renamed", "", "", edited()))

		assert.Empty(t, sessions.forgotten)
		assert.Equal(t, "stored-token", csStore.configs["1"].RefreshToken)
	})

	changes := map[string]func(*models.ConnectionConfig){
		"uri":         func(c *models.ConnectionConfig) { c.URI = "mongodb://new-host" },
		"auth method": func(c *models.ConnectionConfig) { c.AuthMethod = models.AuthPassword },
		"provider":    func(c *models.ConnectionConfig) { c.OIDCConfig.ProviderURL = "https://other-idp" },
	}
	for name, change := range changes {
		t.Run("changing the "+name+" ends the session", func(t *testing.T) {
			csStore := storeWithToken()
			sm, sessions := newService(csStore)
			cfg := edited()
			change(&cfg)

			assert.NoError(t, sm.UpdateServerWithConfig("1", "Server", "", "", cfg))

			assert.Equal(t, []string{"1"}, sessions.forgotten)
			assert.Empty(t, csStore.configs["1"].RefreshToken)
		})
	}

	t.Run("ignores a refresh token sent by the caller", func(t *testing.T) {
		csStore := storeWithToken()
		sm, _ := newService(csStore)
		cfg := edited()
		cfg.URI = "mongodb://new-host"
		cfg.RefreshToken = "from-caller"

		assert.NoError(t, sm.UpdateServerWithConfig("1", "Server", "", "", cfg))

		assert.Empty(t, csStore.configs["1"].RefreshToken)
	})

	t.Run("ends the session when the keyring fails, as the write may still land", func(t *testing.T) {
		sm, sessions := newService(&MockConnectionStringsStore{uris: map[string]string{}, err: errors.New("keyring timed out")})

		assert.Error(t, sm.UpdateServerWithConfig("1", "Server", "", "", edited()))

		assert.Equal(t, []string{"1"}, sessions.forgotten)
	})
}

func TestNewService_NilTokenManagerIsSafe(t *testing.T) {
	sm := NewService(slog.Default(), &mockServerStore{}, &MockConnectionStringsStore{}, nil)

	assert.NotPanics(t, func() { sm.forgetOIDCSession("1") })
}

func TestUpdateServer_EndsOIDCSession(t *testing.T) {
	sessions := &fakeOIDCSessions{}
	sm := newTestServerService(
		&mockServerStore{servers: []models.RegisteredServer{{ID: "1", Name: "Server"}}},
		&MockConnectionStringsStore{uris: map[string]string{}},
	)
	sm.oidcSessions = sessions

	err := sm.UpdateServer("1", "Renamed", "mongodb://new-host", "", "")

	assert.NoError(t, err)
	assert.Equal(t, []string{"1"}, sessions.forgotten)
}
