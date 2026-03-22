package workspaces

import (
	"log/slog"

	"gopkg.in/yaml.v3"

	"vervet/internal/infrastructure"
	"vervet/internal/models"
)

type WorkspaceStore interface {
	Load() (models.WorkspaceData, error)
	Save(data models.WorkspaceData) error
}

type store struct {
	cfgStore infrastructure.Store
	log      *slog.Logger
}

func NewStore(log *slog.Logger) (*store, error) {
	cfgStore, err := infrastructure.NewStore("workspaces.yaml", log)
	if err != nil {
		return nil, err
	}
	return &store{cfgStore: cfgStore, log: log}, nil
}

func (s *store) Load() (models.WorkspaceData, error) {
	b, err := s.cfgStore.Read()
	if err != nil {
		return models.WorkspaceData{}, err
	}

	var data models.WorkspaceData
	if len(b) > 0 {
		if err := yaml.Unmarshal(b, &data); err != nil {
			return models.WorkspaceData{}, err
		}
	}
	return data, nil
}

func (s *store) Save(data models.WorkspaceData) error {
	b, err := yaml.Marshal(&data)
	if err != nil {
		return err
	}
	return s.cfgStore.Save(b)
}
