// Package servers manages the list of registered servers
package servers

import (
	"context"
	"fmt"
	"log"
	"sync"
	"vervet/internal/connectionStrings"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Manager interface {
	Init(ctx context.Context) error
	GetServers() ([]RegisteredServer, error)
	AddServer(parentID, name, uri string) error
	UpdateServer(serverID, name, uri, parentID string) error
	RemoveNode(id string) error
	GetURI(id string) (string, error)
	CreateGroup(parentID string, name string) error
	UpdateGroup(groupID string, name string) error
	GetServer(id string) (*RegisteredServer, error)
}

// ServerManagerImpl manages MongoDB server registeredServer strings
type ServerManagerImpl struct {
	ctx               context.Context
	store             ServerStore
	connectionStrings connectionStrings.Store
	mu                sync.RWMutex
	log               logger.Logger
}

func NewManager(log logger.Logger) Manager {
	return &ServerManagerImpl{
		log:               log,
		mu:                sync.RWMutex{},
		connectionStrings: connectionStrings.NewStore(),
	}
}

// Init initializes the manager.
func (sm *ServerManagerImpl) Init(ctx context.Context) error {
	sm.ctx = ctx

	store, err := NewServerStore(sm.log)

	if err != nil {
		_, _ = runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Unrecoverable Error in Vervet",
			Message: fmt.Sprintf("Error: %v", err),
		})
		panic(err)
	}

	sm.store = store
	return nil
}

// GetServers returns the list of registeredServers and groups for the tree of connections
func (sm *ServerManagerImpl) GetServers() ([]RegisteredServer, error) {
	registeredServers, err := sm.store.LoadServers()
	if err != nil {
		return nil, fmt.Errorf("error getting RegisteredServers: %w", err)
	}
	return registeredServers, nil
}

func (sm *ServerManagerImpl) GetServer(id string) (*RegisteredServer, error) {
	registeredServers, err := sm.store.LoadServers()
	if err != nil {
		return nil, fmt.Errorf("error getting RegisteredServers: %w", err)
	}
	for _, server := range registeredServers {
		if server.ID == id {
			return &server, nil
		}
	}
	return nil, fmt.Errorf("server with ID %s not found", id)
}

// AddServer saves the metadata and the URI securely.
func (sm *ServerManagerImpl) AddServer(parentID, name, uri string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	servers, err := sm.store.LoadServers()

	newId := uuid.New().String()

	parent, _ := findServer(parentID, servers)
	if parent == nil {
		return fmt.Errorf("failed to find parent group for ID %s", parentID)
	}

	servers = append(servers, RegisteredServer{
		ID:       newId,
		Name:     name,
		ParentID: parentID,
		IsGroup:  false,
	})

	err = sm.connectionStrings.StoreRegisteredServerURI(newId, uri)
	if err != nil {
		return fmt.Errorf("failed to securely store registeredServer URI: %w", err)
	}

	err = sm.store.SaveServers(servers)
	if err != nil {
		_ = sm.connectionStrings.DeleteRegisteredServerURI(newId)
		return fmt.Errorf("failed to save registered servers: %w", err)
	}
	return nil
}

func (sm *ServerManagerImpl) UpdateServer(serverID, name, uri, parentID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	servers, err := sm.store.LoadServers()
	if err != nil {
		return fmt.Errorf("failed to load registered servers: %w", err)
	}

	server, _ := findServer(serverID, servers)
	if server == nil {
		return fmt.Errorf("failed to find registered server with ID %s", serverID)
	}

	server.Name = name
	server.ParentID = parentID

	err = sm.store.SaveServers(servers)
	if err != nil {
		return fmt.Errorf("failed to save registered server metadata: %w", err)
	}

	err = sm.connectionStrings.StoreRegisteredServerURI(serverID, uri)
	if err != nil {
		return fmt.Errorf("failed to securely store registeredServer URI: %w", err)
	}

	return nil
}

// RemoveNode removes a group or registeredServer and its uri
func (sm *ServerManagerImpl) RemoveNode(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	servers, err := sm.store.LoadServers()
	if err != nil {
		return fmt.Errorf("failed to load registered servers: %w", err)
	}

	node, idx := findServer(id, servers)
	if node == nil {
		return fmt.Errorf("failed to find registered server with ID %s", id)
	}

	if node.IsGroup && hasChildren(node.ID, servers) {
		return fmt.Errorf("cannot remove node %s from registered servers: still contains children", node.ID)
	}

	servers = append(servers[:idx], servers[idx+1:]...)

	err = sm.store.SaveServers(servers)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	if !node.IsGroup {
		err := sm.connectionStrings.DeleteRegisteredServerURI(id)
		if err != nil {
			log.Printf("Warning: Failed to delete keyring entry for ID %s: %v", id, err)
		}
	}

	return nil
}

func (sm *ServerManagerImpl) GetURI(id string) (string, error) {
	uri, err := sm.connectionStrings.GetRegisteredServerURI(id)
	if err != nil {
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
