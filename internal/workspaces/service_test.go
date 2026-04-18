package workspaces

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"vervet/internal/models"
)

type mockStore struct {
	data models.WorkspaceData
}

func (m *mockStore) Load() (models.WorkspaceData, error)  { return m.data, nil }
func (m *mockStore) Save(data models.WorkspaceData) error { m.data = data; return nil }

func newTestService() (*WorkspaceService, *mockStore) {
	store := &mockStore{}
	svc := NewService(slog.Default(), store)
	return svc, store
}

func TestCreateWorkspace(t *testing.T) {
	svc, store := newTestService()

	ws, err := svc.CreateWorkspace("My Workspace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ws.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if ws.Name != "My Workspace" {
		t.Fatalf("expected name 'My Workspace', got %q", ws.Name)
	}
	if len(store.data.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace in store, got %d", len(store.data.Workspaces))
	}
}

func TestRenameWorkspace(t *testing.T) {
	svc, store := newTestService()

	ws, _ := svc.CreateWorkspace("Old Name")

	err := svc.RenameWorkspace(ws.ID, "New Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store.data.Workspaces[0].Name != "New Name" {
		t.Fatalf("expected name 'New Name', got %q", store.data.Workspaces[0].Name)
	}
}

func TestDeleteWorkspace(t *testing.T) {
	svc, store := newTestService()

	ws, _ := svc.CreateWorkspace("To Delete")

	err := svc.DeleteWorkspace(ws.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.data.Workspaces) != 0 {
		t.Fatalf("expected 0 workspaces, got %d", len(store.data.Workspaces))
	}
}

func TestDeleteActiveWorkspaceClearsActiveID(t *testing.T) {
	svc, store := newTestService()

	ws, _ := svc.CreateWorkspace("Active WS")
	_ = svc.SetActiveWorkspace(ws.ID)

	if store.data.ActiveWorkspaceID != ws.ID {
		t.Fatalf("expected active ID %q, got %q", ws.ID, store.data.ActiveWorkspaceID)
	}

	err := svc.DeleteWorkspace(ws.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store.data.ActiveWorkspaceID != "" {
		t.Fatalf("expected active ID to be cleared, got %q", store.data.ActiveWorkspaceID)
	}
}

func TestSetActiveWorkspace(t *testing.T) {
	svc, store := newTestService()

	ws, _ := svc.CreateWorkspace("WS")

	err := svc.SetActiveWorkspace(ws.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store.data.ActiveWorkspaceID != ws.ID {
		t.Fatalf("expected active ID %q, got %q", ws.ID, store.data.ActiveWorkspaceID)
	}
}

func TestAddFolder(t *testing.T) {
	svc, store := newTestService()

	ws, _ := svc.CreateWorkspace("WS")

	err := svc.AddFolder(ws.ID, "/some/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.data.Workspaces[0].Folders) != 1 {
		t.Fatalf("expected 1 folder, got %d", len(store.data.Workspaces[0].Folders))
	}
	if store.data.Workspaces[0].Folders[0] != "/some/path" {
		t.Fatalf("expected folder '/some/path', got %q", store.data.Workspaces[0].Folders[0])
	}
}

func TestAddDuplicateFolder(t *testing.T) {
	svc, _ := newTestService()

	ws, _ := svc.CreateWorkspace("WS")
	_ = svc.AddFolder(ws.ID, "/some/path")

	err := svc.AddFolder(ws.ID, "/some/path")
	if err == nil {
		t.Fatal("expected error for duplicate folder, got nil")
	}
}

func TestRemoveFolder(t *testing.T) {
	svc, store := newTestService()

	ws, _ := svc.CreateWorkspace("WS")
	_ = svc.AddFolder(ws.ID, "/some/path")

	err := svc.RemoveFolder(ws.ID, "/some/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.data.Workspaces[0].Folders) != 0 {
		t.Fatalf("expected 0 folders, got %d", len(store.data.Workspaces[0].Folders))
	}
}

func TestReadDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create files with various extensions
	os.WriteFile(filepath.Join(dir, "script.js"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "query.mongodb"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "data.json"), []byte(""), 0644)
	os.Mkdir(filepath.Join(dir, "subfolder"), 0755)
	os.Mkdir(filepath.Join(dir, ".hidden"), 0755)

	svc, _ := newTestService()

	entries, err := svc.ReadDirectory(dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include: subfolder (dir), query.mongodb, script.js
	// Should exclude: .hidden (hidden dir), readme.txt, data.json
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d: %+v", len(entries), entries)
	}

	// Folders first
	if !entries[0].IsDirectory || entries[0].Name != "subfolder" {
		t.Fatalf("expected first entry to be 'subfolder' directory, got %+v", entries[0])
	}

	// Then files alphabetically
	if entries[1].Name != "query.mongodb" {
		t.Fatalf("expected second entry 'query.mongodb', got %q", entries[1].Name)
	}
	if entries[2].Name != "script.js" {
		t.Fatalf("expected third entry 'script.js', got %q", entries[2].Name)
	}
}

func TestReadDirectoryCustomExtensions(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "script.js"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "data.json"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte(""), 0644)

	svc, _ := newTestService()

	entries, err := svc.ReadDirectory(dir, []string{".json", ".txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(entries), entries)
	}
	if entries[0].Name != "data.json" {
		t.Fatalf("expected 'data.json', got %q", entries[0].Name)
	}
	if entries[1].Name != "readme.txt" {
		t.Fatalf("expected 'readme.txt', got %q", entries[1].Name)
	}
}

func TestCreateFile(t *testing.T) {
	dir := t.TempDir()

	svc, _ := newTestService()

	path, err := svc.CreateFile(dir, "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(dir, "test.js")
	if path != expected {
		t.Fatalf("expected path %q, got %q", expected, path)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("expected file to exist on disk")
	}
}

func TestCreateFileDuplicate(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "exists.js"), []byte(""), 0644)

	svc, _ := newTestService()

	_, err := svc.CreateFile(dir, "exists.js")
	if err == nil {
		t.Fatal("expected error for duplicate file, got nil")
	}
}

func TestRenameFile(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.js")
	newPath := filepath.Join(dir, "new.js")
	os.WriteFile(oldPath, []byte("content"), 0644)

	svc, _ := newTestService()

	err := svc.RenameFile(oldPath, newPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatal("expected old file to not exist")
	}
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Fatal("expected new file to exist")
	}
}

func TestDeleteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todelete.js")
	os.WriteFile(path, []byte("content"), 0644)

	svc, _ := newTestService()

	err := svc.DeleteFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected file to be deleted")
	}
}
