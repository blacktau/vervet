package servers

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockConfigurationStore struct {
	data []byte
	err  error
}

func (m *MockConfigurationStore) Read() ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

func (m *MockConfigurationStore) Save(data []byte) error {
	if m.err != nil {
		return m.err
	}
	m.data = data
	return nil
}

func (m *MockConfigurationStore) Path() string {
	return "test"
}

func TestLoadServers(t *testing.T) {
	t.Run("successful load", func(t *testing.T) {
		mockCfgStore := &MockConfigurationStore{
			data: []byte(`- id: "1"
  name: Server 1
`),
		}
		store := &Store{cfgStore: mockCfgStore}
		servers, err := store.LoadServers()
		assert.NoError(t, err)
		assert.Len(t, servers, 1)
	})

	t.Run("empty file", func(t *testing.T) {
		mockCfgStore := &MockConfigurationStore{
			data: []byte{},
		}
		store := &Store{cfgStore: mockCfgStore}
		servers, err := store.LoadServers()
		assert.NoError(t, err)
		assert.Len(t, servers, 0)
	})

	t.Run("read error", func(t *testing.T) {
		mockCfgStore := &MockConfigurationStore{
			err: errors.New("read error"),
		}
		store := &Store{cfgStore: mockCfgStore}
		servers, err := store.LoadServers()
		assert.Error(t, err)
		assert.Nil(t, servers)
	})

	t.Run("unmarshal error", func(t *testing.T) {
		mockCfgStore := &MockConfigurationStore{
			data: []byte(`invalid yaml`),
		}
		store := &Store{cfgStore: mockCfgStore}
		servers, err := store.LoadServers()
		assert.Error(t, err)
		assert.Nil(t, servers)
	})
}

func TestSaveServers(t *testing.T) {
	t.Run("successful save", func(t *testing.T) {
		mockCfgStore := &MockConfigurationStore{}
		store := &Store{cfgStore: mockCfgStore}
		servers := []RegisteredServer{
			{ID: "1", Name: "Server 1"},
		}
		err := store.SaveServers(servers)
		assert.NoError(t, err)
		assert.Contains(t, string(mockCfgStore.data), `id: "1"`)
	})

	t.Run("save error", func(t *testing.T) {
		mockCfgStore := &MockConfigurationStore{
			err: errors.New("save error"),
		}
		store := &Store{cfgStore: mockCfgStore}
		var servers []RegisteredServer
		err := store.SaveServers(servers)
		assert.Error(t, err)
	})
}
