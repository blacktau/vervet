package queryengine

import (
	"context"
	"fmt"
	"slices"
	"time"
	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SampleSchema samples up to 100 documents from a collection and returns
// a merged field schema describing the field names, types, and nested structure.
func SampleSchema(ctx context.Context, client *mongo.Client, dbName, collName string) (models.CollectionSchema, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	coll := client.Database(dbName).Collection(collName)
	cursor, err := coll.Find(ctx, bson.D{}, options.Find().SetLimit(100))
	if err != nil {
		return models.CollectionSchema{}, fmt.Errorf("failed to sample documents: %w", err)
	}
	defer cursor.Close(ctx)

	root := make(map[string]*fieldNode)

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		mergeDocument(root, doc)
	}

	if err := cursor.Err(); err != nil {
		return models.CollectionSchema{}, fmt.Errorf("cursor error during schema sampling: %w", err)
	}

	fields := buildFieldInfos(root)
	return models.CollectionSchema{Fields: fields}, nil
}

// fieldNode tracks the observed types and children for a single field path.
type fieldNode struct {
	name     string
	types    map[string]bool
	children map[string]*fieldNode
}

func getOrCreateNode(nodes map[string]*fieldNode, name string) *fieldNode {
	if n, ok := nodes[name]; ok {
		return n
	}
	n := &fieldNode{
		name:     name,
		types:    make(map[string]bool),
		children: make(map[string]*fieldNode),
	}
	nodes[name] = n
	return n
}

// mergeDocument walks a single document and records field names and types.
func mergeDocument(nodes map[string]*fieldNode, doc bson.M) {
	for key, val := range doc {
		node := getOrCreateNode(nodes, key)
		typeName := bsonTypeName(val)
		node.types[typeName] = true

		if subDoc, ok := val.(bson.M); ok {
			mergeDocument(node.children, subDoc)
		}
	}
}

// bsonTypeName maps a Go value (as decoded by the mongo driver) to a schema type string.
func bsonTypeName(v any) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case string:
		return "string"
	case int32, int64, int:
		return "number"
	case float64:
		return "double"
	case bool:
		return "boolean"
	case bson.M:
		return "object"
	case bson.A, []any:
		return "array"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// buildFieldInfos converts the internal fieldNode tree into a sorted slice of FieldInfo.
func buildFieldInfos(nodes map[string]*fieldNode) []models.FieldInfo {
	fields := make([]models.FieldInfo, 0, len(nodes))
	for _, node := range nodes {
		types := make([]string, 0, len(node.types))
		for t := range node.types {
			types = append(types, t)
		}
		slices.Sort(types)

		fi := models.FieldInfo{
			Path:  node.name,
			Types: types,
		}
		if len(node.children) > 0 {
			fi.Children = buildFieldInfos(node.children)
		}
		fields = append(fields, fi)
	}
	slices.SortFunc(fields, func(a, b models.FieldInfo) int {
		if a.Path < b.Path {
			return -1
		}
		if a.Path > b.Path {
			return 1
		}
		return 0
	})
	return fields
}
