package servers

import (
	"encoding/json"
	"testing"
	"vervet/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportServers_BasicImport(t *testing.T) {
	mockStore := &mockServerStore{servers: nil}
	mockCS := &MockConnectionStringsStore{uris: make(map[string]string)}
	svc := newTestServerService(mockStore, mockCS)

	data, _ := json.Marshal(exportFile{
		Version: 1,
		Servers: []exportServerEntry{
			{
				Name:   "My Server",
				Colour: "#ff0000",
				ConnectionConfig: &exportConnectionConfig{
					URI:        "mongodb://localhost:27017",
					AuthMethod: "none",
				},
			},
		},
	})

	result, err := svc.ImportServers(data)

	require.NoError(t, err)
	assert.Len(t, result.Created, 1)
	assert.Equal(t, "My Server", result.Created[0].Name)
	assert.Equal(t, "#ff0000", result.Created[0].Colour)
	assert.NotEmpty(t, result.Created[0].ID)
	assert.False(t, result.Created[0].IsGroup)
}

func TestImportServers_GroupHierarchy(t *testing.T) {
	mockStore := &mockServerStore{servers: nil}
	mockCS := &MockConnectionStringsStore{uris: make(map[string]string)}
	svc := newTestServerService(mockStore, mockCS)

	data, _ := json.Marshal(exportFile{
		Version: 1,
		Servers: []exportServerEntry{
			{
				Name:    "Infra",
				IsGroup: true,
			},
			{
				Name:   "My Server",
				Parent: "Infra",
				ConnectionConfig: &exportConnectionConfig{
					URI:        "mongodb://localhost:27017",
					AuthMethod: "none",
				},
			},
		},
	})

	result, err := svc.ImportServers(data)

	require.NoError(t, err)
	assert.Len(t, result.Created, 2)

	group := result.Created[0]
	server := result.Created[1]
	assert.True(t, group.IsGroup)
	assert.Equal(t, "Infra", group.Name)
	assert.Equal(t, group.ID, server.ParentID)
}

func TestImportServers_AutoCreateMissingGroups(t *testing.T) {
	mockStore := &mockServerStore{servers: nil}
	mockCS := &MockConnectionStringsStore{uris: make(map[string]string)}
	svc := newTestServerService(mockStore, mockCS)

	data, _ := json.Marshal(exportFile{
		Version: 1,
		Servers: []exportServerEntry{
			{
				Name:   "My Server",
				Parent: "Infra/Databases",
				ConnectionConfig: &exportConnectionConfig{
					URI:        "mongodb://localhost:27017",
					AuthMethod: "none",
				},
			},
		},
	})

	result, err := svc.ImportServers(data)

	require.NoError(t, err)
	assert.Len(t, result.Created, 3) // Infra, Databases, My Server

	// Find the groups and server
	byName := make(map[string]*models.RegisteredServer)
	for i := range result.Created {
		byName[result.Created[i].Name] = &result.Created[i]
	}
	infra := byName["Infra"]
	databases := byName["Databases"]
	server := byName["My Server"]

	require.NotNil(t, infra)
	require.NotNil(t, databases)
	require.NotNil(t, server)

	assert.True(t, infra.IsGroup)
	assert.True(t, databases.IsGroup)
	assert.Equal(t, "", infra.ParentID)
	assert.Equal(t, infra.ID, databases.ParentID)
	assert.Equal(t, databases.ID, server.ParentID)
}

func TestImportServers_InvalidJSON(t *testing.T) {
	mockStore := &mockServerStore{servers: nil}
	mockCS := &MockConnectionStringsStore{uris: make(map[string]string)}
	svc := newTestServerService(mockStore, mockCS)

	_, err := svc.ImportServers([]byte("{invalid"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestImportServers_UnsupportedVersion(t *testing.T) {
	mockStore := &mockServerStore{servers: nil}
	mockCS := &MockConnectionStringsStore{uris: make(map[string]string)}
	svc := newTestServerService(mockStore, mockCS)

	data, _ := json.Marshal(exportFile{
		Version: 99,
		Servers: []exportServerEntry{},
	})

	_, err := svc.ImportServers(data)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported export format version")
}

func TestImportServers_MissingName(t *testing.T) {
	mockStore := &mockServerStore{servers: nil}
	mockCS := &MockConnectionStringsStore{uris: make(map[string]string)}
	svc := newTestServerService(mockStore, mockCS)

	data, _ := json.Marshal(exportFile{
		Version: 1,
		Servers: []exportServerEntry{
			{
				Name: "",
				ConnectionConfig: &exportConnectionConfig{
					URI: "mongodb://localhost:27017",
				},
			},
		},
	})

	_, err := svc.ImportServers(data)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "index 0")
	assert.Contains(t, err.Error(), "name")
}

func TestImportServers_DeriveAuthMethod(t *testing.T) {
	mockStore := &mockServerStore{servers: nil}
	mockCS := &MockConnectionStringsStore{uris: make(map[string]string)}
	svc := newTestServerService(mockStore, mockCS)

	data, _ := json.Marshal(exportFile{
		Version: 1,
		Servers: []exportServerEntry{
			{
				Name: "Auth Server",
				ConnectionConfig: &exportConnectionConfig{
					URI: "mongodb://user:pass@localhost:27017",
				},
			},
		},
	})

	result, err := svc.ImportServers(data)

	require.NoError(t, err)
	assert.Len(t, result.Created, 1)
	// The connection config should have been stored - verify via the mock
	assert.Len(t, mockCS.uris, 1)
}

func TestImportServers_NoColourVariants(t *testing.T) {
	mockStore := &mockServerStore{servers: nil}
	mockCS := &MockConnectionStringsStore{uris: make(map[string]string)}
	svc := newTestServerService(mockStore, mockCS)

	data, _ := json.Marshal(exportFile{
		Version: 1,
		Servers: []exportServerEntry{
			{
				Name:   "Server A",
				Colour: "",
				ConnectionConfig: &exportConnectionConfig{
					URI:        "mongodb://localhost:27017",
					AuthMethod: "none",
				},
			},
			{
				Name: "Server B",
				ConnectionConfig: &exportConnectionConfig{
					URI:        "mongodb://localhost:27018",
					AuthMethod: "none",
				},
			},
		},
	})

	result, err := svc.ImportServers(data)

	require.NoError(t, err)
	assert.Len(t, result.Created, 2)
	assert.Equal(t, "", result.Created[0].Colour)
	assert.Equal(t, "", result.Created[1].Colour)
}

func TestImportServers_StoresConnectionConfig(t *testing.T) {
	mockStore := &mockServerStore{servers: nil}
	mockCS := &MockConnectionStringsStore{uris: make(map[string]string)}
	svc := newTestServerService(mockStore, mockCS)

	data, _ := json.Marshal(exportFile{
		Version: 1,
		Servers: []exportServerEntry{
			{
				Name: "Config Server",
				ConnectionConfig: &exportConnectionConfig{
					URI:        "mongodb://localhost:27017",
					AuthMethod: "password",
				},
			},
		},
	})

	result, err := svc.ImportServers(data)

	require.NoError(t, err)
	assert.Len(t, result.Created, 1)

	// Verify the connection config was stored in the keyring mock
	storedURI, ok := mockCS.uris[result.Created[0].ID]
	assert.True(t, ok)
	assert.Equal(t, "mongodb://localhost:27017", storedURI)
}

func TestImportServers_SkipDuplicateServer(t *testing.T) {
	mockStore := &mockServerStore{
		servers: []models.RegisteredServer{
			{ID: "existing-1", Name: "My Server", ParentID: ""},
		},
	}
	mockCS := &MockConnectionStringsStore{uris: map[string]string{
		"existing-1": "mongodb://host:27017",
	}}
	svc := newTestServerService(mockStore, mockCS)

	data, _ := json.Marshal(exportFile{
		Version: 1,
		Servers: []exportServerEntry{
			{
				Name:             "My Server",
				ConnectionConfig: &exportConnectionConfig{URI: "mongodb://host:27017", AuthMethod: "none"},
			},
		},
	})

	result, err := svc.ImportServers(data)
	require.NoError(t, err)
	assert.Len(t, result.Created, 0) // duplicate skipped
}

func TestImportServers_SkipDuplicateGroup(t *testing.T) {
	mockStore := &mockServerStore{
		servers: []models.RegisteredServer{
			{ID: "existing-g1", Name: "Production", IsGroup: true},
		},
	}
	mockCS := &MockConnectionStringsStore{uris: make(map[string]string)}
	svc := newTestServerService(mockStore, mockCS)

	data, _ := json.Marshal(exportFile{
		Version: 1,
		Servers: []exportServerEntry{
			{Name: "Production", IsGroup: true},
		},
	})

	result, err := svc.ImportServers(data)
	require.NoError(t, err)
	assert.Len(t, result.Created, 0) // duplicate skipped
}

func TestImportServers_DuplicateGroupChildrenLinkToExisting(t *testing.T) {
	mockStore := &mockServerStore{
		servers: []models.RegisteredServer{
			{ID: "existing-g1", Name: "Production", IsGroup: true},
		},
	}
	mockCS := &MockConnectionStringsStore{uris: make(map[string]string)}
	svc := newTestServerService(mockStore, mockCS)

	data, _ := json.Marshal(exportFile{
		Version: 1,
		Servers: []exportServerEntry{
			{Name: "Production", IsGroup: true},
			{Name: "New Server", Parent: "Production", ConnectionConfig: &exportConnectionConfig{URI: "mongodb://host:27017"}},
		},
	})

	result, err := svc.ImportServers(data)
	require.NoError(t, err)
	assert.Len(t, result.Created, 1) // only the server, not the duplicate group
	assert.Equal(t, "existing-g1", result.Created[0].ParentID) // linked to existing group
}

func TestImportServers_DifferentURINotDuplicate(t *testing.T) {
	mockStore := &mockServerStore{
		servers: []models.RegisteredServer{
			{ID: "existing-1", Name: "My Server", ParentID: ""},
		},
	}
	mockCS := &MockConnectionStringsStore{uris: map[string]string{
		"existing-1": "mongodb://host-a:27017",
	}}
	svc := newTestServerService(mockStore, mockCS)

	data, _ := json.Marshal(exportFile{
		Version: 1,
		Servers: []exportServerEntry{
			{
				Name:             "My Server",
				ConnectionConfig: &exportConnectionConfig{URI: "mongodb://host-b:27017", AuthMethod: "none"},
			},
		},
	})

	result, err := svc.ImportServers(data)
	require.NoError(t, err)
	assert.Len(t, result.Created, 1) // different URI, not a duplicate
}
