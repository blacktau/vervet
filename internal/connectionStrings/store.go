package connectionStrings

import (
	"fmt"
	"log/slog"
	"vervet/internal/logging"

	"github.com/zalando/go-keyring"
)

const serviceName = "Vervet"

type Store interface {
	StoreRegisteredServerURI(registeredServerID, uri string) error
	GetRegisteredServerURI(registeredServerID string) (string, error)
	DeleteRegisteredServerURI(registeredServerID string) error
}

type store struct {
	log *slog.Logger
}

func NewStore(log *slog.Logger) Store {

	return &store{
		log: log.With(slog.String(logging.SourceKey, "ConnectionStringStore")),
	}
}

// StoreRegisteredServerURI securely saves a connectionURI associated with a user provided name
func (s *store) StoreRegisteredServerURI(registeredServerID, uri string) error {
	key := getKey(registeredServerID)
	err := keyring.Set(serviceName, key, uri)
	if err != nil {
		s.log.Error("Failed to store registeredServer URI securely", slog.Any("error", err))
		return fmt.Errorf("failed to store registeredServer URI securely: %w", err)
	}
	return nil
}

func (s *store) GetRegisteredServerURI(registeredServerID string) (string, error) {
	key := getKey(registeredServerID)
	uri, err := keyring.Get(serviceName, key)
	if err != nil {
		s.log.Error("Failed to retrieve connection URI", slog.Any("error", err))
		return "", fmt.Errorf("failed to retrieve connection URI: %w", err)
	}

	return uri, nil
}

func (s *store) DeleteRegisteredServerURI(registeredServerID string) error {
	key := getKey(registeredServerID)
	err := keyring.Delete(serviceName, key)
	if err != nil {
		s.log.Error("Failed to delete registeredServer URI", slog.Any("error", err))
		return fmt.Errorf("failed to delete registeredServer URI: %w", err)
	}
	return nil
}

func getKey(registeredServerID string) string {
	return fmt.Sprintf("conn_%s", registeredServerID)
}
