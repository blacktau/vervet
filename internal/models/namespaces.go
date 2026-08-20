package models

// NamespaceInventory is the full set of navigable namespaces on a server,
// used to back the data browser's global find.
type NamespaceInventory struct {
	ServerID  string               `json:"serverID"`
	Databases []DatabaseNamespaces `json:"databases"`
}

type DatabaseNamespaces struct {
	Name        string   `json:"name"`
	Collections []string `json:"collections"`
	Views       []string `json:"views"`
}
