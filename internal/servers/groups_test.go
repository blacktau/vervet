package servers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateGroup(t *testing.T) {
	t.Run("successful create", func(t *testing.T) {
		mockStore := &MockServerStore{
			servers: []RegisteredServer{
				{ID: "parent", Name: "Parent Group", IsGroup: true},
			},
		}
		sm := newTestServerManager(mockStore, &MockConnectionStringsStore{})

		err := sm.CreateGroup("parent", "New Group")

		assert.NoError(t, err)
		assert.Len(t, mockStore.servers, 2)
		assert.True(t, mockStore.servers[1].IsGroup)
		assert.Equal(t, "New Group", mockStore.servers[1].Name)
		assert.Equal(t, "parent", mockStore.servers[1].ParentID)
	})
}

func TestUpdateGroup(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		mockStore := &MockServerStore{
			servers: []RegisteredServer{
				{ID: "1", Name: "Group 1", IsGroup: true},
			},
		}
		sm := newTestServerManager(mockStore, &MockConnectionStringsStore{})

		err := sm.UpdateGroup("1", "Updated Group", "")

		assert.NoError(t, err)
		assert.Equal(t, "Updated Group", mockStore.servers[0].Name)
	})

	t.Run("group not found", func(t *testing.T) {
		mockStore := &MockServerStore{
			servers: []RegisteredServer{},
		}
		sm := newTestServerManager(mockStore, &MockConnectionStringsStore{})

		err := sm.UpdateGroup("1", "Updated Group", "")

		assert.Error(t, err)
	})

	t.Run("not a group", func(t *testing.T) {
		mockStore := &MockServerStore{
			servers: []RegisteredServer{
				{ID: "1", Name: "Server 1", IsGroup: false},
			},
		}
		sm := newTestServerManager(mockStore, &MockConnectionStringsStore{})

		err := sm.UpdateGroup("1", "Updated Group", "")

		assert.Error(t, err)
	})
}

func TestFindGroup(t *testing.T) {
	servers := []RegisteredServer{
		{ID: "1", Name: "Group 1", IsGroup: true},
		{ID: "2", Name: "Server 1", IsGroup: false},
	}

	t.Run("found", func(t *testing.T) {
		group := findGroup("1", servers)
		assert.NotNil(t, group)
	})

	t.Run("not found", func(t *testing.T) {
		group := findGroup("3", servers)
		assert.Nil(t, group)
	})

	t.Run("not a group", func(t *testing.T) {
		group := findGroup("2", servers)
		assert.Nil(t, group)
	})
}
