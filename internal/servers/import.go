package servers

import (
	"encoding/json"
	"fmt"
	"strings"
	"vervet/internal/models"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
)

// ImportServers parses a JSON export file and creates all servers and groups,
// storing connection configs in the keyring. It returns the list of created RegisteredServers.
func (sm *ServerService) ImportServers(data []byte) ([]models.RegisteredServer, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var file exportFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if file.Version != 1 {
		return nil, fmt.Errorf("unsupported export format version: %d", file.Version)
	}

	for i, entry := range file.Servers {
		if strings.TrimSpace(entry.Name) == "" {
			return nil, fmt.Errorf("server at index %d has an empty name", i)
		}
	}

	servers, err := sm.store.LoadServers()
	if err != nil {
		return nil, fmt.Errorf("failed to load servers: %w", err)
	}

	// Map of group path -> ID for resolving parent references.
	groupPaths := make(map[string]string)

	// Index existing groups and servers for duplicate detection.
	// Register existing groups in groupPaths so parent resolution finds them.
	existingGroupPaths := buildExistingGroupPaths(servers)
	for path, id := range existingGroupPaths {
		groupPaths[path] = id
	}

	existingServerURIs := sm.buildExistingServerKeys(servers)

	var created []models.RegisteredServer

	for _, entry := range file.Servers {
		parentID := ""
		if entry.Parent != "" {
			parentID = resolveParentPath(entry.Parent, groupPaths, &servers, &created)
		}

		if entry.IsGroup {
			path := entry.Name
			if entry.Parent != "" {
				path = entry.Parent + "/" + entry.Name
			}

			// Skip duplicate groups (same name and parent path).
			if _, exists := groupPaths[path]; exists {
				continue
			}

			newID := uuid.New().String()
			srv := models.RegisteredServer{
				ID:       newID,
				Name:     entry.Name,
				ParentID: parentID,
				Colour:   entry.Colour,
				IsGroup:  true,
			}
			servers = append(servers, srv)
			created = append(created, srv)
			groupPaths[path] = newID
		} else {
			cfg := models.ConnectionConfig{}
			var isCluster, isSrv bool

			if entry.ConnectionConfig != nil {
				authMethod := deriveAuthMethod(entry.ConnectionConfig)
				cfg = models.ConnectionConfig{
					URI:        entry.ConnectionConfig.URI,
					AuthMethod: authMethod,
				}

				if entry.ConnectionConfig.OIDCConfig != nil {
					cfg.OIDCConfig = &models.OIDCConfig{
						ProviderURL:      entry.ConnectionConfig.OIDCConfig.ProviderURL,
						ClientID:         entry.ConnectionConfig.OIDCConfig.ClientID,
						Scopes:           entry.ConnectionConfig.OIDCConfig.Scopes,
						WorkloadIdentity: entry.ConnectionConfig.OIDCConfig.WorkloadIdentity,
					}
				}

				cs, parseErr := connstring.Parse(entry.ConnectionConfig.URI)
				if parseErr == nil {
					isCluster = len(cs.Hosts) > 1
					isSrv = cs.Scheme == connstring.SchemeMongoDBSRV
				}

				// Skip duplicate servers (same name, parent, and URI).
				serverKey := parentID + "\x00" + entry.Name + "\x00" + cfg.URI
				if existingServerURIs[serverKey] {
					continue
				}
				existingServerURIs[serverKey] = true
			}

			newID := uuid.New().String()
			srv := models.RegisteredServer{
				ID:        newID,
				Name:      entry.Name,
				ParentID:  parentID,
				Colour:    entry.Colour,
				IsGroup:   false,
				IsCluster: isCluster,
				IsSrv:     isSrv,
			}
			servers = append(servers, srv)
			created = append(created, srv)

			if entry.ConnectionConfig != nil {
				if err := sm.connectionStrings.StoreConnectionConfig(newID, cfg); err != nil {
					return nil, fmt.Errorf("failed to store connection config for %q: %w", entry.Name, err)
				}
			}
		}
	}

	if err := sm.store.SaveServers(servers); err != nil {
		return nil, fmt.Errorf("failed to save servers: %w", err)
	}

	return created, nil
}

// resolveParentPath takes a slash-delimited path (e.g. "Infra/Databases") and returns
// the ID of the deepest group, creating any missing intermediate groups as needed.
func resolveParentPath(path string, groupPaths map[string]string, servers *[]models.RegisteredServer, created *[]models.RegisteredServer) string {
	parts := strings.Split(path, "/")
	currentPath := ""
	parentID := ""

	for _, part := range parts {
		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}

		if id, ok := groupPaths[currentPath]; ok {
			parentID = id
			continue
		}

		// Create missing group.
		newID := uuid.New().String()
		group := models.RegisteredServer{
			ID:       newID,
			Name:     part,
			ParentID: parentID,
			IsGroup:  true,
		}
		*servers = append(*servers, group)
		*created = append(*created, group)
		groupPaths[currentPath] = newID
		parentID = newID
	}

	return parentID
}

// buildExistingGroupPaths builds a map of path -> ID for all existing groups.
func buildExistingGroupPaths(servers []models.RegisteredServer) map[string]string {
	paths := make(map[string]string)
	for _, srv := range servers {
		if !srv.IsGroup {
			continue
		}
		path := buildParentPath(srv.ParentID, servers)
		if path == "" {
			path = srv.Name
		} else {
			path = path + "/" + srv.Name
		}
		paths[path] = srv.ID
	}
	return paths
}

// buildExistingServerKeys builds a set of "parentID\x00name\x00uri" keys for existing servers.
func (sm *ServerService) buildExistingServerKeys(servers []models.RegisteredServer) map[string]bool {
	keys := make(map[string]bool)
	for _, srv := range servers {
		if srv.IsGroup {
			continue
		}
		cfg, err := sm.connectionStrings.GetConnectionConfig(srv.ID)
		if err != nil {
			continue
		}
		key := srv.ParentID + "\x00" + srv.Name + "\x00" + cfg.URI
		keys[key] = true
	}
	return keys
}

// deriveAuthMethod returns the auth method from the export config. If not specified,
// it parses the URI to determine whether credentials are present.
func deriveAuthMethod(cfg *exportConnectionConfig) models.AuthMethod {
	if cfg.AuthMethod != "" {
		return models.AuthMethod(cfg.AuthMethod)
	}

	cs, err := connstring.Parse(cfg.URI)
	if err != nil {
		return models.AuthNone
	}

	if cs.Username != "" {
		return models.AuthPassword
	}

	return models.AuthNone
}
