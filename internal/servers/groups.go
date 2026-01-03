package servers

import (
	"fmt"

	"github.com/google/uuid"
)

// CreateGroup creates a new group node.
func (sm *ServerManagerImpl) CreateGroup(parentID string, name string) error {
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

	newServer := RegisteredServer{
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

func (sm *ServerManagerImpl) UpdateGroup(groupID string, name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	servers, err := sm.store.LoadServers()
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	group := findGroup(groupID, servers)
	if group == nil {
		return fmt.Errorf("failed to find group for ID %s", groupID)
	}

	group.Name = name
	err = sm.store.SaveServers(servers)

	if err != nil {
		return fmt.Errorf("failed to update server group: %w", err)
	}

	return nil
}

func findGroup(groupID string, servers []RegisteredServer) *RegisteredServer {
	node, _ := findServer(groupID, servers)
	if node == nil || !node.IsGroup {
		return nil
	}

	return node
}
