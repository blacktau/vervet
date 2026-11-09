package files

import (
	"fmt"
	"log/slog"
	"os"

	"vervet/internal/api"
	"vervet/internal/desktop"
)

// Dialogs shows the native file dialogs. A cancelled dialog returns an empty
// path and no error.
type Dialogs interface {
	SelectFile(title string, filters []desktop.FileFilter) (string, error)
	SaveFile(title, defaultFilename string, filters []desktop.FileFilter) (string, error)
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

func (s *Service) SelectFile(title string, filters []api.FileFilter) (string, error) {
	filepath, err := s.dialogs.SelectFile(title, toDialogFilters(filters))
	if err != nil {
		return "", fmt.Errorf("failed to select file: %w", err)
	}

	return filepath, nil
}

func (s *Service) SaveFile(title *string, name *string, filters []api.FileFilter) (string, error) {
	dialogTitle := ""
	if title != nil {
		dialogTitle = *title
	}

	defaultFilename := ""
	if name != nil {
		defaultFilename = *name
	}

	filepath, err := s.dialogs.SaveFile(dialogTitle, defaultFilename, toDialogFilters(filters))
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

func toDialogFilters(filters []api.FileFilter) []desktop.FileFilter {
	result := make([]desktop.FileFilter, len(filters))
	for i, f := range filters {
		result[i] = desktop.FileFilter{
			DisplayName: f.DisplayName,
			Pattern:     f.Pattern,
		}
	}
	return result
}
