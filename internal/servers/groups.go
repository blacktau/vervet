package servers

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// CreateGroup creates a new group node.
func (sm *ServerManagerImpl) CreateGroup(parentID, name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	log := sm.log.With(slog.String("parentID", parentID), slog.String("name", name))
	log.Debug("Creating Server Group")

	servers, err := sm.store.LoadServers()
	if err != nil {
		log.Error(
			"Failed to create Server Group",
			slog.Any("error", err))

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
		log.Error(
			"Failed to save new registered server group",
			slog.Any("error", err))
		return fmt.Errorf("failed to save new registered server group: %w", err)
	}

	log.Debug("Successfully created Server Group")

	return nil
}

func (sm *ServerManagerImpl) UpdateGroup(groupID, name, parentID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	log := sm.log.With(slog.String("groupID", groupID), slog.String("name", name), slog.String("parentID", parentID))
	log.Debug("Updating group")

	servers, err := sm.store.LoadServers()
	if err != nil {
		log.Error("Failed to update group", slog.Any("error", err))
		return fmt.Errorf("failed to update group: %w", err)
	}

	parent, _ := findServer(parentID, servers)
	if parent == nil {
		parentID = ""
	}

	group := findGroup(groupID, servers)
	if group == nil {
		log.Error("Failed to find group")
		return fmt.Errorf("failed to find group for ID %s", groupID)
	}

	group.Name = name
	group.ParentID = parentID

	err = sm.store.SaveServers(servers)

	if err != nil {
		log.Error("Failed to update server group", slog.Any("error", err))
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
