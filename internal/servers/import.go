package servers

import (
	"encoding/json"
	"fmt"
	"strings"
	"vervet/internal/models"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
)

// ImportResult holds the outcome of an import operation.
type ImportResult struct {
	Created  []models.RegisteredServer `json:"created"`
	Warnings []string                  `json:"warnings"`
}

const maxNameLength = 128

// sanitiseName cleans an entry name: trims whitespace, truncates to maxNameLength,
// and falls back to "Unnamed-{index}" if empty. Returns the sanitised name and
// any warning messages produced.
func sanitiseName(name string, index int) (string, []string) {
	var warnings []string
	original := name

	trimmed := strings.TrimSpace(name)
	trimWarning := trimmed != original

	name = trimmed

	if len(name) > maxNameLength {
		name = name[:maxNameLength]
		warnings = append(warnings, fmt.Sprintf("entry at index %d: name truncated to %d characters", index, maxNameLength))
	}

	if name == "" {
		name = fmt.Sprintf("Unnamed-%d", index)
		warnings = append(warnings, fmt.Sprintf("entry at index %d: name was empty, using %q", index, name))
	} else if trimWarning {
		warnings = append(warnings, fmt.Sprintf("entry at index %d: name trimmed from %q", index, original))
	}

	return name, warnings
}

// ImportServers parses a JSON export file and creates all servers and groups,
// storing connection configs in the keyring. It returns an ImportResult with the list of created RegisteredServers.
func (sm *ServerService) ImportServers(data []byte) (*ImportResult, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var file exportFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if file.Version != 1 {
		return nil, fmt.Errorf("unsupported export format version: %d", file.Version)
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
	var warnings []string

	for i, entry := range file.Servers {
		sanitised, nameWarnings := sanitiseName(entry.Name, i)
		warnings = append(warnings, nameWarnings...)
		entry.Name = sanitised

		parentID := ""
		if entry.Parent != "" {
			parentID = resolveParentPath(entry.Parent, groupPaths, &servers, &created)
		}

		if entry.IsGroup {
			escapedName := escapePathSegment(entry.Name)
			path := escapedName
			if entry.Parent != "" {
				path = rebuildEscapedPath(entry.Parent) + "/" + escapedName
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

	return &ImportResult{Created: created, Warnings: warnings}, nil
}

// splitEscapedPath splits a parent path on unescaped `/` delimiters.
// `\/` is a literal `/` in a name, `\\` is a literal `\`.
func splitEscapedPath(path string) []string {
	var parts []string
	var current strings.Builder
	i := 0
	for i < len(path) {
		if path[i] == '\\' && i+1 < len(path) {
			// Escaped character — consume both bytes literally
			current.WriteByte(path[i])
			current.WriteByte(path[i+1])
			i += 2
		} else if path[i] == '/' {
			parts = append(parts, unescapePathSegment(current.String()))
			current.Reset()
			i++
		} else {
			current.WriteByte(path[i])
			i++
		}
	}
	parts = append(parts, unescapePathSegment(current.String()))
	return parts
}

// unescapePathSegment reverses escapePathSegment: `\/` → `/`, `\\` → `\`.
func unescapePathSegment(s string) string {
	s = strings.ReplaceAll(s, `\/`, `/`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}

// rebuildEscapedPath splits and re-escapes a parent path to normalise it for map lookups.
func rebuildEscapedPath(path string) string {
	parts := splitEscapedPath(path)
	escaped := make([]string, len(parts))
	for i, p := range parts {
		escaped[i] = escapePathSegment(p)
	}
	return strings.Join(escaped, "/")
}

// resolveParentPath takes an escaped slash-delimited path (e.g. "Infra/Databases" or "Dev\/Test")
// and returns the ID of the deepest group, creating any missing intermediate groups as needed.
func resolveParentPath(path string, groupPaths map[string]string, servers *[]models.RegisteredServer, created *[]models.RegisteredServer) string {
	parts := splitEscapedPath(path)
	currentPath := ""
	parentID := ""

	for _, part := range parts {
		escapedPart := escapePathSegment(part)
		if currentPath == "" {
			currentPath = escapedPart
		} else {
			currentPath = currentPath + "/" + escapedPart
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

// buildExistingGroupPaths builds a map of escaped-path -> ID for all existing groups.
// buildParentPath already returns escaped paths, so we only need to escape the group name itself.
func buildExistingGroupPaths(servers []models.RegisteredServer) map[string]string {
	paths := make(map[string]string)
	for _, srv := range servers {
		if !srv.IsGroup {
			continue
		}
		path := buildParentPath(srv.ParentID, servers)
		escapedName := escapePathSegment(srv.Name)
		if path == "" {
			path = escapedName
		} else {
			path = path + "/" + escapedName
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
