package models

type IndexKeyField struct {
	Field     string      `json:"field"`
	Direction interface{} `json:"direction"`
}

type Index struct {
	Name   string          `json:"name"`
	Keys   []IndexKeyField `json:"keys"`
	Unique bool            `json:"unique"`
	Sparse bool            `json:"sparse"`
	TTL    *int32          `json:"ttl,omitempty"`
}

type CreateIndexRequest struct {
	Keys   []IndexKeyField `json:"keys"`
	Name   string          `json:"name,omitempty"`
	Unique bool            `json:"unique"`
	Sparse bool            `json:"sparse"`
	TTL    *int32          `json:"ttl,omitempty"`
}
