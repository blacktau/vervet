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

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
)

type ServerManager struct {
	ctx               context.Context
	log               *slog.Logger
	store             ServerStore
	connectionStrings connectionStrings.Store
	mu                sync.RWMutex
}

func NewManager(log *slog.Logger) *ServerManager {
	logger := log.With(slog.String(logging.SourceKey, "ServerManager"))
	return &ServerManager{
		log:               logger,
		mu:                sync.RWMutex{},
		connectionStrings: connectionStrings.NewStore(logger),
	}
}

func (sm *ServerManager) Init(ctx context.Context) error {
	sm.log.Debug("Initializing Server Manager")
	sm.ctx = ctx

	store, err := NewServerStore(sm.log)

	if err != nil {
		_, _ = runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Unrecoverable Error in Vervet",
			Message: fmt.Sprintf("Error: %v", err),
		})
		sm.log.Error("Failed to initialize Server Store", slog.Any("error", err))
		panic(err)
	}

	sm.store = store
	return nil
}

func (sm *ServerManager) GetServers() ([]models.RegisteredServer, error) {
	sm.log.Debug("Getting All models.RegisteredServers")
	registeredServers, err := sm.store.LoadServers()
	if err != nil {
		sm.log.Error("error getting models.RegisteredServers", slog.Any("error", err))
		return nil, fmt.Errorf("error getting models.RegisteredServers: %w", err)
	}
	return registeredServers, nil
}

func (sm *ServerManager) GetServerConfiguration(id string) (*models.RegisteredServerConnection, error) {
	log := sm.log.With(slog.String("serverID", id))
	log.Debug("Getting Server Configuration for Server")
	registeredServers, err := sm.store.LoadServers()
	if err != nil {
		log.Error("error getting RegisteredServer", slog.Any("error", err))
		return nil, fmt.Errorf("error getting models.RegisteredServers: %w", err)
	}

	server, _ := findServer(id, registeredServers)


	if server == nil {
		return nil, fmt.Errorf("server with ID %s not found", id)
	}

	uri, err := sm.GetURI(server.ID)
	if err != nil {
		log.Error("error getting URI for server", slog.Any("error", err))
		return nil, fmt.Errorf("error getting URI for server: %w", err)
	}

	return &models.RegisteredServerConnection{
		RegisteredServer: 		*server,
		URI:                     uri,
	}, nil
}

func (sm *ServerManager) GetServer(id string) (*models.RegisteredServer, error) {
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
func (sm *ServerManager) AddServer(parentID, name, uri, colour string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	log := sm.log.With(slog.String("parentID", parentID), slog.String("name", name))
	log.Debug("Adding Server")

	servers, err := sm.store.LoadServers()
	if err != nil {
		log.Error("Failed to load registered servers", slog.Any("error", err))
		return fmt.Errorf("failed to load registered servers: %w", err)
	}

	newId := uuid.New().String()

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
		ID:        newId,
		Name:      name,
		ParentID:  parentID,
		IsGroup:   false,
		IsCluster: isCluster,
		IsSrv:     isSrv,
		Colour:    colour,
	})

	err = sm.connectionStrings.StoreRegisteredServerURI(newId, uri)
	if err != nil {
		log.Error("Failed to securely store registeredServer URI", slog.Any("error", err))
		return fmt.Errorf("failed to securely store registeredServer URI: %w", err)
	}

	err = sm.store.SaveServers(servers)
	if err != nil {
		_ = sm.connectionStrings.DeleteRegisteredServerURI(newId)
		log.Error("Failed to save registered server", slog.Any("error", err))
		return fmt.Errorf("failed to save registered server: %w", err)
	}
	return nil
}

func (sm *ServerManager) UpdateServer(serverID, name, uri, parentID, colour string) error {
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

	server.Name = name
	server.ParentID = parentID
	server.Colour = colour

	err = sm.store.SaveServers(servers)
	if err != nil {
		log.Error("Failed to save registered server metadata", slog.Any("error", err))
		return fmt.Errorf("failed to save registered server metadata: %w", err)
	}

	err = sm.connectionStrings.StoreRegisteredServerURI(serverID, uri)
	if err != nil {
		log.Error("Failed to securely store registeredServer URI", slog.Any("error", err))
		return fmt.Errorf("failed to securely store registeredServer URI: %w", err)
	}

	return nil
}

// RemoveNode removes a group or registeredServer and its uri
func (sm *ServerManager) RemoveNode(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	log := sm.log.With(slog.String("serverID", id))

	servers, err := sm.store.LoadServers()
	if err != nil {
		log.Error("Failed to load registered servers", slog.Any("error", err))
		return fmt.Errorf("failed to load registered servers: %w", err)
	}

	node, idx := findServer(id, servers)
	if node == nil {
		log.Error("Failed to find registered server with ID")
		return fmt.Errorf("failed to find registered server with ID %s", id)
	}

	if node.IsGroup && hasChildren(node.ID, servers) {
		log.Error("Cannot remove node from registered servers: still contains children")
		return fmt.Errorf("cannot remove node %s from registered servers: still contains children", node.ID)
	}

	servers = append(servers[:idx], servers[idx+1:]...)

	err = sm.store.SaveServers(servers)
	if err != nil {
		log.Error("Failed to delete node", slog.Any("error", err))
		return fmt.Errorf("failed to delete node: %w", err)
	}

	if !node.IsGroup {
		err := sm.connectionStrings.DeleteRegisteredServerURI(id)
		if err != nil {
			log.Error("Failed to delete keyring entry for server", slog.Any("error", err))
		}
	}

	return nil
}

func (sm *ServerManager) GetURI(id string) (string, error) {
	log := sm.log.With(slog.String("serverID", id))

	uri, err := sm.connectionStrings.GetRegisteredServerURI(id)
	if err != nil {
		log.Error("Failed to get uri for registered server", slog.Any("error", err))
		return "", fmt.Errorf("failed to get uri for registered server: %w", err)
	}

	return uri, nil
}

func findServer(serverId string, servers []models.RegisteredServer) (*models.RegisteredServer, int) {
	if len(servers) == 0 {
		return nil, -1
	}

	for idx, server := range servers {
		if server.ID == serverId {
			return &servers[idx], idx
		}
	}

	return nil, -1
}

func hasChildren(parentId string, servers []models.RegisteredServer) bool {
	if len(servers) == 0 {
		return false
	}

	for _, server := range servers {
		if server.ParentID == parentId {
			return true
		}
	}

	return false
}