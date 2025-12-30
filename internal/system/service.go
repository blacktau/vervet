package system

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Service interface {
	Init(ctx context.Context) error
	SelectFile(title string, extensions *[]string) (string, error)
	SaveFile(title *string, name *string, extensions *[]string) (string, error)
}

type systemService struct {
	ctx context.Context
}

func NewSystemService() Service {
	return &systemService{}
}

func (ss *systemService) Init(ctx context.Context) error {
	ss.ctx = ctx
	return nil
}

func (ss *systemService) SelectFile(title string, extensions *[]string) (string, error) {
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
		return "", fmt.Errorf("failed to select file: %w", err)
	}

	return filepath, nil
}

func (ss *systemService) SaveFile(title *string, name *string, extensions *[]string) (string, error) {
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
		return "", fmt.Errorf("failed to select save file: %w", err)
	}

	return filepath, nil
}
