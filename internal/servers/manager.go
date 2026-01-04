// Package servers manages the list of registered servers
package servers

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"vervet/internal/connectionStrings"
	"vervet/internal/logging"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
)

type Manager interface {
	Init(ctx context.Context) error
	GetServers() ([]RegisteredServer, error)
	AddServer(parentID, name, uri, colour string) error
	UpdateServer(serverID, name, uri, parentID, colour string) error
	RemoveNode(id string) error
	GetURI(id string) (string, error)
	CreateGroup(parentID, name string) error
	UpdateGroup(groupID, name, parentID string) error
	GetServer(id string) (*RegisteredServerConnection, error)
}

// ServerManagerImpl manages MongoDB server registeredServer strings
type ServerManagerImpl struct {
	ctx               context.Context
	store             ServerStore
	connectionStrings connectionStrings.Store
	mu                sync.RWMutex
	log               *slog.Logger
}

func NewManager(log *slog.Logger) Manager {
	logger := log.With(slog.String(logging.SourceKey, "ServerManager"))
	return &ServerManagerImpl{
		log:               logger,
		mu:                sync.RWMutex{},
		connectionStrings: connectionStrings.NewStore(logger),
	}
}

// Init initializes the manager.
func (sm *ServerManagerImpl) Init(ctx context.Context) error {
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

// GetServers returns the list of registeredServers and groups for the tree of connections
func (sm *ServerManagerImpl) GetServers() ([]RegisteredServer, error) {
	sm.log.Debug("Getting All RegisteredServers")
	registeredServers, err := sm.store.LoadServers()
	if err != nil {
		sm.log.Error("error getting RegisteredServers", slog.Any("error", err))
		return nil, fmt.Errorf("error getting RegisteredServers: %w", err)
	}
	return registeredServers, nil
}

func (sm *ServerManagerImpl) GetServer(id string) (*RegisteredServerConnection, error) {
	log := sm.log.With(slog.String("serverID", id))
	log.Debug("Getting Server Details for Server")
	registeredServers, err := sm.store.LoadServers()
	if err != nil {
		log.Error("error getting RegisteredServers", slog.Any("error", err))
		return nil, fmt.Errorf("error getting RegisteredServers: %w", err)
	}
	for _, server := range registeredServers {
		if server.ID == id {
			uri, err := sm.GetURI(server.ID)
			if err != nil {
				log.Error("error getting URI for server", slog.Any("error", err))
				return nil, fmt.Errorf("error getting URI for server: %w", err)
			}

			return &RegisteredServerConnection{
				RegisteredServer: server,
				URI:              uri,
			}, nil
		}
	}

	log.Error("server not found")
	return nil, fmt.Errorf("server with ID %s not found", id)
}

// AddServer saves the metadata and the URI securely.
func (sm *ServerManagerImpl) AddServer(parentID, name, uri, colour string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	log := sm.log.With(slog.String("parentID", parentID), slog.String("name", name))
	log.Debug("Adding Server")

	servers, err := sm.store.LoadServers()

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

	isCluster := connString.Hosts != nil && len(connString.Hosts) > 1
	isSrv := connString.Scheme == connstring.SchemeMongoDBSRV

	servers = append(servers, RegisteredServer{
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
		log.Error("Failed to save registered server", err)
		return fmt.Errorf("failed to save registered server: %w", err)
	}
	return nil
}

func (sm *ServerManagerImpl) UpdateServer(serverID, name, uri, parentID, colour string) error {
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
func (sm *ServerManagerImpl) RemoveNode(id string) error {
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

func (sm *ServerManagerImpl) GetURI(id string) (string, error) {
	log := sm.log.With(slog.String("serverID", id))

	uri, err := sm.connectionStrings.GetRegisteredServerURI(id)
	if err != nil {
		log.Error("Failed to get uri for registered server", slog.Any("error", err))
		return "", fmt.Errorf("failed to get uri for registered server: %w", err)
	}

	return uri, nil
}

func findServer(serverId string, servers []RegisteredServer) (*RegisteredServer, int) {
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

func hasChildren(parentId string, servers []RegisteredServer) bool {
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
