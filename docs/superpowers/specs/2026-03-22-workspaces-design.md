# Workspaces Feature Design

## Overview

Add a workspace-based local file manager to Vervet. Workspaces let users organise local filesystem folders containing MongoDB query files and open those files as query tabs against connected servers.

## UI

### Left Ribbon

A new **Workspaces** icon (FolderIcon) is added as the third item in the left ribbon, below Servers and Browser. It is always visible regardless of connection state. Clicking it switches the sidebar to the Workspace pane.

The `NavType` enum gains a new value: `Workspaces`.

When the Workspaces nav is active, the main content area continues to show `UnifiedContentPane` (the server/query tabs). This means the user can browse their workspace file tree on the left while seeing query results on the right — opening a file adds a query tab without switching away from the workspace pane.

### Workspace Pane Layout

The pane has three areas stacked vertically:

1. **Workspace selector toolbar** — a dropdown listing all workspaces by name, a **+** button to create a new workspace, and a **gear** button that opens a popover with Rename and Delete options.
2. **Folder actions bar** — an "Add Folder" button that opens the OS native folder picker dialog.
3. **File tree** — a Naive UI `n-tree` showing the active workspace's folders and their contents.

### File Tree Behaviour

- Tree loads lazily: root folders load on workspace activation, subfolders load on expand.
- `ReadDirectory(path)` returns entries filtered to directories and files matching extensions: `.js`, `.mongodb`.
- Entries are sorted: folders first, then files, both alphabetical.
- No file watching. A **Refresh** option is available via right-click context menu on folders, plus a refresh button in the toolbar.
- Root folders display with a **✕** button to remove them from the workspace (does not delete from disk).

### Context Menu

| Target | Actions |
|---|---|
| **File** | Open, Open on Server..., Rename, Delete |
| **Subfolder** | New File, Refresh, Rename, Delete |
| **Root folder** | New File, Refresh, Remove from Workspace |

- **Open** — opens the file as a query tab on the currently active server tab. If no server tab is active, shows the server picker dialog.
- **Open on Server...** — always shows the server picker dialog regardless of whether a server tab is active.
- **Rename** — shows a small input dialog pre-filled with the current name.
- **New File** — shows a small input dialog for the filename. Default extension is `.js`. Creates the file via `CreateFile(dirPath, name)` and refreshes the parent folder in the tree.
- **Delete** on files and subfolders deletes from disk with a confirmation dialog.
- **Remove from Workspace** detaches the folder from the workspace without deleting anything from disk.

### Server Picker Dialog

A reusable modal dialog listing all currently connected servers. The user selects one to open the file against. If no servers are connected, the dialog shows a message telling the user to connect to a server first.

This dialog is used in two cases:
1. When the user chooses "Open on Server..." from the context menu.
2. When the user chooses "Open" but there is no currently active server tab.

After selecting a server, a **database picker** is shown listing the databases on that server. The user must choose which database to run the query against.

### Opening Files as Query Tabs

When a file is opened (via double-click, "Open", or "Open on Server..."):

1. **Determine the target server** — use the active server tab, or show the server picker dialog.
2. **Pick a database** — show a database picker (dropdown or small dialog) listing the databases on the selected server. The user must choose which database to run the query against, since workspace files have no inherent database association.
3. **Open the query tab** — call `openQuery(serverId, database)` on the tab store to create the inner query tab.
4. **Load file content** — read the file via `filesProxy.ReadFile(path)` and set the content on the query editor. Set `filePath` and `savedContent` on the query state so the tab label shows the filename and dirty-tracking works correctly. This requires new logic in the query store (e.g. `loadFileByPath(queryId, filePath)`) since the existing `openFile` method opens an OS file picker dialog rather than accepting a known path.

The query tab label displays the filename (existing `QueryTabItem.filePath` labelling logic handles this).

### Workspace Management

- **Create**: Click **+** → creates a new workspace named "New Workspace" and switches to it. The name is immediately editable.
- **Switch**: Select from the dropdown → calls `SetActiveWorkspace`, reloads the file tree.
- **Rename**: Gear menu → Rename → inline edit.
- **Delete**: Gear menu → Delete → confirmation dialog → removes workspace definition (not folders on disk).

### Empty States

- **No workspaces exist**: Centred prompt — "Create a workspace to organise your query files" with a "Create Workspace" button.
- **Workspace has no folders**: "Add a folder to get started" with an "Add Folder" button.
- **Folder contains no query files**: Tree node shows greyed-out "(no query files)".

## Data Model

### Persistence

A new file `~/.config/vervet/workspaces.yaml` stores all workspace definitions:

```yaml
activeWorkspaceId: "abc-123"
workspaces:
  - id: "abc-123"
    name: "My Project Queries"
    folders:
      - "/home/sean/projects/prod-queries"
      - "/home/sean/projects/shared-scripts"
  - id: "def-456"
    name: "Client Work"
    folders:
      - "/home/sean/clients/acme/queries"
```

Each workspace has:
- `id` — generated UUID
- `name` — user-chosen display name
- `folders` — ordered list of absolute filesystem paths

The `activeWorkspaceId` tracks which workspace was last open (persisted across restarts). Tree expansion state and open files are **not** persisted.

### Go Types

```go
// internal/models/workspaces.go

type Workspace struct {
    ID      string   `yaml:"id"`
    Name    string   `yaml:"name"`
    Folders []string `yaml:"folders"`
}

type WorkspaceData struct {
    ActiveWorkspaceID string      `yaml:"activeWorkspaceId"`
    Workspaces        []Workspace `yaml:"workspaces"`
}

type DirectoryEntry struct {
    Name        string           `json:"name"`
    Path        string           `json:"path"`
    IsDirectory bool             `json:"isDirectory"`
    Children    []DirectoryEntry `json:"children,omitempty"`
}
```

## Backend Architecture

### Package: `internal/workspaces/`

A `WorkspaceService` that uses the existing `infrastructure.Store` for YAML persistence to `workspaces.yaml`.

Responsibilities:
- CRUD operations on workspace definitions
- Active workspace tracking
- Folder management within workspaces
- Directory reading with file-type filtering
- File operations (create, rename, delete)

### API: `internal/api/workspaces.go`

A new `WorkspacesProxy` following the existing `Result[T]` / `EmptyResult` pattern:

| Method | Returns | Description |
|---|---|---|
| `GetWorkspaces()` | `Result[WorkspaceData]` | All workspace definitions + active ID |
| `CreateWorkspace(name string)` | `Result[Workspace]` | Create with generated UUID |
| `RenameWorkspace(id, name string)` | `EmptyResult` | Update workspace name |
| `DeleteWorkspace(id string)` | `EmptyResult` | Remove workspace definition |
| `SetActiveWorkspace(id string)` | `EmptyResult` | Set active workspace ID |
| `AddFolder(workspaceId string)` | `Result[string]` | Opens OS folder picker (via Wails runtime dialog), adds path, returns it |
| `RemoveFolder(workspaceId, path string)` | `EmptyResult` | Detach folder from workspace |
| `ReadDirectory(path string)` | `Result[[]DirectoryEntry]` | Filtered directory listing |
| `CreateFile(dirPath, name string)` | `Result[string]` | Create empty query file, return path |
| `RenameFile(oldPath, newPath string)` | `EmptyResult` | Rename file or folder on disk |
| `DeleteFile(path string)` | `EmptyResult` | Delete file or folder from disk |

File reading and writing for query content reuses the existing `FilesProxy`.

### Wiring

The `WorkspacesProxy` is instantiated in `internal/app/` and added to the Wails `Bind` list, following the pattern of existing proxies.

## Frontend Architecture

### Store: `features/workspaces/workspaceStore.ts`

Pinia store managing:

| State | Type | Description |
|---|---|---|
| `workspaces` | `Workspace[]` | All workspace definitions |
| `activeWorkspaceId` | `string \| null` | Currently active workspace |
| `treeData` | `TreeOption[]` | Lazily-loaded directory tree for active workspace |
| `expandedKeys` | `string[]` | Currently expanded tree nodes |
| `loading` | `boolean` | Loading state |

Key actions:
- `loadWorkspaces()` — fetch all from backend
- `createWorkspace(name)` — create and switch to new workspace
- `renameWorkspace(id, name)`
- `deleteWorkspace(id)` — delete and switch to another if active was deleted
- `setActiveWorkspace(id)` — switch and reload tree
- `addFolder()` — open folder picker, add to active workspace, refresh tree
- `removeFolder(path)` — remove from active workspace, refresh tree
- `loadDirectory(path)` — lazy load a directory's contents into tree
- `refreshTree()` — reload all root folders

### Components

| Component | Location | Description |
|---|---|---|
| `WorkspacePane.vue` | `features/workspaces/` | Main pane: toolbar + tree |
| `WorkspaceTree.vue` | `features/workspaces/` | `n-tree` with lazy loading and context menu |
| `WorkspaceToolbar.vue` | `features/workspaces/` | Workspace selector dropdown, +, gear, add folder, refresh |
| `ServerPickerDialog.vue` | `features/workspaces/` | Modal to pick a connected server, then a database on that server |
| `WorkspaceEmptyState.vue` | `features/workspaces/` | Empty state prompts |

### Integration Points

- **`LeftRibbon.vue`** — add Workspaces icon, extend `NavType` to include `'workspaces'`
- **`AppContent.vue`** — render `WorkspacePane` when `NavType` is `'workspaces'`
- **`tabs.ts`** — reuse `openQuery()` for opening files as query tabs
- **`queryStore.ts`** — add new `loadFileByPath(queryId, filePath)` action for loading file content by known path into a query tab
- **`dialogStore.ts`** — add `showServerPicker` flag if using the shared dialog pattern
- **`i18n/en-GB/`** — add workspace-related translation keys

## File Type Filtering

Query-relevant file extensions:
- `.js` — JavaScript query files
- `.mongodb` — MongoDB-specific query files

These are the only files shown in the workspace tree. Directories that contain no matching files (recursively) show a greyed-out "(no query files)" label.

## Error Handling

All backend methods return `Result[T]` or `EmptyResult`. The frontend store checks `isSuccess` and uses `useNotifier()` for error display, following existing patterns.

Specific error cases:
- **Folder no longer exists on disk**: Show notification on tree load, grey out the root folder with "(folder not found)".
- **File deleted externally**: Show notification when attempting to open, remove from tree on next refresh.
- **Permission denied**: Surface OS error message via notification.

## i18n

All UI strings go through the `i18n/en-GB/` translation files. New keys needed for:
- Ribbon tooltip
- Toolbar labels and buttons
- Context menu items
- Empty state messages
- Server picker dialog
- Confirmation dialogs
- Error messages
