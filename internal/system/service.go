package system

import (
	"context"
	"fmt"
	"log/slog"
	"vervet/internal/logging"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Service struct {
	ctx context.Context
	log *slog.Logger
}

func NewSystemService(log *slog.Logger) *Service {
	return &Service{
		log: log.With(slog.String(logging.SourceKey, "Service")),
	}
}

func (ss *Service) Init(ctx context.Context) error {
	ss.log.Debug("Initializing System Service")
	ss.ctx = ctx
	return nil
}

func (ss *Service) SelectFile(title string, extensions *[]string) (string, error) {
	var filters []runtime.FileFilter
	if extensions == nil {
		extensions = &[]string{}
	}
	for _, extension := range *extensions {
		filters = append(filters, runtime.FileFilter{
			Pattern: "*." + extension,
		})
	}

	filepath, err := runtime.OpenFileDialog(ss.ctx, runtime.OpenDialogOptions{
		Title:           title,
		Filters:         filters,
		ShowHiddenFiles: true,
	})

	if err != nil {
		ss.log.Error("Failed to select file", slog.Any("error", err))
		return "", fmt.Errorf("failed to select file: %w", err)
	}

	return filepath, nil
}

func (ss *Service) SaveFile(title *string, name *string, extensions *[]string) (string, error) {
	var filters []runtime.FileFilter

	if extensions == nil {
		extensions = &[]string{}
	}

	for _, extension := range *extensions {
		filters = append(filters, runtime.FileFilter{
			Pattern: "*." + extension,
		})
	}

	filepath, err := runtime.SaveFileDialog(ss.ctx, runtime.SaveDialogOptions{
		Title:           *title,
		DefaultFilename: *name,
		Filters:         filters,
	})

	if err != nil {
		ss.log.Error("Failed to select save file", slog.Any("error", err))
		return "", fmt.Errorf("failed to select save file: %w", err)
	}

	return filepath, nil
}
