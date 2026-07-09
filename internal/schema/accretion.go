package schema

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func (a *accumulator) add(doc bson.M) {
	walkObject(a.root, doc)
}

func walkObject(nodes map[string]*fieldNode, doc bson.M) {
	for k, v := range doc {
		n := getOrCreateNode(nodes, k)
		n.count++
		applyValue(n, v)
	}
}

func getOrCreateNode(nodes map[string]*fieldNode, name string) *fieldNode {
	if n, ok := nodes[name]; ok {
		return n
	}
	n := &fieldNode{
		name:     name,
		types:    make(map[string]*typeStat),
		children: make(map[string]*fieldNode),
	}
	nodes[name] = n
	return n
}

func applyValue(n *fieldNode, v any) {
	typeName := bsonTypeName(v)
	ts := getOrCreateType(n, typeName)
	ts.count++

	switch val := v.(type) {
	case bson.M:
		walkObject(n.children, val)
	case bson.A:
		recordLen(ts, len(val))
		elem := getOrCreateNode(n.children, "[]")
		for _, item := range val {
			elem.count++
			applyValue(elem, item)
		}
	case []any:
		recordLen(ts, len(val))
		elem := getOrCreateNode(n.children, "[]")
		for _, item := range val {
			elem.count++
			applyValue(elem, item)
		}
	case string:
		recordLen(ts, len([]rune(val)))
		recordStr(ts, val)
	case int32:
		recordNum(ts, float64(val))
	case int64:
		recordNum(ts, float64(val))
	case int:
		recordNum(ts, float64(val))
	case float64:
		recordNum(ts, val)
	case bson.DateTime:
		recordNum(ts, float64(val))
	case time.Time:
		recordNum(ts, float64(val.UnixMilli()))
	}
}

func getOrCreateType(n *fieldNode, name string) *typeStat {
	if t, ok := n.types[name]; ok {
		return t
	}
	t := &typeStat{name: name}
	n.types[name] = t
	return t
}

func recordNum(t *typeStat, v float64) {
	if t.minNum == nil || v < *t.minNum {
		cp := v
		t.minNum = &cp
	}
	if t.maxNum == nil || v > *t.maxNum {
		cp := v
		t.maxNum = &cp
	}
}

func recordStr(t *typeStat, v string) {
	if t.minStr == nil || v < *t.minStr {
		cp := v
		t.minStr = &cp
	}
	if t.maxStr == nil || v > *t.maxStr {
		cp := v
		t.maxStr = &cp
	}
}

func recordLen(t *typeStat, n int) {
	if t.minLen == nil || n < *t.minLen {
		cp := n
		t.minLen = &cp
	}
	if t.maxLen == nil || n > *t.maxLen {
		cp := n
		t.maxLen = &cp
	}
}

func bsonTypeName(v any) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case int32:
		return "int"
	case int64, int:
		return "long"
	case float64:
		return "double"
	case bson.Decimal128:
		return "decimal"
	case bson.ObjectID:
		return "objectId"
	case bson.DateTime, time.Time:
		return "date"
	case bson.Binary:
		return "binary"
	case bson.Regex:
		return "regex"
	case bson.M:
		return "object"
	case bson.A, []any:
		return "array"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func (a *accumulator) build(sampledCount int) models.CollectionSchema {
	return models.CollectionSchema{
		SampledCount: sampledCount,
		Fields:       buildFields("", a.root),
	}
}

func buildFields(parentPath string, nodes map[string]*fieldNode) []models.FieldInfo {
	out := make([]models.FieldInfo, 0, len(nodes))
	for _, n := range nodes {
		path := n.name
		if parentPath != "" {
			if n.name == "[]" {
				path = parentPath + "[]"
			} else {
				path = parentPath + "." + n.name
			}
		}
		fi := models.FieldInfo{
			Path:     path,
			Name:     n.name,
			Count:    n.count,
			Types:    buildTypes(n.types),
			Children: buildFields(path, n.children),
		}
		out = append(out, fi)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func buildTypes(types map[string]*typeStat) []models.TypeStat {
	out := make([]models.TypeStat, 0, len(types))
	for _, t := range types {
		ts := models.TypeStat{Type: t.name, Count: t.count}
		if t.minNum != nil {
			s := strconv.FormatFloat(*t.minNum, 'f', -1, 64)
			ts.Min = &s
		}
		if t.maxNum != nil {
			s := strconv.FormatFloat(*t.maxNum, 'f', -1, 64)
			ts.Max = &s
		}
		if t.minLen != nil {
			cp := *t.minLen
			ts.MinLen = &cp
		}
		if t.maxLen != nil {
			cp := *t.maxLen
			ts.MaxLen = &cp
		}
		out = append(out, ts)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}
