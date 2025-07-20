// Package servers manages the list of registered servers
package servers

import (
	"context"
	"fmt"
	"log"
	"sync"
	"vervet/internal/configuration"
)

// ServerManager manages MongoDB server connection strings
type ServerManager struct {
	ctx        context.Context
	settingsDB *configuration.SettingsDatabase
	mu         sync.RWMutex
}

func NewRegisteredServerManager() *ServerManager {
	return &ServerManager{
		mu: sync.RWMutex{},
	}
}

// Init initializes the manager.
func (cm *ServerManager) Init(ctx context.Context) error {
	database, err := configuration.NewSettingsDatabase()
	if err != nil {
		return err
	}

	cm.settingsDB = database
	return nil
}

// GetRegisteredServers returns the list of connections and folders for the tree of connections
func (cm *ServerManager) GetRegisteredServers() ([]configuration.RegisteredServer, error) {
	registeredServers, err := cm.settingsDB.GetRegisteredServersTree()
	if err != nil {
		return nil, fmt.Errorf("error getting RegisteredServers: %w", err)
	}
	return registeredServers, nil
}

// CreateGroup creates a new folder node.
func (cm *ServerManager) CreateGroup(name string, parentID int) error {
	_, err := cm.settingsDB.CreateFolder(name, parentID)
	if err != nil {
		return fmt.Errorf("failed to create Server Group: %w", err)
	}

	return nil
}

// SaveRegisterServer saves the metadata and the URI securely.
func (cm *ServerManager) SaveRegisterServer(name string, parentID int, uri string) error {
	connectionID, err := cm.settingsDB.SaveRegisteredServer(name, parentID)
	if err != nil {
		return fmt.Errorf("failed to save registered server metadata: %w", err)
	}

	err = configuration.StoreConnectionURI(connectionID, uri)
	if err != nil {
		_ = cm.settingsDB.DeleteNode(int(connectionID))
		return fmt.Errorf("failed to securely store connection URI: %w", err)
	}

	return nil
}

// RemoveNode removes a folder or connection and its uri
func (cm *ServerManager) RemoveNode(id int, isFolder bool) error {
	err := cm.settingsDB.DeleteNode(id)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	if !isFolder {
		err := configuration.DeleteConnectionURI(id)
		if err != nil {
			log.Printf("Warning: Failed to delete keyring entry for ID %d: %v", id, err)
		}
	}

	return nil
}
