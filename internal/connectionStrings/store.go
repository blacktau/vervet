package connectionStrings

import (
	"fmt"
	"log/slog"
	"time"
	"vervet/internal/logging"

	"github.com/zalando/go-keyring"
)

const serviceName = "Vervet"
const keyringTimeout = 5 * time.Second

type Store interface {
	StoreRegisteredServerURI(registeredServerID, uri string) error
	GetRegisteredServerURI(registeredServerID string) (string, error)
	DeleteRegisteredServerURI(registeredServerID string) error
}

type store struct {
	log *slog.Logger
}

func NewStore(log *slog.Logger) *store {
	return &store{
		log: log.With(slog.String(logging.SourceKey, "ConnectionStringStore")),
	}
}

// StoreRegisteredServerURI securely saves a connectionURI associated with a user provided name
func (s *store) StoreRegisteredServerURI(registeredServerID, uri string) error {
	key := getKey(registeredServerID)
	err := withTimeout(keyringTimeout, func() error {
		return keyring.Set(serviceName, key, uri)
	})
	if err != nil {
		s.log.Error("Failed to store registeredServer URI securely", slog.Any("error", err))
		return fmt.Errorf("failed to store registeredServer URI securely: %w", err)
	}
	return nil
}

func (s *store) GetRegisteredServerURI(registeredServerID string) (string, error) {
	key := getKey(registeredServerID)
	var uri string
	err := withTimeout(keyringTimeout, func() error {
		var getErr error
		uri, getErr = keyring.Get(serviceName, key)
		return getErr
	})
	if err != nil {
		s.log.Error("Failed to retrieve connection URI", slog.Any("error", err))
		return "", fmt.Errorf("failed to retrieve connection URI: %w", err)
	}

	return uri, nil
}

func (s *store) DeleteRegisteredServerURI(registeredServerID string) error {
	key := getKey(registeredServerID)
	err := withTimeout(keyringTimeout, func() error {
		return keyring.Delete(serviceName, key)
	})
	if err != nil {
		s.log.Error("Failed to delete registeredServer URI", slog.Any("error", err))
		return fmt.Errorf("failed to delete registeredServer URI: %w", err)
	}
	return nil
}

// withTimeout runs fn in a goroutine and returns its error, or a timeout error
// if it doesn't complete within the given duration. This prevents keyring operations
// from hanging indefinitely when the OS secret service (D-Bus) is unavailable,
// which is common in environments like WSL2 without a running keyring daemon.
func withTimeout(timeout time.Duration, fn func() error) error {
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("keyring operation timed out after %v — the OS secret service may be unavailable (check that a keyring daemon is running)", timeout)
	}
}

func getKey(registeredServerID string) string {
	return fmt.Sprintf("conn_%s", registeredServerID)
}
