// Package servers manages the list of registered servers
package servers

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"vervet/internal/connectionStrings"
	"vervet/internal/logging"
	"vervet/internal/models"
	"vervet/internal/oidc"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
)

const ConfigParseErrorEvent = "config-parse-error"

type ServerService struct {
	ctx               context.Context
	log               *slog.Logger
	store             ServerStore
	connectionStrings connectionStrings.Store
	tokenManager      *oidc.TokenManager
	mu                sync.RWMutex
}

func NewService(log *slog.Logger, store ServerStore, connectionStrings connectionStrings.Store, tokenManager *oidc.TokenManager) *ServerService {
	logger := log.With(slog.String(logging.SourceKey, "ServerService"))
	return &ServerService{
		log:               logger,
		mu:                sync.RWMutex{},
		store:             store,
		connectionStrings: connectionStrings,
		tokenManager:      tokenManager,
	}
}

func (sm *ServerService) Init(ctx context.Context) {
	sm.log.Debug("Initializing Server Service")
	sm.ctx = ctx
}

func (sm *ServerService) GetServers() ([]models.RegisteredServer, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sm.log.Debug("Getting All models.RegisteredServers")
	registeredServers, err := sm.store.LoadServers()
	if err != nil {
		sm.log.Error("error getting models.RegisteredServers", slog.Any("error", err))
		if registeredServers != nil {
			// Store returned fallback data (e.g. empty list on parse error)
			// — emit a warning event so the frontend can alert the user
			runtime.EventsEmit(sm.ctx, ConfigParseErrorEvent, err.Error())
			return registeredServers, nil
		}
		return nil, fmt.Errorf("error getting models.RegisteredServers: %w", err)
	}
	return registeredServers, nil
}

func (sm *ServerService) GetServer(id string) (*models.RegisteredServer, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	log := sm.log.With(slog.String("serverID", id))
	log.Debug("Getting Server Configuration for Server")
	registeredServers, err := sm.store.LoadServers()

	if err != nil {
		log.Error("error getting RegisteredServer", slog.Any("error", err))
		return nil, fmt.Errorf("error getting models.RegisteredServers: %w", err)
	}

	server, _ := findServer(id, registeredServers)

	if server != nil {
		return server, nil
	}


	return nil, fmt.Errorf("server with ID %s not found", id)
}

// AddServer saves the metadata and the URI securely.
func (sm *ServerService) AddServer(parentID, name, uri, colour string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	log := sm.log.With(slog.String("parentID", parentID), slog.String("name", name))
	log.Debug("Adding Server")

	servers, err := sm.store.LoadServers()
	if err != nil {
		log.Error("Failed to load registered servers", slog.Any("error", err))
		return fmt.Errorf("failed to load registered servers: %w", err)
	}

	newID := uuid.New().String()

	parent, _ := findServer(parentID, servers)
	if parent == nil {
		parentID = ""
	}

	connString, err := connstring.Parse(uri)
	if err != nil {
		log.Error("Failed to parse connection string", slog.Any("error", err))
		return fmt.Errorf("failed to parse connection string: %w", err)
	}

	isCluster := len(connString.Hosts) > 1
	isSrv := connString.Scheme == connstring.SchemeMongoDBSRV

	servers = append(servers, models.RegisteredServer{
		ID:        newID,
		Name:      name,
		ParentID:  parentID,
		IsGroup:   false,
		IsCluster: isCluster,
		IsSrv:     isSrv,
		Colour:    colour,
	})

	err = sm.connectionStrings.StoreRegisteredServerURI(newID, uri)
	if err != nil {
		log.Error("Failed to securely store registeredServer URI", slog.Any("error", err))
		return fmt.Errorf("failed to securely store registeredServer URI: %w", err)
	}

	err = sm.store.SaveServers(servers)
	if err != nil {
		_ = sm.connectionStrings.DeleteRegisteredServerURI(newID)
		log.Error("Failed to save registered server", slog.Any("error", err))
		return fmt.Errorf("failed to save registered server: %w", err)
	}
	return nil
}

func (sm *ServerService) UpdateServer(serverID, name, uri, parentID, colour string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	log := sm.log.With(
		slog.String("serverID", serverID),
		slog.String("name", name),
		slog.String("parentID", parentID),
		slog.String("colour", colour))

	log.Debug("Updating Server")

	servers, err := sm.store.LoadServers()
	if err != nil {
		log.Error("Failed to load registered servers", slog.Any("error", err))
		return fmt.Errorf("failed to load registered servers: %w", err)
	}

	server, _ := findServer(serverID, servers)
	if server == nil {
		log.Error("Failed to find registered server")
		return fmt.Errorf("failed to find registered server with ID %s", serverID)
	}

	err = sm.connectionStrings.StoreRegisteredServerURI(serverID, uri)
	if err != nil {
		log.Error("Failed to securely store registeredServer URI", slog.Any("error", err))
		return fmt.Errorf("failed to securely store registeredServer URI: %w", err)
	}

	server.Name = name
	server.ParentID = parentID
	server.Colour = colour

	err = sm.store.SaveServers(servers)
	if err != nil {
		log.Error("Failed to save registered server metadata", slog.Any("error", err))
		return fmt.Errorf("failed to save registered server metadata: %w", err)
	}

	return nil
}

func (sm *ServerService) AddServerWithConfig(parentID, name, colour string, cfg models.ConnectionConfig) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	log := sm.log.With(slog.String("parentID", parentID), slog.String("name", name))
	log.Debug("Adding Server")

	servers, err := sm.store.LoadServers()
	if err != nil {
		log.Error("Failed to load registered servers", slog.Any("error", err))
		return fmt.Errorf("failed to load registered servers: %w", err)
	}

	newID := uuid.New().String()

	parent, _ := findServer(parentID, servers)
	if parent == nil {
		parentID = ""
	}

	connString, err := connstring.Parse(cfg.URI)
	if err != nil {
		log.Error("Failed to parse connection string", slog.Any("error", err))
		return fmt.Errorf("failed to parse connection string: %w", err)
	}

	isCluster := len(connString.Hosts) > 1
	isSrv := connString.Scheme == connstring.SchemeMongoDBSRV

	servers = append(servers, models.RegisteredServer{
		ID:        newID,
		Name:      name,
		ParentID:  parentID,
		IsGroup:   false,
		IsCluster: isCluster,
		IsSrv:     isSrv,
		Colour:    colour,
	})

	err = sm.connectionStrings.StoreConnectionConfig(newID, cfg)
	if err != nil {
		log.Error("Failed to securely store connection config", slog.Any("error", err))
		return fmt.Errorf("failed to securely store connection config: %w", err)
	}

	err = sm.store.SaveServers(servers)
	if err != nil {
		_ = sm.connectionStrings.DeleteRegisteredServerURI(newID)
		log.Error("Failed to save registered server", slog.Any("error", err))
		return fmt.Errorf("failed to save registered server: %w", err)
	}
	return nil
}

func (sm *ServerService) UpdateServerWithConfig(serverID, name, parentID, colour string, cfg models.ConnectionConfig) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	log := sm.log.With(
		slog.String("serverID", serverID),
		slog.String("name", name),
		slog.String("parentID", parentID),
		slog.String("colour", colour))

	log.Debug("Updating Server")

	servers, err := sm.store.LoadServers()
	if err != nil {
		log.Error("Failed to load registered servers", slog.Any("error", err))
		return fmt.Errorf("failed to load registered servers: %w", err)
	}

	server, _ := findServer(serverID, servers)
	if server == nil {
		log.Error("Failed to find registered server")
		return fmt.Errorf("failed to find registered server with ID %s", serverID)
	}

	err = sm.connectionStrings.StoreConnectionConfig(serverID, cfg)
	if err != nil {
		log.Error("Failed to securely store connection config", slog.Any("error", err))
		return fmt.Errorf("failed to securely store connection config: %w", err)
	}

	server.Name = name
	server.ParentID = parentID
	server.Colour = colour

	err = sm.store.SaveServers(servers)
	if err != nil {
		log.Error("Failed to save registered server metadata", slog.Any("error", err))
		return fmt.Errorf("failed to save registered server metadata: %w", err)
	}

	return nil
}

func (sm *ServerService) GetConnectionConfig(serverID string) (models.ConnectionConfig, error) {
	return sm.connectionStrings.GetConnectionConfig(serverID)
}

// RemoveNode removes a group or registeredServer and its uri
func (sm *ServerService) RemoveNode(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	log := sm.log.With(slog.String("serverID", id))

	servers, err := sm.store.LoadServers()
	if err != nil {
		log.Error("Failed to load registered servers", slog.Any("error", err))
		return fmt.Errorf("failed to load registered servers: %w", err)
	}

	node, _ := findServer(id, servers)
	if node == nil {
		log.Error("Failed to find registered server with ID")
		return fmt.Errorf("failed to find registered server with ID %s", id)
	}

	removeIDs := map[string]bool{id: true}
	if node.IsGroup {
		collectDescendants(id, servers, removeIDs)
	}

	var remaining []models.RegisteredServer
	for _, s := range servers {
		if !removeIDs[s.ID] {
			remaining = append(remaining, s)
		}
	}

	err = sm.store.SaveServers(remaining)
	if err != nil {
		log.Error("Failed to delete node", slog.Any("error", err))
		return fmt.Errorf("failed to delete node: %w", err)
	}

	for rid := range removeIDs {
		removed, _ := findServer(rid, servers)
		if removed != nil && !removed.IsGroup {
			if err := sm.connectionStrings.DeleteRegisteredServerURI(rid); err != nil {
				log.Error("Failed to delete keyring entry for server", slog.String("serverID", rid), slog.Any("error", err))
			}
			if sm.tokenManager != nil {
				sm.tokenManager.CleanupServer(rid)
			}
		}
	}

	return nil
}

func (sm *ServerService) GetURI(id string) (string, error) {
	log := sm.log.With(slog.String("serverID", id))

	uri, err := sm.connectionStrings.GetRegisteredServerURI(id)
	if err != nil {
		log.Error("Failed to get uri for registered server", slog.Any("error", err))
		return "", fmt.Errorf("failed to get uri for registered server: %w", err)
	}

	return uri, nil
}

func findServer(serverID string, servers []models.RegisteredServer) (*models.RegisteredServer, int) {
	if len(servers) == 0 {
		return nil, -1
	}

	for idx, server := range servers {
		if server.ID == serverID {
			return &servers[idx], idx
		}
	}

	return nil, -1
}

func hasChildren(parentID string, servers []models.RegisteredServer) bool {
	for _, server := range servers {
		if server.ParentID == parentID {
			return true
		}
	}

	return false
}

func collectDescendants(parentID string, servers []models.RegisteredServer, ids map[string]bool) {
	for _, server := range servers {
		if server.ParentID == parentID {
			ids[server.ID] = true
			if server.IsGroup {
				collectDescendants(server.ID, servers, ids)
			}
		}
	}
}