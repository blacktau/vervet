package models

// FileFilter restricts a file dialog to matching files, e.g. "*.js;*.mongodb".
type FileFilter struct {
	DisplayName string `json:"displayName"`
	Pattern     string `json:"pattern"`
}
