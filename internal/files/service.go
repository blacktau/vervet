package files

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Service struct {
	ctx context.Context
	log *slog.Logger
}

func NewService(log *slog.Logger) *Service {
	return &Service{
		log: log.With(slog.String("source", "FilesService")),
	}
}

func (s *Service) Init(ctx context.Context) error {
	s.ctx = ctx
	return nil
}

func (s *Service) SelectFile(title string, extensions *[]string) (string, error) {
	filters := make([]runtime.FileFilter, 0)
	if extensions != nil {
		for _, ext := range *extensions {
			filters = append(filters, runtime.FileFilter{
				Pattern: ext,
			})
		}
	}

	filepath, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title:           title,
		Filters:         filters,
		ShowHiddenFiles: true,
	})
	if err != nil {
		s.log.Error("Failed to select file", slog.Any("error", err))
		return "", fmt.Errorf("failed to select file: %w", err)
	}

	return filepath, nil
}

func (s *Service) SaveFile(title *string, name *string, extensions *[]string) (string, error) {
	filters := make([]runtime.FileFilter, 0)
	if extensions != nil {
		for _, ext := range *extensions {
			filters = append(filters, runtime.FileFilter{
				Pattern: ext,
			})
		}
	}

	dialogTitle := ""
	if title != nil {
		dialogTitle = *title
	}

	defaultFilename := ""
	if name != nil {
		defaultFilename = *name
	}

	filepath, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{
		Title:           dialogTitle,
		DefaultFilename: defaultFilename,
		Filters:         filters,
	})
	if err != nil {
		s.log.Error("Failed to select save file", slog.Any("error", err))
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
