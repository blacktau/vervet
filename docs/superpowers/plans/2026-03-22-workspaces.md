# Workspaces Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a workspace file manager that lets users organise local query files into named workspaces and open them as query tabs against connected MongoDB servers.

**Architecture:** New Go package `internal/workspaces/` with a `WorkspaceService` and YAML store, exposed via `WorkspacesProxy`. Frontend gets a new Pinia store, workspace pane components, and a server/database picker dialog. Integrates into the existing ribbon, tab system, and settings.

**Tech Stack:** Go (YAML persistence via `infrastructure.Store`), Vue 3 + Naive UI (`n-tree`, `n-select`, `n-dynamic-tags`), Pinia, Wails runtime dialogs, TypeScript.

---

## File Structure

### Backend (Go)

| Action | File | Responsibility |
|--------|------|---------------|
| Create | `internal/models/workspaces.go` | `Workspace`, `WorkspaceData`, `DirectoryEntry` types |
| Modify | `internal/models/settings.go` | Add `WorkspacesSettings` to `Settings` |
| Create | `internal/workspaces/store.go` | `WorkspaceStore` interface + YAML-backed implementation |
| Create | `internal/workspaces/service.go` | `WorkspaceService` — CRUD, directory reading, file ops |
| Create | `internal/workspaces/service_test.go` | Unit tests for service |
| Create | `internal/api/workspaces.go` | `WorkspacesProxy` — Wails-bound API layer |
| Modify | `internal/app/app.go` | Wire `WorkspaceService` + `WorkspacesProxy` |

### Frontend

| Action | File | Responsibility |
|--------|------|---------------|
| Create | `frontend/src/features/workspaces/workspaceStore.ts` | Pinia store for workspace state |
| Create | `frontend/src/features/workspaces/WorkspacePane.vue` | Main pane: toolbar + tree + empty states |
| Create | `frontend/src/features/workspaces/WorkspaceTree.vue` | `n-tree` with lazy loading and context menu |
| Create | `frontend/src/features/workspaces/WorkspaceToolbar.vue` | Workspace selector, +, gear, add folder, refresh |
| Create | `frontend/src/features/workspaces/ServerPickerDialog.vue` | Server + database picker modal |
| Create | `frontend/src/features/workspaces/WorkspaceEmptyState.vue` | Empty state prompts |
| Create | `frontend/src/features/settings/WorkspacesSettings.vue` | File extension config in settings dialog |
| Modify | `frontend/src/features/sidebar/LeftRibbon.vue` | Add Workspaces nav icon |
| Modify | `frontend/src/app/AppContent.vue` | Render `WorkspacePane` for workspaces nav |
| Modify | `frontend/src/features/tabs/tabs.ts` | Add `Workspaces` to `NavType` enum |
| Modify | `frontend/src/features/queries/queryStore.ts` | Add `loadFileByPath()` action |
| Modify | `frontend/src/features/settings/SettingsDialog.vue` | Add Workspaces tab |
| Modify | `frontend/src/stores/dialog.ts` | Add `ServerPicker` to `DialogType` |
| Modify | `frontend/src/i18n/en-GB/index.ts` | Add workspace translation keys |

---

### Task 1: Go Models

**Files:**
- Create: `internal/models/workspaces.go`
- Modify: `internal/models/settings.go`

- [ ] **Step 1: Create workspace model types**

Create `internal/models/workspaces.go`:

```go
package models

type Workspace struct {
	ID      string   `json:"id" yaml:"id"`
	Name    string   `json:"name" yaml:"name"`
	Folders []string `json:"folders" yaml:"folders"`
}

type WorkspaceData struct {
	ActiveWorkspaceID string      `json:"activeWorkspaceId" yaml:"activeWorkspaceId"`
	Workspaces        []Workspace `json:"workspaces" yaml:"workspaces"`
}

type DirectoryEntry struct {
	Name        string           `json:"name"`
	Path        string           `json:"path"`
	IsDirectory bool             `json:"isDirectory"`
	Children    []DirectoryEntry `json:"children,omitempty"`
}
```

- [ ] **Step 2: Add WorkspacesSettings to Settings**

In `internal/models/settings.go`, add the new struct and field:

```go
type WorkspacesSettings struct {
	FileExtensions []string `json:"fileExtensions" yaml:"fileExtensions"`
}
```

Add to the `Settings` struct:

```go
Workspaces WorkspacesSettings `json:"workspaces" yaml:"workspaces"`
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./internal/models/...`
Expected: clean build, no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/models/workspaces.go internal/models/settings.go
git commit -m "feat(workspaces): add Go model types for workspaces and settings"
```

---

### Task 2: Workspace Store (Go)

**Files:**
- Create: `internal/workspaces/store.go`

- [ ] **Step 1: Create the workspace store**

Create `internal/workspaces/store.go` following the pattern from `internal/servers/store.go`:

```go
package workspaces

import (
	"log/slog"

	"github.com/goccy/go-yaml"

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
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/workspaces/...`
Expected: clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/workspaces/store.go
git commit -m "feat(workspaces): add YAML-backed workspace store"
```

---

### Task 3: Workspace Service (Go)

**Files:**
- Create: `internal/workspaces/service.go`
- Create: `internal/workspaces/service_test.go`

- [ ] **Step 1: Write tests for workspace CRUD**

Create `internal/workspaces/service_test.go` with tests for create, rename, delete, set active, add/remove folder:

```go
package workspaces

import (
	"testing"

	"vervet/internal/models"
)

// mockStore implements WorkspaceStore in-memory for testing
type mockStore struct {
	data models.WorkspaceData
}

func (m *mockStore) Load() (models.WorkspaceData, error) {
	return m.data, nil
}

func (m *mockStore) Save(data models.WorkspaceData) error {
	m.data = data
	return nil
}

func newTestService() *WorkspaceService {
	return NewService(nil, &mockStore{data: models.WorkspaceData{}})
}

func TestCreateWorkspace(t *testing.T) {
	svc := newTestService()
	ws, err := svc.CreateWorkspace("Test Workspace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.Name != "Test Workspace" {
		t.Errorf("expected name 'Test Workspace', got '%s'", ws.Name)
	}
	if ws.ID == "" {
		t.Error("expected non-empty ID")
	}

	data, _ := svc.GetWorkspaces()
	if len(data.Workspaces) != 1 {
		t.Errorf("expected 1 workspace, got %d", len(data.Workspaces))
	}
}

func TestRenameWorkspace(t *testing.T) {
	svc := newTestService()
	ws, _ := svc.CreateWorkspace("Old Name")
	err := svc.RenameWorkspace(ws.ID, "New Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := svc.GetWorkspaces()
	if data.Workspaces[0].Name != "New Name" {
		t.Errorf("expected 'New Name', got '%s'", data.Workspaces[0].Name)
	}
}

func TestDeleteWorkspace(t *testing.T) {
	svc := newTestService()
	ws, _ := svc.CreateWorkspace("To Delete")
	err := svc.DeleteWorkspace(ws.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := svc.GetWorkspaces()
	if len(data.Workspaces) != 0 {
		t.Errorf("expected 0 workspaces, got %d", len(data.Workspaces))
	}
}

func TestDeleteActiveWorkspaceClearsActiveID(t *testing.T) {
	svc := newTestService()
	ws, _ := svc.CreateWorkspace("Active")
	_ = svc.SetActiveWorkspace(ws.ID)
	_ = svc.DeleteWorkspace(ws.ID)

	data, _ := svc.GetWorkspaces()
	if data.ActiveWorkspaceID != "" {
		t.Errorf("expected empty active ID, got '%s'", data.ActiveWorkspaceID)
	}
}

func TestSetActiveWorkspace(t *testing.T) {
	svc := newTestService()
	ws, _ := svc.CreateWorkspace("My Workspace")
	err := svc.SetActiveWorkspace(ws.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := svc.GetWorkspaces()
	if data.ActiveWorkspaceID != ws.ID {
		t.Errorf("expected active ID '%s', got '%s'", ws.ID, data.ActiveWorkspaceID)
	}
}

func TestAddFolder(t *testing.T) {
	svc := newTestService()
	ws, _ := svc.CreateWorkspace("With Folders")
	err := svc.AddFolder(ws.ID, "/home/user/queries")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := svc.GetWorkspaces()
	if len(data.Workspaces[0].Folders) != 1 {
		t.Errorf("expected 1 folder, got %d", len(data.Workspaces[0].Folders))
	}
	if data.Workspaces[0].Folders[0] != "/home/user/queries" {
		t.Errorf("expected '/home/user/queries', got '%s'", data.Workspaces[0].Folders[0])
	}
}

func TestAddDuplicateFolder(t *testing.T) {
	svc := newTestService()
	ws, _ := svc.CreateWorkspace("Dup")
	_ = svc.AddFolder(ws.ID, "/home/user/queries")
	err := svc.AddFolder(ws.ID, "/home/user/queries")
	if err == nil {
		t.Error("expected error for duplicate folder")
	}
}

func TestRemoveFolder(t *testing.T) {
	svc := newTestService()
	ws, _ := svc.CreateWorkspace("Remove")
	_ = svc.AddFolder(ws.ID, "/home/user/queries")
	err := svc.RemoveFolder(ws.ID, "/home/user/queries")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := svc.GetWorkspaces()
	if len(data.Workspaces[0].Folders) != 0 {
		t.Errorf("expected 0 folders, got %d", len(data.Workspaces[0].Folders))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/workspaces/... -v`
Expected: compilation errors — `WorkspaceService` doesn't exist yet.

- [ ] **Step 3: Implement the workspace service**

Create `internal/workspaces/service.go`:

```go
package workspaces

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
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

	for i := range data.Workspaces {
		if data.Workspaces[i].ID == id {
			data.Workspaces[i].Name = name
			return s.store.Save(data)
		}
	}
	return fmt.Errorf("workspace not found: %s", id)
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

	for i := range data.Workspaces {
		if data.Workspaces[i].ID == workspaceID {
			if slices.Contains(data.Workspaces[i].Folders, path) {
				return fmt.Errorf("folder already in workspace: %s", path)
			}
			data.Workspaces[i].Folders = append(data.Workspaces[i].Folders, path)
			return s.store.Save(data)
		}
	}
	return fmt.Errorf("workspace not found: %s", workspaceID)
}

func (s *WorkspaceService) RemoveFolder(workspaceID, path string) error {
	data, err := s.store.Load()
	if err != nil {
		return err
	}

	for i := range data.Workspaces {
		if data.Workspaces[i].ID == workspaceID {
			idx := slices.Index(data.Workspaces[i].Folders, path)
			if idx == -1 {
				return fmt.Errorf("folder not found: %s", path)
			}
			data.Workspaces[i].Folders = slices.Delete(data.Workspaces[i].Folders, idx, idx+1)
			return s.store.Save(data)
		}
	}
	return fmt.Errorf("workspace not found: %s", workspaceID)
}

func (s *WorkspaceService) ReadDirectory(dirPath string, extensions []string) ([]models.DirectoryEntry, error) {
	if len(extensions) == 0 {
		extensions = defaultExtensions
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var result []models.DirectoryEntry
	var dirs []models.DirectoryEntry
	var files []models.DirectoryEntry

	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry.Name())

		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			dirs = append(dirs, models.DirectoryEntry{
				Name:        entry.Name(),
				Path:        fullPath,
				IsDirectory: true,
			})
		} else {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if slices.Contains(extensions, ext) {
				files = append(files, models.DirectoryEntry{
					Name:        entry.Name(),
					Path:        fullPath,
					IsDirectory: false,
				})
			}
		}
	}

	// Folders first, then files (both already alphabetical from os.ReadDir)
	result = append(result, dirs...)
	result = append(result, files...)
	return result, nil
}

func (s *WorkspaceService) CreateFile(dirPath, name string) (string, error) {
	fullPath := filepath.Join(dirPath, name)
	if _, err := os.Stat(fullPath); err == nil {
		return "", fmt.Errorf("file already exists: %s", name)
	}
	if err := os.WriteFile(fullPath, []byte(""), 0644); err != nil {
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/workspaces/... -v`
Expected: all tests pass.

- [ ] **Step 5: Write tests for ReadDirectory and file operations**

Add to `internal/workspaces/service_test.go`:

```go
func TestReadDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "query.js"), []byte("db.test.find()"), 0644)
	os.WriteFile(filepath.Join(dir, "other.txt"), []byte("not a query"), 0644)
	os.WriteFile(filepath.Join(dir, "script.mongodb"), []byte("show dbs"), 0644)
	os.Mkdir(filepath.Join(dir, "subfolder"), 0755)

	svc := newTestService()
	entries, err := svc.ReadDirectory(dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have: subfolder, query.js, script.mongodb (not other.txt)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	// Folders first
	if !entries[0].IsDirectory || entries[0].Name != "subfolder" {
		t.Errorf("expected first entry to be subfolder dir, got %+v", entries[0])
	}
	// Then files alphabetically
	if entries[1].Name != "query.js" {
		t.Errorf("expected 'query.js', got '%s'", entries[1].Name)
	}
	if entries[2].Name != "script.mongodb" {
		t.Errorf("expected 'script.mongodb', got '%s'", entries[2].Name)
	}
}

func TestReadDirectoryCustomExtensions(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "query.js"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "custom.mql"), []byte(""), 0644)

	svc := newTestService()
	entries, err := svc.ReadDirectory(dir, []string{".mql"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 1 || entries[0].Name != "custom.mql" {
		t.Errorf("expected only custom.mql, got %+v", entries)
	}
}

func TestCreateFile(t *testing.T) {
	dir := t.TempDir()
	svc := newTestService()

	path, err := svc.CreateFile(dir, "new-query.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(path) != "new-query.js" {
		t.Errorf("expected 'new-query.js', got '%s'", filepath.Base(path))
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("file was not created")
	}
}

func TestCreateFileDuplicate(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "exists.js"), []byte(""), 0644)

	svc := newTestService()
	_, err := svc.CreateFile(dir, "exists.js")
	if err == nil {
		t.Error("expected error for duplicate file")
	}
}

func TestRenameFile(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "old.js")
	os.WriteFile(old, []byte("content"), 0644)

	svc := newTestService()
	newPath := filepath.Join(dir, "new.js")
	err := svc.RenameFile(old, newPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Error("renamed file does not exist")
	}
}

func TestDeleteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "delete-me.js")
	os.WriteFile(path, []byte(""), 0644)

	svc := newTestService()
	err := svc.DeleteFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file was not deleted")
	}
}
```

Add these imports to the top of the test file:

```go
import (
	"os"
	"path/filepath"
	"testing"

	"vervet/internal/models"
)
```

- [ ] **Step 6: Run all tests**

Run: `go test ./internal/workspaces/... -v`
Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/workspaces/service.go internal/workspaces/service_test.go
git commit -m "feat(workspaces): add workspace service with CRUD, directory reading, and file ops"
```

---

### Task 4: Workspaces API Proxy (Go)

**Files:**
- Create: `internal/api/workspaces.go`
- Modify: `internal/app/app.go`

- [ ] **Step 1: Create the WorkspacesProxy**

Create `internal/api/workspaces.go` following the pattern from `internal/api/servers.go`:

```go
package api

import (
	"context"

	"vervet/internal/models"
	"vervet/internal/workspaces"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type WorkspacesProxy struct {
	ctx     context.Context
	service *workspaces.WorkspaceService
	settingsProvider SettingsProvider
}

func NewWorkspacesProxy(service *workspaces.WorkspaceService, settingsProvider SettingsProvider) *WorkspacesProxy {
	return &WorkspacesProxy{service: service, settingsProvider: settingsProvider}
}

func (p *WorkspacesProxy) Init(ctx context.Context) {
	p.ctx = ctx
}

func (p *WorkspacesProxy) GetWorkspaces() Result[models.WorkspaceData] {
	data, err := p.service.GetWorkspaces()
	if err != nil {
		return FailResult[models.WorkspaceData](err)
	}
	return SuccessResult(data)
}

func (p *WorkspacesProxy) CreateWorkspace(name string) Result[models.Workspace] {
	ws, err := p.service.CreateWorkspace(name)
	if err != nil {
		return FailResult[models.Workspace](err)
	}
	return SuccessResult(ws)
}

func (p *WorkspacesProxy) RenameWorkspace(id, name string) EmptyResult {
	if err := p.service.RenameWorkspace(id, name); err != nil {
		return Fail(err)
	}
	return Success()
}

func (p *WorkspacesProxy) DeleteWorkspace(id string) EmptyResult {
	if err := p.service.DeleteWorkspace(id); err != nil {
		return Fail(err)
	}
	return Success()
}

func (p *WorkspacesProxy) SetActiveWorkspace(id string) EmptyResult {
	if err := p.service.SetActiveWorkspace(id); err != nil {
		return Fail(err)
	}
	return Success()
}

func (p *WorkspacesProxy) AddFolder(workspaceID string) Result[string] {
	path, err := wailsRuntime.OpenDirectoryDialog(p.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Folder",
	})
	if err != nil {
		return FailResult[string](err)
	}
	if path == "" {
		return FailResult[string](fmt.Errorf("no folder selected"))
	}

	if err := p.service.AddFolder(workspaceID, path); err != nil {
		return FailResult[string](err)
	}
	return SuccessResult(path)
}

func (p *WorkspacesProxy) RemoveFolder(workspaceID, path string) EmptyResult {
	if err := p.service.RemoveFolder(workspaceID, path); err != nil {
		return Fail(err)
	}
	return Success()
}

func (p *WorkspacesProxy) ReadDirectory(path string) Result[[]models.DirectoryEntry] {
	settings, err := p.settingsProvider.GetSettings()
	if err != nil {
		return FailResult[[]models.DirectoryEntry](err)
	}

	entries, err := p.service.ReadDirectory(path, settings.Workspaces.FileExtensions)
	if err != nil {
		return FailResult[[]models.DirectoryEntry](err)
	}
	return SuccessResult(entries)
}

func (p *WorkspacesProxy) CreateFile(dirPath, name string) Result[string] {
	path, err := p.service.CreateFile(dirPath, name)
	if err != nil {
		return FailResult[string](err)
	}
	return SuccessResult(path)
}

func (p *WorkspacesProxy) RenameFile(oldPath, newPath string) EmptyResult {
	if err := p.service.RenameFile(oldPath, newPath); err != nil {
		return Fail(err)
	}
	return Success()
}

func (p *WorkspacesProxy) DeleteFile(path string) EmptyResult {
	if err := p.service.DeleteFile(path); err != nil {
		return Fail(err)
	}
	return Success()
}
```

Note: Use the existing `SettingsProvider` interface already defined in `internal/api/settings.go` — it includes `GetSettings() (models.Settings, error)` which is all we need. Do **not** define a new interface; reuse the existing one.

Add `"fmt"` to the import block.

- [ ] **Step 2: Wire into app.go**

In `internal/app/app.go`:

1. Add field to `App` struct:
```go
WorkspacesProxy *api.WorkspacesProxy
```

2. In `NewApp()`, create the workspace store and service:
```go
workspaceStore, err := workspaces.NewStore(log)
if err != nil {
	log.Error("failed to create workspace store", "error", err)
}
workspaceService := workspaces.NewService(log, workspaceStore)
```

3. Wire the proxy:
```go
WorkspacesProxy: api.NewWorkspacesProxy(workspaceService, settingsService),
```

4. In `Startup()`, initialise the proxy:
```go
a.WorkspacesProxy.Init(ctx)
```

5. Add to the Wails `Bind` list in `main.go` (or wherever bindings are declared) — check how other proxies are bound and follow the same pattern.

- [ ] **Step 3: Verify it compiles**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/api/workspaces.go internal/app/app.go
git commit -m "feat(workspaces): add WorkspacesProxy and wire into app"
```

---

### Task 5: i18n Translation Keys

**Files:**
- Modify: `frontend/src/i18n/en-GB/index.ts`

- [ ] **Step 1: Add workspace translation keys**

Add the following sections to the translations file:

```typescript
workspaces: {
  name: 'Workspaces',
  createWorkspace: 'Create Workspace',
  renameWorkspace: 'Rename Workspace',
  deleteWorkspace: 'Delete Workspace',
  deleteWorkspaceConfirm: 'Are you sure you want to delete this workspace? The folders on disk will not be affected.',
  addFolder: 'Add Folder',
  removeFromWorkspace: 'Remove from Workspace',
  refresh: 'Refresh',
  open: 'Open',
  openOnServer: 'Open on Server...',
  newFile: 'New File',
  rename: 'Rename',
  delete: 'Delete',
  deleteConfirm: 'Are you sure you want to delete "{name}"? This cannot be undone.',
  newFileName: 'Enter file name',
  renamePrompt: 'Enter new name',
  noQueryFiles: '(no query files)',
  folderNotFound: '(folder not found)',
  emptyNoWorkspaces: 'Create a workspace to organise your query files',
  emptyNoFolders: 'Add a folder to get started',
  connectFirst: 'Connect to a server first',
  selectServer: 'Select Server',
  selectDatabase: 'Select Database',
  defaultWorkspaceName: 'New Workspace',
},
settings: {
  // ... existing settings keys ...
  workspaces: {
    name: 'Workspaces',
    fileExtensions: 'File Extensions',
    fileExtensionsTip: 'File types shown in the workspace tree',
    resetDefaults: 'Reset to Defaults',
  },
},
```

Add `workspaces` as a new top-level key. Add `settings.workspaces` nested under the existing `settings` key.

Also add to `ribbon`:
```typescript
ribbon: {
  // ... existing keys ...
  workspaces: 'Workspaces',
},
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/i18n/en-GB/index.ts
git commit -m "feat(workspaces): add i18n translation keys"
```

---

### Task 6: NavType + Dialog Store + Tab Store Updates

**Files:**
- Modify: `frontend/src/features/tabs/tabs.ts`
- Modify: `frontend/src/stores/dialog.ts`

- [ ] **Step 1: Add Workspaces to NavType and modify openQuery return**

In `frontend/src/features/tabs/tabs.ts`:

1. Add to the `NavType` enum:

```typescript
export enum NavType {
  Servers = 'servers',
  Browser = 'browser',
  Workspaces = 'workspaces',
}
```

2. Modify `openQuery` to return the query tab ID. Find the `openQuery` action and add `return queryItem.id` as the last line of the function, changing the return type to `string`. This is needed by the `ServerPickerDialog` to call `loadFileByPath` on the correct query tab.

- [ ] **Step 2: Add ServerPicker to DialogType**

In `frontend/src/stores/dialog.ts`, add to the `DialogType` enum:

```typescript
ServerPicker = 'serverPicker',
```

And add a convenience method if desired:

```typescript
openServerPickerDialog(data?: unknown) {
  this.showNewDialog(DialogType.ServerPicker, data)
},
```

- [ ] **Step 3: Verify frontend compiles**

Run from `frontend/`: `bun run lint`
Expected: no type errors related to these changes.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/features/tabs/tabs.ts frontend/src/stores/dialog.ts
git commit -m "feat(workspaces): add Workspaces NavType and ServerPicker DialogType"
```

---

### Task 7: Query Store — `loadFileByPath`

**Files:**
- Modify: `frontend/src/features/queries/queryStore.ts`

- [ ] **Step 1: Add loadFileByPath action**

In `frontend/src/features/queries/queryStore.ts`, add a new action alongside the existing `openFile`:

```typescript
async loadFileByPath(queryId: string, filePath: string): Promise<boolean> {
  const result = await filesProxy.ReadFile(filePath)
  if (!result.isSuccess) {
    useNotifier().error(result.errorDetail || 'Failed to read file')
    return false
  }

  const state = this.getQueryState(queryId)
  if (!state) {
    return false
  }

  state.filePath = filePath
  state.savedContent = result.data
  state.currentContent = result.data
  state.isDirty = false
  return true
},
```

This reads a file by its known path (unlike `openFile` which shows a file picker dialog) and sets the query state so dirty-tracking and tab labelling work correctly.

- [ ] **Step 2: Verify frontend compiles**

Run from `frontend/`: `bun run lint`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/features/queries/queryStore.ts
git commit -m "feat(workspaces): add loadFileByPath action to query store"
```

---

### Task 8: Workspace Pinia Store

**Files:**
- Create: `frontend/src/features/workspaces/workspaceStore.ts`

- [ ] **Step 1: Create the workspace store**

Create `frontend/src/features/workspaces/workspaceStore.ts`:

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TreeOption } from 'naive-ui'
import { GetWorkspaces, CreateWorkspace, RenameWorkspace, DeleteWorkspace, SetActiveWorkspace, AddFolder, RemoveFolder, ReadDirectory } from 'wailsjs/go/api/WorkspacesProxy'
import { useNotifier } from '@/utils/notifier'

// These types mirror the Go models — Wails generates them, but define locally
// until bindings are generated. Replace with imports from wailsjs/go/models once available.
interface Workspace {
  id: string
  name: string
  folders: string[]
}

interface WorkspaceData {
  activeWorkspaceId: string
  workspaces: Workspace[]
}

interface DirectoryEntry {
  name: string
  path: string
  isDirectory: boolean
  children?: DirectoryEntry[]
}

export const useWorkspaceStore = defineStore('workspaces', () => {
  const notifier = useNotifier()

  const workspaces = ref<Workspace[]>([])
  const activeWorkspaceId = ref<string | null>(null)
  const treeData = ref<TreeOption[]>([])
  const expandedKeys = ref<string[]>([])
  const loading = ref(false)

  const activeWorkspace = computed(() =>
    workspaces.value.find(w => w.id === activeWorkspaceId.value) ?? null
  )

  const hasWorkspaces = computed(() => workspaces.value.length > 0)

  async function loadWorkspaces() {
    const result = await GetWorkspaces()
    if (!result.isSuccess) {
      notifier.error(result.errorDetail || 'Failed to load workspaces')
      return
    }
    workspaces.value = result.data.workspaces || []
    activeWorkspaceId.value = result.data.activeWorkspaceId || null

    if (activeWorkspace.value) {
      await loadTree()
    }
  }

  async function createWorkspace(name: string) {
    const result = await CreateWorkspace(name)
    if (!result.isSuccess) {
      notifier.error(result.errorDetail || 'Failed to create workspace')
      return
    }
    workspaces.value.push(result.data)
    await setActiveWorkspace(result.data.id)
  }

  async function renameWorkspace(id: string, name: string) {
    const result = await RenameWorkspace(id, name)
    if (!result.isSuccess) {
      notifier.error(result.errorDetail || 'Failed to rename workspace')
      return
    }
    const ws = workspaces.value.find(w => w.id === id)
    if (ws) {
      ws.name = name
    }
  }

  async function deleteWorkspace(id: string) {
    const result = await DeleteWorkspace(id)
    if (!result.isSuccess) {
      notifier.error(result.errorDetail || 'Failed to delete workspace')
      return
    }
    workspaces.value = workspaces.value.filter(w => w.id !== id)
    if (activeWorkspaceId.value === id) {
      activeWorkspaceId.value = workspaces.value.length > 0
        ? workspaces.value[0].id
        : null
      if (activeWorkspace.value) {
        await setActiveWorkspace(activeWorkspace.value.id)
      } else {
        treeData.value = []
      }
    }
  }

  async function setActiveWorkspace(id: string) {
    const result = await SetActiveWorkspace(id)
    if (!result.isSuccess) {
      notifier.error(result.errorDetail || 'Failed to switch workspace')
      return
    }
    activeWorkspaceId.value = id
    expandedKeys.value = []
    await loadTree()
  }

  async function addFolder() {
    if (!activeWorkspaceId.value) {
      return
    }
    const result = await AddFolder(activeWorkspaceId.value)
    if (!result.isSuccess) {
      // User may have cancelled the dialog — don't show error for empty selection
      if (result.errorDetail !== 'no folder selected') {
        notifier.error(result.errorDetail || 'Failed to add folder')
      }
      return
    }
    const ws = activeWorkspace.value
    if (ws) {
      ws.folders.push(result.data)
    }
    await loadTree()
  }

  async function removeFolder(path: string) {
    if (!activeWorkspaceId.value) {
      return
    }
    const result = await RemoveFolder(activeWorkspaceId.value, path)
    if (!result.isSuccess) {
      notifier.error(result.errorDetail || 'Failed to remove folder')
      return
    }
    const ws = activeWorkspace.value
    if (ws) {
      ws.folders = ws.folders.filter(f => f !== path)
    }
    await loadTree()
  }

  async function loadTree() {
    const ws = activeWorkspace.value
    if (!ws) {
      treeData.value = []
      return
    }

    loading.value = true
    const roots: TreeOption[] = []

    for (const folder of ws.folders) {
      const result = await ReadDirectory(folder)
      if (!result.isSuccess) {
        // Folder may not exist — show placeholder
        roots.push({
          key: folder,
          label: folder.split('/').pop() || folder,
          isLeaf: true,
          disabled: true,
          prefix: () => '📁',
        })
        continue
      }
      roots.push({
        key: folder,
        label: folder.split('/').pop() || folder,
        children: entriesToTreeOptions(result.data),
        isLeaf: false,
      })
    }

    treeData.value = roots
    loading.value = false
  }

  async function loadDirectory(path: string): Promise<TreeOption[]> {
    const result = await ReadDirectory(path)
    if (!result.isSuccess) {
      return []
    }
    return entriesToTreeOptions(result.data)
  }

  async function refreshTree() {
    expandedKeys.value = []
    await loadTree()
  }

  function entriesToTreeOptions(entries: DirectoryEntry[]): TreeOption[] {
    return entries.map(entry => ({
      key: entry.path,
      label: entry.name,
      isLeaf: !entry.isDirectory,
    }))
  }

  return {
    workspaces,
    activeWorkspaceId,
    activeWorkspace,
    hasWorkspaces,
    treeData,
    expandedKeys,
    loading,
    loadWorkspaces,
    createWorkspace,
    renameWorkspace,
    deleteWorkspace,
    setActiveWorkspace,
    addFolder,
    removeFolder,
    loadTree,
    loadDirectory,
    refreshTree,
  }
})
```

- [ ] **Step 2: Verify frontend compiles**

Run from `frontend/`: `bun run lint`
Expected: may show warnings about missing wailsjs bindings (they're generated later by `wails dev`). Type errors in the store logic itself should be clean.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/features/workspaces/workspaceStore.ts
git commit -m "feat(workspaces): add workspace Pinia store"
```

---

### Task 9: WorkspaceEmptyState Component

**Files:**
- Create: `frontend/src/features/workspaces/WorkspaceEmptyState.vue`

- [ ] **Step 1: Create the empty state component**

Create `frontend/src/features/workspaces/WorkspaceEmptyState.vue`:

```vue
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { FolderPlusIcon } from '@heroicons/vue/24/outline'

const { t } = useI18n()

defineProps<{
  type: 'no-workspaces' | 'no-folders'
}>()

defineEmits<{
  createWorkspace: []
  addFolder: []
}>()
</script>

<template>
  <div class="empty-state">
    <n-icon :size="48" :depth="3">
      <folder-plus-icon />
    </n-icon>
    <p v-if="type === 'no-workspaces'">{{ t('workspaces.emptyNoWorkspaces') }}</p>
    <p v-else>{{ t('workspaces.emptyNoFolders') }}</p>
    <n-button
      v-if="type === 'no-workspaces'"
      type="primary"
      @click="$emit('createWorkspace')"
    >
      {{ t('workspaces.createWorkspace') }}
    </n-button>
    <n-button
      v-else
      type="primary"
      @click="$emit('addFolder')"
    >
      {{ t('workspaces.addFolder') }}
    </n-button>
  </div>
</template>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 12px;
  padding: 24px;
  text-align: center;
  color: var(--n-text-color-3);
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/features/workspaces/WorkspaceEmptyState.vue
git commit -m "feat(workspaces): add empty state component"
```

---

### Task 10: WorkspaceToolbar Component

**Files:**
- Create: `frontend/src/features/workspaces/WorkspaceToolbar.vue`

- [ ] **Step 1: Create the toolbar component**

Create `frontend/src/features/workspaces/WorkspaceToolbar.vue`:

```vue
<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  PlusIcon,
  Cog6ToothIcon,
  FolderPlusIcon,
  ArrowPathIcon,
} from '@heroicons/vue/24/outline'
import { useWorkspaceStore } from './workspaceStore'
import { useDialoger } from '@/utils/dialog'
import IconButton from '@/features/common/IconButton.vue'

const { t } = useI18n()
const workspaceStore = useWorkspaceStore()

const showGearMenu = ref(false)
const renameMode = ref(false)
const renameValue = ref('')

const workspaceOptions = computed(() =>
  workspaceStore.workspaces.map(w => ({
    label: w.name,
    value: w.id,
  }))
)

const gearOptions = computed(() => [
  { label: t('workspaces.renameWorkspace'), key: 'rename' },
  { label: t('workspaces.deleteWorkspace'), key: 'delete' },
])

function onSelectWorkspace(id: string) {
  workspaceStore.setActiveWorkspace(id)
}

function onCreateWorkspace() {
  workspaceStore.createWorkspace(t('workspaces.defaultWorkspaceName'))
}

function onGearSelect(key: string) {
  showGearMenu.value = false
  if (key === 'rename') {
    renameMode.value = true
    renameValue.value = workspaceStore.activeWorkspace?.name ?? ''
  } else if (key === 'delete') {
    handleDelete()
  }
}

function onRenameConfirm() {
  if (workspaceStore.activeWorkspaceId && renameValue.value.trim()) {
    workspaceStore.renameWorkspace(workspaceStore.activeWorkspaceId, renameValue.value.trim())
  }
  renameMode.value = false
}

function handleDelete() {
  if (!workspaceStore.activeWorkspaceId) {
    return
  }
  const dialoger = useDialoger()
  dialoger.warning(t('workspaces.deleteWorkspaceConfirm'), () => {
    if (workspaceStore.activeWorkspaceId) {
      workspaceStore.deleteWorkspace(workspaceStore.activeWorkspaceId)
    }
  })
}
</script>

<template>
  <div class="workspace-toolbar">
    <!-- Workspace selector or rename input -->
    <div class="selector-row">
      <n-input
        v-if="renameMode"
        v-model:value="renameValue"
        size="small"
        autofocus
        @blur="onRenameConfirm"
        @keyup.enter="onRenameConfirm"
        @keyup.escape="renameMode = false"
      />
      <n-select
        v-else
        :value="workspaceStore.activeWorkspaceId"
        :options="workspaceOptions"
        size="small"
        :placeholder="t('workspaces.name')"
        @update:value="onSelectWorkspace"
      />
      <icon-button :tooltip="t('workspaces.createWorkspace')" @click="onCreateWorkspace">
        <plus-icon />
      </icon-button>
      <n-dropdown
        v-if="workspaceStore.activeWorkspace"
        :options="gearOptions"
        trigger="click"
        @select="onGearSelect"
      >
        <icon-button :tooltip="t('settings.name')">
          <cog6-tooth-icon />
        </icon-button>
      </n-dropdown>
    </div>

    <!-- Folder actions -->
    <div v-if="workspaceStore.activeWorkspace" class="actions-row">
      <icon-button :tooltip="t('workspaces.addFolder')" @click="workspaceStore.addFolder()">
        <folder-plus-icon />
      </icon-button>
      <icon-button :tooltip="t('workspaces.refresh')" @click="workspaceStore.refreshTree()">
        <arrow-path-icon />
      </icon-button>
    </div>
  </div>
</template>

<style scoped>
.workspace-toolbar {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px;
  border-bottom: 1px solid var(--n-border-color);
}

.selector-row {
  display: flex;
  align-items: center;
  gap: 4px;
}

.selector-row .n-select,
.selector-row .n-input {
  flex: 1;
}

.actions-row {
  display: flex;
  align-items: center;
  gap: 4px;
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/features/workspaces/WorkspaceToolbar.vue
git commit -m "feat(workspaces): add workspace toolbar component"
```

---

### Task 11: WorkspaceTree Component

**Files:**
- Create: `frontend/src/features/workspaces/WorkspaceTree.vue`

- [ ] **Step 1: Create the tree component**

Create `frontend/src/features/workspaces/WorkspaceTree.vue`:

```vue
<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TreeOption, TreeDropInfo } from 'naive-ui'
import { useWorkspaceStore } from './workspaceStore'
import { useTabStore } from '@/features/tabs/tabs'
import { useDialogStore, DialogType } from '@/stores/dialog'
import { CreateFile, RenameFile, DeleteFile } from 'wailsjs/go/api/WorkspacesProxy'
import { useDialoger, useNotifier } from '@/utils/dialog'

const { t } = useI18n()
const workspaceStore = useWorkspaceStore()
const tabStore = useTabStore()
const dialogStore = useDialogStore()
const notifier = useNotifier()

const showContextMenu = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)
const contextNode = ref<TreeOption | null>(null)

const isRootFolder = (node: TreeOption) => {
  const ws = workspaceStore.activeWorkspace
  return ws ? ws.folders.includes(node.key as string) : false
}

const contextMenuOptions = computed(() => {
  const node = contextNode.value
  if (!node) {
    return []
  }

  if (node.isLeaf) {
    // File
    return [
      { label: t('workspaces.open'), key: 'open' },
      { label: t('workspaces.openOnServer'), key: 'openOnServer' },
      { type: 'divider', key: 'd1' },
      { label: t('workspaces.rename'), key: 'rename' },
      { label: t('workspaces.delete'), key: 'delete' },
    ]
  }

  if (isRootFolder(node)) {
    // Root folder
    return [
      { label: t('workspaces.newFile'), key: 'newFile' },
      { label: t('workspaces.refresh'), key: 'refresh' },
      { type: 'divider', key: 'd1' },
      { label: t('workspaces.removeFromWorkspace'), key: 'removeFromWorkspace' },
    ]
  }

  // Subfolder
  return [
    { label: t('workspaces.newFile'), key: 'newFile' },
    { label: t('workspaces.refresh'), key: 'refresh' },
    { type: 'divider', key: 'd1' },
    { label: t('workspaces.rename'), key: 'rename' },
    { label: t('workspaces.delete'), key: 'delete' },
  ]
})

function onContextMenu(e: MouseEvent, node: TreeOption) {
  e.preventDefault()
  contextNode.value = node
  contextMenuX.value = e.clientX
  contextMenuY.value = e.clientY
  showContextMenu.value = true
}

function onContextMenuSelect(key: string) {
  showContextMenu.value = false
  const node = contextNode.value
  if (!node) {
    return
  }

  const path = node.key as string

  switch (key) {
    case 'open':
      handleOpenFile(path, false)
      break
    case 'openOnServer':
      handleOpenFile(path, true)
      break
    case 'newFile':
      handleNewFile(path)
      break
    case 'rename':
      handleRename(path, node.label as string)
      break
    case 'delete':
      handleDelete(path, node.label as string)
      break
    case 'refresh':
      workspaceStore.refreshTree()
      break
    case 'removeFromWorkspace':
      workspaceStore.removeFolder(path)
      break
  }
}

function handleOpenFile(filePath: string, forceServerPicker: boolean) {
  const currentTab = tabStore.currentTab
  if (!forceServerPicker && currentTab) {
    // Open on active server — show database picker via ServerPickerDialog
    // Pass the filePath and serverId as dialog data
    dialogStore.showNewDialog(DialogType.ServerPicker, {
      filePath,
      serverId: currentTab.serverId,
      skipServerSelection: true,
    })
  } else {
    // Show full server + database picker
    dialogStore.showNewDialog(DialogType.ServerPicker, { filePath })
  }
}

function handleNewFile(dirPath: string) {
  const dialoger = useDialoger()
  const inputValue = ref('new-query.js')
  dialoger.show({
    title: t('workspaces.newFile'),
    content: () =>
      h(NInput, {
        value: inputValue.value,
        onUpdateValue: (v: string) => { inputValue.value = v },
        placeholder: t('workspaces.newFileName'),
      }),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      const name = inputValue.value.trim()
      if (!name) {
        return
      }
      const result = await CreateFile(dirPath, name)
      if (!result.isSuccess) {
        notifier.error(result.errorDetail || 'Failed to create file')
        return
      }
      await workspaceStore.refreshTree()
    },
  })
}

function handleRename(path: string, currentName: string) {
  const dialoger = useDialoger()
  const inputValue = ref(currentName)
  dialoger.show({
    title: t('workspaces.rename'),
    content: () =>
      h(NInput, {
        value: inputValue.value,
        onUpdateValue: (v: string) => { inputValue.value = v },
        placeholder: t('workspaces.renamePrompt'),
      }),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      const newName = inputValue.value.trim()
      if (!newName || newName === currentName) {
        return
      }
      const dir = path.substring(0, path.lastIndexOf('/'))
      const newPath = `${dir}/${newName}`
      const result = await RenameFile(path, newPath)
      if (!result.isSuccess) {
        notifier.error(result.errorDetail || 'Failed to rename')
        return
      }
      await workspaceStore.refreshTree()
    },
  })
}

function handleDelete(path: string, name: string) {
  const dialoger = useDialoger()
  dialoger.warning(t('workspaces.deleteConfirm', { name }), async () => {
    const result = await DeleteFile(path)
    if (!result.isSuccess) {
      notifier.error(result.errorDetail || 'Failed to delete')
      return
    }
    await workspaceStore.refreshTree()
  })
}

async function onNodeExpand(keys: string[], option: Array<TreeOption | null>) {
  workspaceStore.expandedKeys = keys
  // Lazy load children for newly expanded directories
  for (const node of option) {
    if (node && !node.isLeaf && !node.children?.length) {
      const children = await workspaceStore.loadDirectory(node.key as string)
      node.children = children
    }
  }
}

function onNodeDblClick(_: MouseEvent, node: TreeOption) {
  if (node.isLeaf) {
    handleOpenFile(node.key as string, false)
  }
}
</script>

<template>
  <div class="workspace-tree">
    <n-tree
      :data="workspaceStore.treeData"
      :expanded-keys="workspaceStore.expandedKeys"
      block-line
      :on-update:expanded-keys="onNodeExpand"
      :node-props="({ option }: { option: TreeOption }) => ({
        onContextmenu: (e: MouseEvent) => onContextMenu(e, option),
        onDblclick: (e: MouseEvent) => onNodeDblClick(e, option),
      })"
    />

    <n-dropdown
      trigger="manual"
      :show="showContextMenu"
      :options="contextMenuOptions"
      :x="contextMenuX"
      :y="contextMenuY"
      placement="bottom-start"
      @select="onContextMenuSelect"
      @clickoutside="showContextMenu = false"
    />
  </div>
</template>

<style scoped>
.workspace-tree {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}
</style>
```

Note: Add `import { h, ref } from 'vue'` and `import { NInput } from 'naive-ui'` at the top of the script. The `h` function is needed for rendering dialog content programmatically. The codebase uses `useDialoger()` from `@/utils/dialog` (not `window.$dialog`) — `dialoger.warning(content, onConfirm)` for simple confirmations and `dialoger.show(options)` for dialogs with custom content.

- [ ] **Step 2: Commit**

```bash
git add frontend/src/features/workspaces/WorkspaceTree.vue
git commit -m "feat(workspaces): add workspace tree with context menu and lazy loading"
```

---

### Task 12: ServerPickerDialog Component

**Files:**
- Create: `frontend/src/features/workspaces/ServerPickerDialog.vue`

- [ ] **Step 1: Create the server/database picker dialog**

Create `frontend/src/features/workspaces/ServerPickerDialog.vue`:

```vue
<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDialogStore, DialogType } from '@/stores/dialog'
import { useDataBrowserStore } from '@/features/data-browser/browserStore'
import { useTabStore } from '@/features/tabs/tabs'
import { useQueryStore } from '@/features/queries/queryStore'
import { GetDatabases } from 'wailsjs/go/api/DatabasesProxy'

const { t } = useI18n()
const dialogStore = useDialogStore()
const browserStore = useDataBrowserStore()
const tabStore = useTabStore()
const queryStore = useQueryStore()

const selectedServerId = ref<string | null>(null)
const selectedDatabase = ref<string | null>(null)
const databases = ref<string[]>([])
const loadingDatabases = ref(false)

interface PickerDialogData {
  filePath: string
  serverId?: string
  skipServerSelection?: boolean
}

const dialogData = computed(() =>
  dialogStore.getDialogData<PickerDialogData>(DialogType.ServerPicker)
)

const visible = computed(() => dialogStore.isVisible(DialogType.ServerPicker))

const serverOptions = computed(() =>
  browserStore.connections.map(c => ({
    label: c.name,
    value: c.serverID,
  }))
)

const databaseOptions = computed(() =>
  databases.value.map(db => ({
    label: db,
    value: db,
  }))
)

const hasConnections = computed(() => browserStore.connections.length > 0)

watch(visible, (val) => {
  if (val) {
    selectedDatabase.value = null
    databases.value = []
    const data = dialogData.value
    if (data?.skipServerSelection && data.serverId) {
      selectedServerId.value = data.serverId
      loadDatabases(data.serverId)
    } else {
      selectedServerId.value = null
    }
  }
})

async function loadDatabases(serverId: string) {
  loadingDatabases.value = true
  const result = await GetDatabases(serverId)
  loadingDatabases.value = false
  if (result.isSuccess) {
    // GetDatabases returns Result[[]string] — data is already string[]
    databases.value = result.data
  }
}

function onServerChange(serverId: string) {
  selectedServerId.value = serverId
  selectedDatabase.value = null
  loadDatabases(serverId)
}

async function onConfirm() {
  const data = dialogData.value
  if (!selectedServerId.value || !selectedDatabase.value || !data?.filePath) {
    return
  }

  const serverId = selectedServerId.value
  const database = selectedDatabase.value
  const filePath = data.filePath

  // Open query tab on selected server + database
  const queryId = tabStore.openQuery(serverId, database)
  if (queryId) {
    await queryStore.loadFileByPath(queryId, filePath)
  }

  dialogStore.hide(DialogType.ServerPicker)
}

function onClose() {
  dialogStore.hide(DialogType.ServerPicker)
}
</script>

<template>
  <n-modal
    :show="visible"
    preset="dialog"
    :title="t('workspaces.selectServer')"
    :positive-text="t('common.confirm')"
    :negative-text="t('common.cancel')"
    :positive-button-props="{ disabled: !selectedServerId || !selectedDatabase }"
    @positive-click="onConfirm"
    @negative-click="onClose"
    @close="onClose"
  >
    <div v-if="!hasConnections" class="no-connections">
      <p>{{ t('workspaces.connectFirst') }}</p>
    </div>
    <div v-else class="picker-form">
      <div v-if="!dialogData?.skipServerSelection" class="picker-field">
        <label>{{ t('workspaces.selectServer') }}</label>
        <n-select
          :value="selectedServerId"
          :options="serverOptions"
          :placeholder="t('workspaces.selectServer')"
          @update:value="onServerChange"
        />
      </div>
      <div class="picker-field">
        <label>{{ t('workspaces.selectDatabase') }}</label>
        <n-select
          :value="selectedDatabase"
          :options="databaseOptions"
          :placeholder="t('workspaces.selectDatabase')"
          :loading="loadingDatabases"
          :disabled="!selectedServerId"
          @update:value="(v: string) => selectedDatabase = v"
        />
      </div>
    </div>
  </n-modal>
</template>

<style scoped>
.picker-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 8px 0;
}

.picker-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.picker-field label {
  font-size: 13px;
  font-weight: 500;
}

.no-connections {
  padding: 16px 0;
  text-align: center;
  color: var(--n-text-color-3);
}
</style>
```

Note: `GetDatabases` is on `DatabasesProxy` (not `ConnectionsProxy`). It returns `Result[[]string]` — the data is already a string array of database names. The `openQuery` return value modification was done in Task 6.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/features/workspaces/ServerPickerDialog.vue
git commit -m "feat(workspaces): add server and database picker dialog"
```

---

### Task 13: WorkspacePane Component

**Files:**
- Create: `frontend/src/features/workspaces/WorkspacePane.vue`

- [ ] **Step 1: Create the main workspace pane**

Create `frontend/src/features/workspaces/WorkspacePane.vue`:

```vue
<script setup lang="ts">
import { onMounted } from 'vue'
import { useWorkspaceStore } from './workspaceStore'
import { useI18n } from 'vue-i18n'
import WorkspaceToolbar from './WorkspaceToolbar.vue'
import WorkspaceTree from './WorkspaceTree.vue'
import WorkspaceEmptyState from './WorkspaceEmptyState.vue'

const { t } = useI18n()
const workspaceStore = useWorkspaceStore()

onMounted(() => {
  workspaceStore.loadWorkspaces()
})
</script>

<template>
  <div class="workspace-pane">
    <workspace-empty-state
      v-if="!workspaceStore.hasWorkspaces"
      type="no-workspaces"
      @create-workspace="workspaceStore.createWorkspace(t('workspaces.defaultWorkspaceName'))"
    />
    <template v-else>
      <workspace-toolbar />
      <workspace-empty-state
        v-if="workspaceStore.activeWorkspace && workspaceStore.activeWorkspace.folders.length === 0"
        type="no-folders"
        @add-folder="workspaceStore.addFolder()"
      />
      <workspace-tree v-else />
    </template>
  </div>
</template>

<style scoped>
.workspace-pane {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/features/workspaces/WorkspacePane.vue
git commit -m "feat(workspaces): add main workspace pane component"
```

---

### Task 14: Integrate into Ribbon + AppContent

**Files:**
- Modify: `frontend/src/features/sidebar/LeftRibbon.vue`
- Modify: `frontend/src/app/AppContent.vue`

- [ ] **Step 1: Add Workspaces icon to LeftRibbon**

In `frontend/src/features/sidebar/LeftRibbon.vue`:

1. Import the folder icon:
```typescript
import { FolderIcon } from '@heroicons/vue/24/outline'
```

2. Add the Workspaces entry to the `menuOptions` computed array (after the existing `browser` and `servers` entries):
```typescript
{
  label: t('ribbon.workspaces'),
  key: 'workspaces',
  icon: FolderIcon,
  show: true,  // Always visible
}
```

Look at how the existing entries are structured and follow the same pattern.

- [ ] **Step 2: Add WorkspacePane to AppContent**

In `frontend/src/app/AppContent.vue`:

1. Import the component:
```typescript
import WorkspacePane from '@/features/workspaces/WorkspacePane.vue'
import ServerPickerDialog from '@/features/workspaces/ServerPickerDialog.vue'
```

2. Add a new `v-show` block following the pattern of the existing Browser and Servers blocks:
```vue
<!-- Workspaces pane -->
<div v-show="tabStore.nav === NavType.Workspaces" class="content-area flex-box-h">
  <resizeable-wrapper v-model:size="settingsStore.window.asideWidth">
    <workspace-pane />
  </resizeable-wrapper>
  <unified-content-pane />
</div>
```

3. Add the `ServerPickerDialog` alongside other global dialogs in the template:
```vue
<server-picker-dialog />
```

4. Import `NavType` if not already imported (it should be available from the tab store).

- [ ] **Step 3: Verify frontend compiles**

Run from `frontend/`: `bun run lint`
Expected: no type errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/features/sidebar/LeftRibbon.vue frontend/src/app/AppContent.vue
git commit -m "feat(workspaces): integrate workspace pane into ribbon and app layout"
```

---

### Task 15: Settings — Workspaces File Extensions

**Files:**
- Create: `frontend/src/features/settings/WorkspacesSettings.vue`
- Modify: `frontend/src/features/settings/SettingsDialog.vue`

- [ ] **Step 1: Create the WorkspacesSettings component**

Create `frontend/src/features/settings/WorkspacesSettings.vue`:

```vue
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from './settingsStore'

const { t } = useI18n()
const settingsStore = useSettingsStore()

const defaultExtensions = ['.js', '.mongodb']

function resetDefaults() {
  settingsStore.workspaces.fileExtensions = [...defaultExtensions]
}
</script>

<template>
  <div class="settings-section">
    <div class="setting-item">
      <label>{{ t('settings.workspaces.fileExtensions') }}</label>
      <p class="tip">{{ t('settings.workspaces.fileExtensionsTip') }}</p>
      <n-dynamic-tags v-model:value="settingsStore.workspaces.fileExtensions" />
      <n-button text type="primary" size="small" @click="resetDefaults">
        {{ t('settings.workspaces.resetDefaults') }}
      </n-button>
    </div>
  </div>
</template>

<style scoped>
.settings-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.setting-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.setting-item label {
  font-weight: 500;
}

.tip {
  font-size: 12px;
  color: var(--n-text-color-3);
  margin: 0;
}
</style>
```

- [ ] **Step 2: Add Workspaces tab to SettingsDialog**

In `frontend/src/features/settings/SettingsDialog.vue`, add a new tab pane:

```vue
<n-tab-pane name="workspaces" :tab="t('settings.workspaces.name')">
  <workspaces-settings />
</n-tab-pane>
```

Import the component:
```typescript
import WorkspacesSettings from './WorkspacesSettings.vue'
```

- [ ] **Step 3: Update settingsStore for workspaces field**

Check `frontend/src/features/settings/settingsStore.ts` — ensure the store maps the `workspaces` field from the backend `Settings` struct. If the settings store destructures settings into individual fields (general, editor, terminal), add a `workspaces` field following the same pattern:

```typescript
workspaces: {
  fileExtensions: [] as string[],
},
```

And map it in `loadSettings()` and `saveConfiguration()` alongside the other settings fields.

- [ ] **Step 4: Verify frontend compiles**

Run from `frontend/`: `bun run lint`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/features/settings/WorkspacesSettings.vue frontend/src/features/settings/SettingsDialog.vue frontend/src/features/settings/settingsStore.ts
git commit -m "feat(workspaces): add file extension settings in settings dialog"
```

---

### Task 16: Generate Wails Bindings + Integration Test

**Files:**
- No new files — this is a build and verify step

- [ ] **Step 1: Generate Wails bindings**

Run: `wails dev` (or `wails generate module`) to generate the TypeScript bindings for `WorkspacesProxy` into `frontend/wailsjs/go/api/WorkspacesProxy.js`.

Verify the generated file exists and exports the expected functions:
`GetWorkspaces`, `CreateWorkspace`, `RenameWorkspace`, `DeleteWorkspace`, `SetActiveWorkspace`, `AddFolder`, `RemoveFolder`, `ReadDirectory`, `CreateFile`, `RenameFile`, `DeleteFile`.

- [ ] **Step 2: Update imports in frontend files**

Replace any placeholder type imports in `workspaceStore.ts` with the generated model types from `wailsjs/go/models`. Verify all Wails proxy imports in workspace components resolve correctly.

- [ ] **Step 3: Run all Go tests**

Run: `go test ./... -v`
Expected: all tests pass, including new workspace tests.

- [ ] **Step 4: Run frontend lint**

Run from `frontend/`: `bun run lint`
Expected: no errors.

- [ ] **Step 5: Manual smoke test**

With `wails dev` running:
1. Click the Workspaces icon in the ribbon — empty state appears
2. Create a workspace — dropdown shows it
3. Add a folder containing `.js` files — tree renders
4. Right-click a file → Open — server/database picker appears
5. Select server + database → query tab opens with file content
6. Right-click → New File — creates file, tree refreshes
7. Right-click → Rename — renames file
8. Right-click → Delete — deletes with confirmation
9. Settings → Workspaces tab — add/remove extensions, tree filters accordingly
10. Close and reopen app — last active workspace is remembered

- [ ] **Step 6: Commit any fixups**

```bash
git add -A
git commit -m "feat(workspaces): integration fixes and binding updates"
```
