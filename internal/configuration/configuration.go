// Package configuration manages the configuration for the application
package configuration

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const serviceName = "Vervet"

// StoreConnectionURI securly saves a conectionURI associtated with a user provided name
func StoreConnectionURI(connectionID int64, uri string) error {
	key := getKey(int(connectionID))
	err := keyring.Set(serviceName, key, uri)
	if err != nil {
		return fmt.Errorf("failed to store connection URI securly: %w", err)
	}
	return nil
}

func GetConnectionURI(connectionID int) (string, error) {
	key := getKey(connectionID)
	uri, err := keyring.Get(serviceName, key)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve conneciton URI: %w", err)
	}

	return uri, nil
}

func DeleteConnectionURI(connectionID int) error {
	key := getKey(connectionID)
	err := keyring.Delete(serviceName, key)
	if err != nil {
		return fmt.Errorf("failed to delete connection URI: %w", err)
	}
	return nil
}

func getKey(connectionID int) string {
	return fmt.Sprintf("conn_%d", connectionID)
}
