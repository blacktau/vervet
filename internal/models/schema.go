package models

type CollectionSchema struct {
	SampledCount int         `json:"sampledCount"`
	TotalCount   int64       `json:"totalCount"`
	Fields       []FieldInfo `json:"fields"`
}

type FieldInfo struct {
	Path     string      `json:"path"`
	Name     string      `json:"name"`
	Count    int         `json:"count"`
	Types    []TypeStat  `json:"types"`
	Children []FieldInfo `json:"children"`
}

type TypeStat struct {
	Type   string  `json:"type"`
	Count  int     `json:"count"`
	Min    *string `json:"min,omitempty"`
	Max    *string `json:"max,omitempty"`
	MinLen *int    `json:"minLen,omitempty"`
	MaxLen *int    `json:"maxLen,omitempty"`
}
