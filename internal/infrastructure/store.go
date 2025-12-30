package infrastructure

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/labstack/gommon/log"
	"github.com/wailsapp/wails/v2/pkg/logger"
)

const configFolder = "vervet"

type Store interface {
	Read() ([]byte, error)
	Save(data []byte) error
}

type cfgStore struct {
	ConfigPath string
	log        logger.Logger
}

func NewStore(filename string, log logger.Logger) (Store, error) {
	configDir, err := getConfigDirectory()
	if err != nil {
		return nil, err
	}
	return &cfgStore{
		ConfigPath: path.Join(configDir, filename),
		log:        log,
	}, nil
}

func (s *cfgStore) Read() ([]byte, error) {
	if _, err := os.Stat(s.ConfigPath); os.IsNotExist(err) {
		log.Info("Configuration file does not exist yet. Creating it.")
		err := os.WriteFile(s.ConfigPath, []byte{}, 0700)
		if err != nil {
			log.Errorf("Error creating configuration file: %v", err)
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

func (s *cfgStore) Save(data []byte) error {
	if err := os.WriteFile(s.ConfigPath, data, 0700); err != nil {
		return fmt.Errorf("error saving configuration: %w", err)
	}
	return nil
}

func getConfigDirectory() (string, error) {

	configHome, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not determine config home directory: %w", err)
	}

	configDir := filepath.Join(configHome, configFolder)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		err := os.MkdirAll(configDir, 0700)
		if err != nil {
			return "", fmt.Errorf("could not create config home directory '%s': %w", configDir, err)
		}
	}

	return configDir, nil
}
