package servers

import (
	"fmt"
	"vervet/internal/models"

	"github.com/google/uuid"
)

// CreateGroup creates a new group node.
func (sm *ServerService) CreateGroup(parentID, name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	servers, err := sm.store.LoadServers()
	if err != nil {
		return fmt.Errorf("failed to create Server Group: %w", err)
	}

	parent, _ := findServer(parentID, servers)

	if parent == nil {
		parentID = ""
	}

	newServer := models.RegisteredServer{
		ID:       uuid.New().String(),
		Name:     name,
		IsGroup:  true,
		ParentID: parentID,
	}

	servers = append(servers, newServer)
	err = sm.store.SaveServers(servers)
	if err != nil {
		return fmt.Errorf("failed to save new registered server group: %w", err)
	}

	return nil
}

func (sm *ServerService) UpdateGroup(groupID, name, parentID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	servers, err := sm.store.LoadServers()
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	parent, _ := findServer(parentID, servers)
	if parent == nil {
		parentID = ""
	}

	group := findGroup(groupID, servers)
	if group == nil {
		return fmt.Errorf("failed to find group for ID %s", groupID)
	}

	group.Name = name
	group.ParentID = parentID

	err = sm.store.SaveServers(servers)

	if err != nil {
		return fmt.Errorf("failed to update server group: %w", err)
	}

	return nil
}

func findGroup(groupID string, servers []models.RegisteredServer) *models.RegisteredServer {
	node, _ := findServer(groupID, servers)
	if node == nil || !node.IsGroup {
		return nil
	}

	return node
}
