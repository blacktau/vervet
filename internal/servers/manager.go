// Package servers manages the list of registered servers
package servers

import (
	"context"
	"fmt"
	"log"
	"sync"
	"vervet/internal/common"
	"vervet/internal/configuration"
)

// RegisteredServerManager manages MongoDB connections and interactions.
// This struct will be bound to Wails for frontend access.
type RegisteredServerManager struct {
	ctx        context.Context
	settingsDB *configuration.SettingsDatabase
	mu         sync.RWMutex
}

func NewRegisteredServerManager() *RegisteredServerManager {
	return &RegisteredServerManager{
		mu: sync.RWMutex{},
	}
}

// Init initializes the manager.
func (cm *RegisteredServerManager) Init(ctx context.Context) error {
	database, err := configuration.NewSettingsDatabase()
	if err != nil {
		return err
	}

	cm.settingsDB = database
	return nil
}

// Bindable methods for Folder/Connection Management

// GetRegisteredServers returns the list of connections and folders for the tree of connections
// this is exposed to wails
func (cm *RegisteredServerManager) GetRegisteredServers() common.Result[[]configuration.RegisteredServer] {
	registeredServers, err := cm.settingsDB.GetRegisteredServersTree()
	if err != nil {
		return common.Result[[]configuration.RegisteredServer]{
			IsSuccess: false,
			Error:     fmt.Sprintf("Error getting Registered Servers Tree: %v", err),
		}
	}
	return common.Result[[]configuration.RegisteredServer]{
		IsSuccess: true,
		Data:      registeredServers,
	}
}

// CreateFolder creates a new folder node.
// this is exposed to wails
func (cm *RegisteredServerManager) CreateFolder(name string, parentID int) common.EmptyResult {
	_, err := cm.settingsDB.CreateFolder(name, parentID)
	if err != nil {
		return common.EmptyResult{
			IsSuccess: false,
			Error:     fmt.Sprintf("Failed to create folder: %v", err),
		}
	}

	return common.EmptyResult{
		IsSuccess: true,
	}
}

// SaveRegisterServer saves the metadata and the URI securely.
// this is exposed to wails
func (cm *RegisteredServerManager) SaveRegisterServer(name string, parentID int, uri string) common.EmptyResult {
	connectionID, err := cm.settingsDB.SaveRegisteredServer(name, parentID)
	if err != nil {
		return common.EmptyResult{
			IsSuccess: false,
			Error:     fmt.Sprintf("Failed to save registerd server metadata: %v", err),
		}
	}

	err = configuration.StoreConnectionURI(connectionID, uri)
	if err != nil {
		_ = cm.settingsDB.DeleteNode(int(connectionID))
		return common.EmptyResult{
			IsSuccess: false,
			Error:     fmt.Sprintf("Failed to securely store connection URI: %v", err),
		}
	}

	return common.EmptyResult{
		IsSuccess: true,
	}
}

// RemoveNode removes a folder or connection and its uri
// this is exposed to wails
func (cm *RegisteredServerManager) RemoveNode(id int, isFolder bool) common.EmptyResult {
	err := cm.settingsDB.DeleteNode(id)
	if err != nil {
		return common.EmptyResult{
			IsSuccess: false,
			Error:     fmt.Sprintf("Failed to delete node: %v", err),
		}
	}

	if !isFolder {
		err := configuration.DeleteConnectionURI(id)
		if err != nil {
			log.Printf("Warning: Failed to delete keyring entry for ID %d: %v", id, err)
		}
	}

	return common.EmptyResult{
		IsSuccess: true,
		Error:     "Node deleted successfully",
	}
}

// -- End of wails binding methods
