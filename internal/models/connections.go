package models

type Connection struct {
	ServerID string `json:"serverID,omitempty"`
	Name     string `json:"name,omitempty"`
}