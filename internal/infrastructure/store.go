package infrastructure

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"vervet/internal/logging"
)

const configFolder = "vervet"

type Store interface {
	Read() ([]byte, error)
	Save(data []byte) error
}

type cfgStore struct {
	ConfigPath string
	log        *slog.Logger
	mu         sync.Mutex
}

func NewStore(filename string, log *slog.Logger) (Store, error) {
	configDir, err := getConfigDirectory()
	if err != nil {
		return nil, err
	}
	return &cfgStore{
		ConfigPath: filepath.Join(configDir, filename),
		log:        log.With(slog.String(logging.SourceKey, "ConfigurationStore")),
	}, nil
}

func (s *cfgStore) Read() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.ConfigPath); os.IsNotExist(err) {
		err := os.WriteFile(s.ConfigPath, []byte{}, 0600)
		if err != nil {
			return []byte{}, err
		}
	} else if err != nil {
		return []byte{}, err
	}

	d, err := os.ReadFile(s.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration from %s: %w", s.ConfigPath, err)
	}
	return d, nil
}

// Save writes data atomically: it writes to a sibling temp file, fsyncs,
// then renames over the target. A mutex serialises concurrent callers so
// two goroutines cannot interleave truncate+write on the same path.
func (s *cfgStore) Save(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Dir(s.ConfigPath)
	tmp, err := os.CreateTemp(dir, filepath.Base(s.ConfigPath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("error saving configuration: %w", err)
	}
	tmpPath := tmp.Name()

	cleanup := func() {
		_ = os.Remove(tmpPath)
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("error saving configuration: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("error saving configuration: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("error saving configuration: %w", err)
	}
	if err := os.Chmod(tmpPath, 0600); err != nil {
		cleanup()
		return fmt.Errorf("error saving configuration: %w", err)
	}
	if err := os.Rename(tmpPath, s.ConfigPath); err != nil {
		cleanup()
		return fmt.Errorf("error saving configuration: %w", err)
	}
	return nil
}

func getConfigDirectory() (string, error) {

	configHome, err := os.UserConfigDir()
	if err != nil {
		slog.Error("could not determine config home directory", slog.Any("error", err))
		return "", fmt.Errorf("could not determine config home directory: %w", err)
	}

	configDir := filepath.Join(configHome, configFolder)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		err := os.MkdirAll(configDir, 0700)
		if err != nil {
			slog.Error("could not create config home directory", slog.String("configDir", configDir), slog.Any("error", err))
			return "", fmt.Errorf("could not create config home directory '%s': %w", configDir, err)
		}
	}

	return configDir, nil
}
