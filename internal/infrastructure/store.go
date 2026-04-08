package infrastructure

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
	if _, err := os.Stat(s.ConfigPath); os.IsNotExist(err) {
		if s.log != nil {
			s.log.Info("Configuration file does not exist yet. Creating it.")
		}
		err := os.WriteFile(s.ConfigPath, []byte{}, 0600)
		if err != nil {
			if s.log != nil {
				s.log.Error("Error creating configuration file", slog.Any("error", err))
			}
			return []byte{}, err
		}
	} else if err != nil {
		return []byte{}, err
	}

	d, err := os.ReadFile(s.ConfigPath)
	if err != nil {
		if s.log != nil {
			s.log.Error("Error reading configuration", slog.Any("error", err))
		}
		return nil, fmt.Errorf("failed to read configuration from %s: %w", s.ConfigPath, err)
	}
	return d, nil
}

func (s *cfgStore) Save(data []byte) error {
	if err := os.WriteFile(s.ConfigPath, data, 0600); err != nil {
		if s.log != nil {
			s.log.Error("Error saving configuration", slog.Any("error", err))
		}
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
