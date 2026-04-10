package servers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"vervet/internal/models"
)

type exportFile struct {
	Version int                 `json:"version"`
	Servers []exportServerEntry `json:"servers"`
}

type exportConnectionConfig struct {
	URI        string            `json:"uri"`
	AuthMethod string            `json:"authMethod,omitempty"`
	OIDCConfig *exportOIDCConfig `json:"oidcConfig,omitempty"`
}

type exportOIDCConfig struct {
	ProviderURL      string   `json:"providerUrl"`
	ClientID         string   `json:"clientId"`
	Scopes           []string `json:"scopes,omitempty"`
	WorkloadIdentity bool     `json:"workloadIdentity,omitempty"`
}

type exportServerEntry struct {
	Name             string                  `json:"name"`
	Parent           string                  `json:"parent,omitempty"`
	Colour           string                  `json:"colour,omitempty"`
	IsGroup          bool                    `json:"isGroup,omitempty"`
	ConnectionConfig *exportConnectionConfig  `json:"connectionConfig,omitempty"`
}

// ExportServers exports the given server IDs (expanding groups to include descendants)
// as JSON bytes. If includeSensitiveData is false, credentials are stripped from URIs.
func (sm *ServerService) ExportServers(serverIDs []string, includeSensitiveData bool) ([]byte, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	servers, err := sm.store.LoadServers()
	if err != nil {
		return nil, fmt.Errorf("failed to load servers: %w", err)
	}

	// Collect all IDs to export, expanding groups to include descendants.
	ids := make(map[string]bool)
	for _, id := range serverIDs {
		ids[id] = true
		srv, _ := findServer(id, servers)
		if srv != nil && srv.IsGroup {
			collectDescendants(id, servers, ids)
		}
	}

	var entries []exportServerEntry
	for _, srv := range servers {
		if !ids[srv.ID] {
			continue
		}

		entry := exportServerEntry{
			Name:    srv.Name,
			Colour:  srv.Colour,
			IsGroup: srv.IsGroup,
			Parent:  buildParentPath(srv.ParentID, servers),
		}

		if !srv.IsGroup {
			cfg, err := sm.connectionStrings.GetConnectionConfig(srv.ID)
			if err != nil {
				sm.log.Warn("Failed to get connection config for export", slog.String("serverID", srv.ID), slog.Any("error", err))
			} else {
				uri := cfg.URI
				if !includeSensitiveData {
					uri = stripCredentials(uri)
				}

				exportCfg := &exportConnectionConfig{
					URI:        uri,
					AuthMethod: string(cfg.AuthMethod),
				}

				if cfg.OIDCConfig != nil {
					exportCfg.OIDCConfig = &exportOIDCConfig{
						ProviderURL:      cfg.OIDCConfig.ProviderURL,
						ClientID:         cfg.OIDCConfig.ClientID,
						Scopes:           cfg.OIDCConfig.Scopes,
						WorkloadIdentity: cfg.OIDCConfig.WorkloadIdentity,
					}
				}

				entry.ConnectionConfig = exportCfg
			}
		}

		entries = append(entries, entry)
	}

	file := exportFile{
		Version: 1,
		Servers: entries,
	}

	return json.MarshalIndent(file, "", "  ")
}

// stripCredentials removes username and password from a MongoDB URI.
// If parsing fails, the original URI is returned unchanged.
func stripCredentials(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	parsed.User = nil
	return parsed.String()
}

// escapePathSegment escapes `/` and `\` in a group name so it can be safely
// embedded in a `/`-delimited parent path. Escape `\` first to avoid double-escaping.
func escapePathSegment(name string) string {
	name = strings.ReplaceAll(name, `\`, `\\`)
	name = strings.ReplaceAll(name, `/`, `\/`)
	return name
}

// buildParentPath walks up the parent chain and returns a slash-delimited path of ancestor names.
func buildParentPath(parentID string, servers []models.RegisteredServer) string {
	if parentID == "" {
		return ""
	}

	var parts []string
	current := parentID
	for current != "" {
		srv, _ := findServer(current, servers)
		if srv == nil {
			break
		}
		parts = append(parts, escapePathSegment(srv.Name))
		current = srv.ParentID
	}

	// Reverse so the root ancestor is first.
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}

	return strings.Join(parts, "/")
}
