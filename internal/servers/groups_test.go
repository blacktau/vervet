package servers

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"vervet/internal/models"
)

func TestCreateGroup(t *testing.T) {
	t.Run("successful create", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "parent", Name: "Parent Group", IsGroup: true},
			},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		id, err := sm.CreateGroup("parent", "New Group")

		assert.NoError(t, err)
		assert.NotEmpty(t, id)
		assert.Len(t, mockStore.servers, 2)
		assert.Equal(t, id, mockStore.servers[1].ID)
		assert.True(t, mockStore.servers[1].IsGroup)
		assert.Equal(t, "New Group", mockStore.servers[1].Name)
		assert.Equal(t, "parent", mockStore.servers[1].ParentID)
	})

	t.Run("duplicate sibling name rejected", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "g1", Name: "Test Group", IsGroup: true, ParentID: ""},
			},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		id, err := sm.CreateGroup("", "Test Group")

		assert.Empty(t, id)
		assert.ErrorIs(t, err, ErrDuplicateGroupName)
		assert.Len(t, mockStore.servers, 1)
	})

	t.Run("duplicate sibling name case-insensitive rejected", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "g1", Name: "Test Group", IsGroup: true, ParentID: ""},
			},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		_, err := sm.CreateGroup("", "  test group  ")

		assert.True(t, errors.Is(err, ErrDuplicateGroupName))
	})

	t.Run("same name under different parent allowed", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "p1", Name: "Parent", IsGroup: true, ParentID: ""},
				{ID: "g1", Name: "Test Group", IsGroup: true, ParentID: ""},
			},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		id, err := sm.CreateGroup("p1", "Test Group")

		assert.NoError(t, err)
		assert.NotEmpty(t, id)
	})
}

func TestUpdateGroup(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Group 1", IsGroup: true},
			},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		err := sm.UpdateGroup("1", "Updated Group", "")

		assert.NoError(t, err)
		assert.Equal(t, "Updated Group", mockStore.servers[0].Name)
	})

	t.Run("group not found", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		err := sm.UpdateGroup("1", "Updated Group", "")

		assert.Error(t, err)
	})

	t.Run("not a group", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Server 1", IsGroup: false},
			},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		err := sm.UpdateGroup("1", "Updated Group", "")

		assert.Error(t, err)
	})

	t.Run("rename to sibling name rejected", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Alpha", IsGroup: true, ParentID: ""},
				{ID: "2", Name: "Beta", IsGroup: true, ParentID: ""},
			},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		err := sm.UpdateGroup("2", "Alpha", "")

		assert.ErrorIs(t, err, ErrDuplicateGroupName)
		assert.Equal(t, "Beta", mockStore.servers[1].Name)
	})

	t.Run("rename to own name allowed (no-op)", func(t *testing.T) {
		mockStore := &mockServerStore{
			servers: []models.RegisteredServer{
				{ID: "1", Name: "Alpha", IsGroup: true, ParentID: ""},
			},
		}
		sm := newTestServerService(mockStore, &MockConnectionStringsStore{})

		err := sm.UpdateGroup("1", "Alpha", "")

		assert.NoError(t, err)
	})
}

func TestFindGroup(t *testing.T) {
	servers := []models.RegisteredServer{
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
