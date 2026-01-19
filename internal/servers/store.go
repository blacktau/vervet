package servers

import (
	"fmt"
	"log/slog"
	"sync"
	"vervet/internal/infrastructure"
	"vervet/internal/logging"
	"vervet/internal/models"

	"gopkg.in/yaml.v3"
)

type Store struct {
	cfgStore infrastructure.Store
	mutex    sync.Mutex
	log      *slog.Logger
}

type ServerStore interface {
	LoadServers() ([]models.RegisteredServer, error)
	SaveServers(servers []models.RegisteredServer) error
}

func NewServerStore(log *slog.Logger) (ServerStore, error) {
	logger := log.With(slog.String(logging.SourceKey, "ServerStore"))
	cfgStore, err := infrastructure.NewStore("connections.yaml", logger)
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %v", err)
	}

	return &Store{
		cfgStore: cfgStore,
		log:      logger,
	}, nil
}

func (s *Store) LoadServers() ([]models.RegisteredServer, error) {
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
		s.log.Error("error parsing registered servers", slog.Any("error", err))
		return nil, fmt.Errorf("error parsing registered servers: %v", err)
	}

	return registeredServers, nil
}

func (s *Store) SaveServers(registeredServers []models.RegisteredServer) error {
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