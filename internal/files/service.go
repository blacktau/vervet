package files

import (
	"fmt"
	"log/slog"
	"os"

	"vervet/internal/models"
)

// Dialogs shows the native file dialogs. A cancelled dialog returns an empty
// path and no error.
type Dialogs interface {
	SelectFile(title string, filters []models.FileFilter) (string, error)
	SaveFile(title, defaultFilename string, filters []models.FileFilter) (string, error)
}

type Service struct {
	log     *slog.Logger
	dialogs Dialogs
}

func NewService(log *slog.Logger, dialogs Dialogs) *Service {
	return &Service{
		log:     log.With(slog.String("source", "FilesService")),
		dialogs: dialogs,
	}
}

func (s *Service) SelectFile(title string, filters []models.FileFilter) (string, error) {
	filepath, err := s.dialogs.SelectFile(title, filters)
	if err != nil {
		return "", fmt.Errorf("failed to select file: %w", err)
	}

	return filepath, nil
}

func (s *Service) SaveFile(title *string, name *string, filters []models.FileFilter) (string, error) {
	dialogTitle := ""
	if title != nil {
		dialogTitle = *title
	}

	defaultFilename := ""
	if name != nil {
		defaultFilename = *name
	}

	filepath, err := s.dialogs.SaveFile(dialogTitle, defaultFilename, filters)
	if err != nil {
		return "", fmt.Errorf("failed to select save file: %w", err)
	}

	return filepath, nil
}

func (s *Service) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(data), nil
}

func (s *Service) WriteFile(path string, content string) error {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
