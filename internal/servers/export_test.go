package servers

import (
	"encoding/json"
	"errors"
	"testing"
	"vervet/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConnStoreWithConfigs wraps MockConnectionStringsStore but stores actual ConnectionConfig values.
type mockConnStoreWithConfigs struct {
	MockConnectionStringsStore
	configs map[string]models.ConnectionConfig
}

func (m *mockConnStoreWithConfigs) GetConnectionConfig(id string) (models.ConnectionConfig, error) {
	if m.err != nil {
		return models.ConnectionConfig{}, m.err
	}
	cfg, ok := m.configs[id]
	if !ok {
		return models.ConnectionConfig{}, errors.New("config not found")
	}
	return cfg, nil
}

func TestExportServers_FlatList(t *testing.T) {
	store := &mockServerStore{
		servers: []models.RegisteredServer{
			{ID: "s1", Name: "Server Alpha", Colour: "#ff0000"},
			{ID: "s2", Name: "Server Beta", Colour: "#00ff00"},
		},
	}
	csStore := &mockConnStoreWithConfigs{
		MockConnectionStringsStore: MockConnectionStringsStore{uris: make(map[string]string)},
		configs: map[string]models.ConnectionConfig{
			"s1": {URI: "mongodb://localhost:27017", AuthMethod: models.AuthPassword},
			"s2": {URI: "mongodb://localhost:27018", AuthMethod: models.AuthPassword},
		},
	}
	svc := newTestServerService(store, csStore)

	data, err := svc.ExportServers([]string{"s1", "s2"}, true)
	require.NoError(t, err)

	var result exportFile
	require.NoError(t, json.Unmarshal(data, &result))

	assert.Equal(t, 1, result.Version)
	assert.Len(t, result.Servers, 2)
	assert.Equal(t, "Server Alpha", result.Servers[0].Name)
	assert.Equal(t, "#ff0000", result.Servers[0].Colour)
	assert.Equal(t, "Server Beta", result.Servers[1].Name)
	assert.Equal(t, "#00ff00", result.Servers[1].Colour)
	assert.Empty(t, result.Servers[0].Parent)
	assert.Empty(t, result.Servers[1].Parent)
}

func TestExportServers_NestedHierarchy(t *testing.T) {
	store := &mockServerStore{
		servers: []models.RegisteredServer{
			{ID: "g1", Name: "Production", IsGroup: true},
			{ID: "g2", Name: "US East", IsGroup: true, ParentID: "g1"},
			{ID: "s1", Name: "Mongo Primary", ParentID: "g2", Colour: "#0000ff"},
		},
	}
	csStore := &mockConnStoreWithConfigs{
		MockConnectionStringsStore: MockConnectionStringsStore{uris: make(map[string]string)},
		configs: map[string]models.ConnectionConfig{
			"s1": {URI: "mongodb://primary:27017", AuthMethod: models.AuthPassword},
		},
	}
	svc := newTestServerService(store, csStore)

	data, err := svc.ExportServers([]string{"g1", "g2", "s1"}, true)
	require.NoError(t, err)

	var result exportFile
	require.NoError(t, json.Unmarshal(data, &result))

	// Find the server entry
	var serverEntry *exportServerEntry
	for i := range result.Servers {
		if result.Servers[i].Name == "Mongo Primary" {
			serverEntry = &result.Servers[i]
			break
		}
	}
	require.NotNil(t, serverEntry)
	assert.Equal(t, "Production/US East", serverEntry.Parent)
}

func TestExportServers_CredentialStripping(t *testing.T) {
	store := &mockServerStore{
		servers: []models.RegisteredServer{
			{ID: "s1", Name: "Secured Server"},
		},
	}
	csStore := &mockConnStoreWithConfigs{
		MockConnectionStringsStore: MockConnectionStringsStore{uris: make(map[string]string)},
		configs: map[string]models.ConnectionConfig{
			"s1": {URI: "mongodb://admin:secret@localhost:27017/mydb", AuthMethod: models.AuthPassword},
		},
	}
	svc := newTestServerService(store, csStore)

	data, err := svc.ExportServers([]string{"s1"}, false)
	require.NoError(t, err)

	var result exportFile
	require.NoError(t, json.Unmarshal(data, &result))

	require.Len(t, result.Servers, 1)
	require.NotNil(t, result.Servers[0].ConnectionConfig)
	assert.NotContains(t, result.Servers[0].ConnectionConfig.URI, "admin")
	assert.NotContains(t, result.Servers[0].ConnectionConfig.URI, "secret")
	assert.Contains(t, result.Servers[0].ConnectionConfig.URI, "localhost:27017")
}

func TestExportServers_WithCredentials(t *testing.T) {
	store := &mockServerStore{
		servers: []models.RegisteredServer{
			{ID: "s1", Name: "Secured Server"},
		},
	}
	csStore := &mockConnStoreWithConfigs{
		MockConnectionStringsStore: MockConnectionStringsStore{uris: make(map[string]string)},
		configs: map[string]models.ConnectionConfig{
			"s1": {URI: "mongodb://admin:secret@localhost:27017/mydb", AuthMethod: models.AuthPassword},
		},
	}
	svc := newTestServerService(store, csStore)

	data, err := svc.ExportServers([]string{"s1"}, true)
	require.NoError(t, err)

	var result exportFile
	require.NoError(t, json.Unmarshal(data, &result))

	require.Len(t, result.Servers, 1)
	require.NotNil(t, result.Servers[0].ConnectionConfig)
	assert.Contains(t, result.Servers[0].ConnectionConfig.URI, "admin")
	assert.Contains(t, result.Servers[0].ConnectionConfig.URI, "secret")
}

func TestExportServers_GroupExpandsDescendants(t *testing.T) {
	store := &mockServerStore{
		servers: []models.RegisteredServer{
			{ID: "g1", Name: "My Group", IsGroup: true},
			{ID: "s1", Name: "Child Server", ParentID: "g1"},
			{ID: "s2", Name: "Unrelated Server"},
		},
	}
	csStore := &mockConnStoreWithConfigs{
		MockConnectionStringsStore: MockConnectionStringsStore{uris: make(map[string]string)},
		configs: map[string]models.ConnectionConfig{
			"s1": {URI: "mongodb://child:27017", AuthMethod: models.AuthPassword},
			"s2": {URI: "mongodb://unrelated:27017", AuthMethod: models.AuthPassword},
		},
	}
	svc := newTestServerService(store, csStore)

	// Only pass the group ID
	data, err := svc.ExportServers([]string{"g1"}, true)
	require.NoError(t, err)

	var result exportFile
	require.NoError(t, json.Unmarshal(data, &result))

	// Group + child should be included, but not unrelated
	names := make([]string, len(result.Servers))
	for i, s := range result.Servers {
		names[i] = s.Name
	}
	assert.Contains(t, names, "My Group")
	assert.Contains(t, names, "Child Server")
	assert.NotContains(t, names, "Unrelated Server")
}

func TestExportImport_RoundTrip(t *testing.T) {
	// Set up a service with existing servers
	store := &mockServerStore{
		servers: []models.RegisteredServer{
			{ID: "g1", Name: "Production", IsGroup: true},
			{ID: "s1", Name: "Primary", ParentID: "g1", Colour: "#FF0000"},
			{ID: "s2", Name: "Standalone"},
		},
	}
	connStore := &mockConnStoreWithConfigs{
		MockConnectionStringsStore: MockConnectionStringsStore{uris: make(map[string]string)},
		configs: map[string]models.ConnectionConfig{
			"s1": {URI: "mongodb+srv://cluster.example.com", AuthMethod: models.AuthPassword},
			"s2": {URI: "mongodb://localhost:27017", AuthMethod: models.AuthNone},
		},
	}
	exportSvc := newTestServerService(store, connStore)

	// Export all
	data, err := exportSvc.ExportServers([]string{"g1", "s2"}, false)
	require.NoError(t, err)

	// Import into a fresh service
	importStore := &mockServerStore{servers: []models.RegisteredServer{}}
	importConnStore := &MockConnectionStringsStore{uris: make(map[string]string)}
	importSvc := newTestServerService(importStore, importConnStore)

	importResult, err := importSvc.ImportServers(data)
	require.NoError(t, err)
	assert.Len(t, importResult.Created, 3) // group + 2 servers

	// Verify structure is preserved
	byName := make(map[string]*models.RegisteredServer)
	for i := range importResult.Created {
		byName[importResult.Created[i].Name] = &importResult.Created[i]
	}

	assert.True(t, byName["Production"].IsGroup)
	assert.Equal(t, byName["Production"].ID, byName["Primary"].ParentID)
	assert.Equal(t, "#FF0000", byName["Primary"].Colour)
	assert.Empty(t, byName["Standalone"].ParentID)
}

func TestExportServers_NoColourOmitted(t *testing.T) {
	store := &mockServerStore{
		servers: []models.RegisteredServer{
			{ID: "s1", Name: "No Colour Server", Colour: ""},
		},
	}
	csStore := &mockConnStoreWithConfigs{
		MockConnectionStringsStore: MockConnectionStringsStore{uris: make(map[string]string)},
		configs: map[string]models.ConnectionConfig{
			"s1": {URI: "mongodb://localhost:27017", AuthMethod: models.AuthPassword},
		},
	}
	svc := newTestServerService(store, csStore)

	data, err := svc.ExportServers([]string{"s1"}, true)
	require.NoError(t, err)

	// Check raw JSON doesn't contain "colour" key
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))

	var servers []json.RawMessage
	require.NoError(t, json.Unmarshal(raw["servers"], &servers))
	require.Len(t, servers, 1)

	var serverMap map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(servers[0], &serverMap))
	_, hasColour := serverMap["colour"]
	assert.False(t, hasColour, "expected 'colour' key to be omitted from JSON when empty")
}

func TestExportImport_RoundTripWithSlashInName(t *testing.T) {
	store := &mockServerStore{
		servers: []models.RegisteredServer{
			{ID: "g1", Name: "Dev/Test", IsGroup: true},
			{ID: "s1", Name: "Primary", ParentID: "g1", Colour: "#FF0000"},
		},
	}
	connStore := &mockConnStoreWithConfigs{
		MockConnectionStringsStore: MockConnectionStringsStore{uris: make(map[string]string)},
		configs: map[string]models.ConnectionConfig{
			"s1": {URI: "mongodb://localhost:27017", AuthMethod: models.AuthNone},
		},
	}
	exportSvc := newTestServerService(store, connStore)

	data, err := exportSvc.ExportServers([]string{"g1", "s1"}, true)
	require.NoError(t, err)

	importStore := &mockServerStore{servers: []models.RegisteredServer{}}
	importConnStore := &MockConnectionStringsStore{uris: make(map[string]string)}
	importSvc := newTestServerService(importStore, importConnStore)

	importResult, err := importSvc.ImportServers(data)
	require.NoError(t, err)
	assert.Len(t, importResult.Created, 2)

	byName := make(map[string]*models.RegisteredServer)
	for i := range importResult.Created {
		byName[importResult.Created[i].Name] = &importResult.Created[i]
	}

	group := byName["Dev/Test"]
	server := byName["Primary"]
	require.NotNil(t, group, "group 'Dev/Test' should exist with original name")
	require.NotNil(t, server)
	assert.True(t, group.IsGroup)
	assert.Equal(t, group.ID, server.ParentID)
}

func TestBuildParentPath_EscapesSlashInName(t *testing.T) {
	servers := []models.RegisteredServer{
		{ID: "g1", Name: "Dev/Test", IsGroup: true},
		{ID: "g2", Name: "Sub\\Group", IsGroup: true, ParentID: "g1"},
	}
	path := buildParentPath("g2", servers)
	assert.Equal(t, `Dev\/Test/Sub\\Group`, path)
}

func TestBuildParentPath_NoEscapingNeeded(t *testing.T) {
	servers := []models.RegisteredServer{
		{ID: "g1", Name: "Production", IsGroup: true},
		{ID: "g2", Name: "US-East", IsGroup: true, ParentID: "g1"},
	}
	path := buildParentPath("g2", servers)
	assert.Equal(t, "Production/US-East", path)
}
