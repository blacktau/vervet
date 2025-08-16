// Package configuration manages the configuration for the application
package configuration

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const serviceName = "Vervet"

// StoreRegisteredServerURI securly saves a conectionURI associtated with a user provided name
func StoreRegisteredServerURI(registeredServerID int, uri string) error {
	key := getKey(int(registeredServerID))
	err := keyring.Set(serviceName, key, uri)
	if err != nil {
		return fmt.Errorf("failed to store registeredServer URI securly: %w", err)
	}
	return nil
}

func GetRegisteredServerURI(registeredServerID int) (string, error) {
	key := getKey(registeredServerID)
	uri, err := keyring.Get(serviceName, key)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve conneciton URI: %w", err)
	}

	return uri, nil
}

func DeleteRegisteredServerURI(registeredServerID int) error {
	key := getKey(registeredServerID)
	err := keyring.Delete(serviceName, key)
	if err != nil {
		return fmt.Errorf("failed to delete registeredServer URI: %w", err)
	}
	return nil
}

func getKey(registeredServerID int) string {
	return fmt.Sprintf("conn_%d", registeredServerID)
}
