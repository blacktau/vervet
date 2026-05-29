package servers

import (
	"errors"
	"fmt"
	"strings"
	"vervet/internal/models"

	"github.com/google/uuid"
)

// ErrDuplicateGroupName is returned when a create/update would produce two
// sibling groups with the same name (case-insensitive) under the same parent.
var ErrDuplicateGroupName = errors.New("a group with this name already exists in this location")

// CreateGroup creates a new group node and returns its id.
func (sm *ServerService) CreateGroup(parentID, name string) (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	servers, err := sm.store.LoadServers()
	if err != nil {
		return "", fmt.Errorf("failed to create Server Group: %w", err)
	}

	parent, _ := findServer(parentID, servers)

	if parent == nil {
		parentID = ""
	}

	if siblingGroupNameExists(servers, parentID, name, "") {
		return "", ErrDuplicateGroupName
	}

	newServer := models.RegisteredServer{
		ID:       uuid.New().String(),
		Name:     name,
		IsGroup:  true,
		ParentID: parentID,
	}

	servers = append(servers, newServer)
	if err := sm.store.SaveServers(servers); err != nil {
		return "", fmt.Errorf("failed to save new registered server group: %w", err)
	}

	return newServer.ID, nil
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

	if siblingGroupNameExists(servers, parentID, name, groupID) {
		return ErrDuplicateGroupName
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

// siblingGroupNameExists reports whether another group under parentID already
// has the given name (case-insensitive, trimmed). excludeID is skipped to allow
// renaming a group to its own name without conflict.
func siblingGroupNameExists(servers []models.RegisteredServer, parentID, name, excludeID string) bool {
	target := strings.ToLower(strings.TrimSpace(name))
	if target == "" {
		return false
	}
	for _, s := range servers {
		if !s.IsGroup {
			continue
		}
		if s.ID == excludeID {
			continue
		}
		if s.ParentID != parentID {
			continue
		}
		if strings.ToLower(strings.TrimSpace(s.Name)) == target {
			return true
		}
	}
	return false
}
