package servers

import (
	"fmt"
	"log/slog"
	"vervet/internal/infrastructure"
	"vervet/internal/logging"
	"vervet/internal/models"

	"gopkg.in/yaml.v3"
)

type store struct {
	cfgStore infrastructure.Store
	log      *slog.Logger
}

type ServerStore interface {
	LoadServers() ([]models.RegisteredServer, error)
	SaveServers(servers []models.RegisteredServer) error
}

func NewServerStore(log *slog.Logger) (*store, error) {
	logger := log.With(slog.String(logging.SourceKey, "ServerStore"))
	cfgStore, err := infrastructure.NewStore("connections.yaml", logger)
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %v", err)
	}

	return &store{
		cfgStore: cfgStore,
		log:      logger,
	}, nil
}

func (s *store) LoadServers() ([]models.RegisteredServer, error) {
	b, err := s.cfgStore.Read()
	if err != nil {
		s.log.Error("error loading registered servers", slog.Any("error", err))
		return nil, fmt.Errorf("error loading registered servers: %v", err)
	}

	registeredServers := make([]models.RegisteredServer, 0)

	if len(b) <= 0 {
		return registeredServers, nil
	}

	if err = yaml.Unmarshal(b, &registeredServers); err != nil {
		s.log.Error("error parsing registered servers, returning empty list", slog.Any("error", err))
		return make([]models.RegisteredServer, 0), fmt.Errorf("server configuration file is corrupted and could not be read — your server list may be empty until the file is repaired: %w", err)
	}

	return registeredServers, nil
}

func (s *store) SaveServers(registeredServers []models.RegisteredServer) error {
	s.log.Debug("Saving Registered Servers")
	b, err := yaml.Marshal(&registeredServers)
	if err != nil {
		s.log.Error("error marshalling registered servers", slog.Any("error", err))
		return fmt.Errorf("error marshalling registered servers: %v", err)
	}

	if err = s.cfgStore.Save(b); err != nil {
		s.log.Error("error saving registered servers", slog.Any("error", err))
		return fmt.Errorf("error saving registered servers: %v", err)
	}

	return nil
}
