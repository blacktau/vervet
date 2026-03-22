package workspaces

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"

	"vervet/internal/models"
)

var defaultExtensions = []string{".js", ".mongodb"}

type WorkspaceService struct {
	log   *slog.Logger
	store WorkspaceStore
}

func NewService(log *slog.Logger, store WorkspaceStore) *WorkspaceService {
	return &WorkspaceService{log: log, store: store}
}

func (s *WorkspaceService) GetWorkspaces() (models.WorkspaceData, error) {
	return s.store.Load()
}

func (s *WorkspaceService) CreateWorkspace(name string) (models.Workspace, error) {
	data, err := s.store.Load()
	if err != nil {
		return models.Workspace{}, err
	}

	ws := models.Workspace{
		ID:      uuid.New().String(),
		Name:    name,
		Folders: []string{},
	}

	data.Workspaces = append(data.Workspaces, ws)

	if err := s.store.Save(data); err != nil {
		return models.Workspace{}, err
	}

	return ws, nil
}

func (s *WorkspaceService) RenameWorkspace(id, name string) error {
	data, err := s.store.Load()
	if err != nil {
		return err
	}

	idx := slices.IndexFunc(data.Workspaces, func(w models.Workspace) bool {
		return w.ID == id
	})
	if idx == -1 {
		return fmt.Errorf("workspace not found: %s", id)
	}

	data.Workspaces[idx].Name = name
	return s.store.Save(data)
}

func (s *WorkspaceService) DeleteWorkspace(id string) error {
	data, err := s.store.Load()
	if err != nil {
		return err
	}

	idx := slices.IndexFunc(data.Workspaces, func(w models.Workspace) bool {
		return w.ID == id
	})
	if idx == -1 {
		return fmt.Errorf("workspace not found: %s", id)
	}

	data.Workspaces = slices.Delete(data.Workspaces, idx, idx+1)

	if data.ActiveWorkspaceID == id {
		data.ActiveWorkspaceID = ""
	}

	return s.store.Save(data)
}

func (s *WorkspaceService) SetActiveWorkspace(id string) error {
	data, err := s.store.Load()
	if err != nil {
		return err
	}

	found := slices.ContainsFunc(data.Workspaces, func(w models.Workspace) bool {
		return w.ID == id
	})
	if !found {
		return fmt.Errorf("workspace not found: %s", id)
	}

	data.ActiveWorkspaceID = id
	return s.store.Save(data)
}

func (s *WorkspaceService) AddFolder(workspaceID, path string) error {
	data, err := s.store.Load()
	if err != nil {
		return err
	}

	idx := slices.IndexFunc(data.Workspaces, func(w models.Workspace) bool {
		return w.ID == workspaceID
	})
	if idx == -1 {
		return fmt.Errorf("workspace not found: %s", workspaceID)
	}

	if slices.Contains(data.Workspaces[idx].Folders, path) {
		return fmt.Errorf("folder already exists: %s", path)
	}

	data.Workspaces[idx].Folders = append(data.Workspaces[idx].Folders, path)
	return s.store.Save(data)
}

func (s *WorkspaceService) RemoveFolder(workspaceID, path string) error {
	data, err := s.store.Load()
	if err != nil {
		return err
	}

	idx := slices.IndexFunc(data.Workspaces, func(w models.Workspace) bool {
		return w.ID == workspaceID
	})
	if idx == -1 {
		return fmt.Errorf("workspace not found: %s", workspaceID)
	}

	folderIdx := slices.Index(data.Workspaces[idx].Folders, path)
	if folderIdx == -1 {
		return fmt.Errorf("folder not found: %s", path)
	}

	data.Workspaces[idx].Folders = slices.Delete(data.Workspaces[idx].Folders, folderIdx, folderIdx+1)
	return s.store.Save(data)
}

func (s *WorkspaceService) ReadDirectory(dirPath string, extensions []string) ([]models.DirectoryEntry, error) {
	if len(extensions) == 0 {
		extensions = defaultExtensions
	}

	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var dirs []models.DirectoryEntry
	var files []models.DirectoryEntry

	for _, entry := range dirEntries {
		name := entry.Name()

		if entry.IsDir() {
			if strings.HasPrefix(name, ".") {
				continue
			}
			dirs = append(dirs, models.DirectoryEntry{
				Name:        name,
				Path:        filepath.Join(dirPath, name),
				IsDirectory: true,
			})
		} else {
			ext := strings.ToLower(filepath.Ext(name))
			if slices.Contains(extensions, ext) {
				files = append(files, models.DirectoryEntry{
					Name: name,
					Path: filepath.Join(dirPath, name),
				})
			}
		}
	}

	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Name < dirs[j].Name
	})
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return append(dirs, files...), nil
}

func (s *WorkspaceService) CreateFile(dirPath, name string) (string, error) {
	fullPath := filepath.Join(dirPath, name)

	if _, err := os.Stat(fullPath); err == nil {
		return "", fmt.Errorf("file already exists: %s", fullPath)
	}

	if err := os.WriteFile(fullPath, []byte{}, 0644); err != nil {
		return "", err
	}

	return fullPath, nil
}

func (s *WorkspaceService) RenameFile(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

func (s *WorkspaceService) DeleteFile(path string) error {
	return os.RemoveAll(path)
}
