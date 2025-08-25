// Package servers manages the list of registered servers
package servers

import (
	"context"
	"fmt"
	"log"
	"sync"
	"vervet/internal/configuration"

	"github.com/wailsapp/wails/v2/pkg/logger"
)

// ServerManager manages MongoDB server registeredServer strings
type ServerManager struct {
	ctx        context.Context
	settingsDB *configuration.SettingsDatabase
	mu         sync.RWMutex
	log        logger.Logger
}

func NewRegisteredServerManager(log logger.Logger) *ServerManager {
	return &ServerManager{
		log: log,
		mu:  sync.RWMutex{},
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

// GetRegisteredServers returns the list of registeredServers and groups for the tree of connections
func (cm *ServerManager) GetRegisteredServers() ([]configuration.RegisteredServer, error) {
	registeredServers, err := cm.settingsDB.GetRegisteredServersTree()
	if err != nil {
		return nil, fmt.Errorf("error getting RegisteredServers: %w", err)
	}
	return registeredServers, nil
}

// CreateGroup creates a new group node.
func (cm *ServerManager) CreateGroup(parentID int, name string) error {
	_, err := cm.settingsDB.CreateGroup(parentID, name)
	if err != nil {
		return fmt.Errorf("failed to create Server Group: %w", err)
	}

	return nil
}

func (cm *ServerManager) UpdateGroup(groupID, parentID int, name string) error {
	err := cm.settingsDB.UpdateGroup(groupID, parentID, name)
	if err != nil {
		return fmt.Errorf("failed to update server group: %w", err)
	}

	return nil
}

// AddRegisterServer saves the metadata and the URI securely.
func (cm *ServerManager) AddRegisterServer(parentID int, name, uri string) error {
	registeredServerID, err := cm.settingsDB.SaveRegisteredServer(parentID, name)
	if err != nil {
		return fmt.Errorf("failed to save registered server metadata: %w", err)
	}

	err = configuration.StoreRegisteredServerURI(registeredServerID, uri)
	if err != nil {
		_ = cm.settingsDB.DeleteNode(int(registeredServerID))
		return fmt.Errorf("failed to securely store registeredServer URI: %w", err)
	}

	return nil
}

func (cm *ServerManager) UpdateRegisterServer(registeredServerID, parentID int, name, uri string) error {
	err := cm.settingsDB.UpdateRegisteredServer(registeredServerID, parentID, name)
	if err != nil {
		return fmt.Errorf("failed to save registered server metadata: %w", err)
	}

	err = configuration.StoreRegisteredServerURI(registeredServerID, uri)
	if err != nil {
		_ = cm.settingsDB.DeleteNode(int(registeredServerID))
		return fmt.Errorf("failed to securely store registeredServer URI: %w", err)
	}

	return nil
}

// RemoveNode removes a group or registeredServer and its uri
func (cm *ServerManager) RemoveNode(id int, isgroup bool) error {
	err := cm.settingsDB.DeleteNode(id)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	if !isgroup {
		err := configuration.DeleteRegisteredServerURI(id)
		if err != nil {
			log.Printf("Warning: Failed to delete keyring entry for ID %d: %v", id, err)
		}
	}

	return nil
}

func (cm *ServerManager) GetURI(id int) (string, error) {
	uri, err := configuration.GetRegisteredServerURI(id)
	if err != nil {
		return "", fmt.Errorf("failed to get uri for registered server: %w", err)
	}

	return uri, nil
}
