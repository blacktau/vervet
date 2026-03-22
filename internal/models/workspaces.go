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
