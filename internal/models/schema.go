package models

type CollectionSchema struct {
	Fields []FieldInfo `json:"fields"`
}

type FieldInfo struct {
	Path     string      `json:"path"`
	Types    []string    `json:"types"`
	Children []FieldInfo `json:"children"`
}
