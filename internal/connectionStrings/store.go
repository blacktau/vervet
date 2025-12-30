package connectionStrings

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const serviceName = "Vervet"

type Store interface {
	StoreRegisteredServerURI(registeredServerID, uri string) error
	GetRegisteredServerURI(registeredServerID string) (string, error)
	DeleteRegisteredServerURI(registeredServerID string) error
}

type store struct {
}

func NewStore() Store {
	return &store{}
}

// StoreRegisteredServerURI securely saves a connectionURI associated with a user provided name
func (s *store) StoreRegisteredServerURI(registeredServerID, uri string) error {
	key := getKey(registeredServerID)
	err := keyring.Set(serviceName, key, uri)
	if err != nil {
		return fmt.Errorf("failed to store registeredServer URI securly: %w", err)
	}
	return nil
}

func (s *store) GetRegisteredServerURI(registeredServerID string) (string, error) {
	key := getKey(registeredServerID)
	uri, err := keyring.Get(serviceName, key)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve conneciton URI: %w", err)
	}

	return uri, nil
}

func (s *store) DeleteRegisteredServerURI(registeredServerID string) error {
	key := getKey(registeredServerID)
	err := keyring.Delete(serviceName, key)
	if err != nil {
		return fmt.Errorf("failed to delete registeredServer URI: %w", err)
	}
	return nil
}

func getKey(registeredServerID string) string {
	return fmt.Sprintf("conn_%s", registeredServerID)
}
