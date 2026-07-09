package schema

import (
	"context"
	"testing"
	"time"

	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestSample_NilClientReturnsError(t *testing.T) {
	_, err := Sample(context.Background(), nil, "db", "coll", 100)
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func indexByPath(fs []models.FieldInfo) map[string]models.FieldInfo {
	out := make(map[string]models.FieldInfo, len(fs))
	for _, f := range fs {
		out[f.Path] = f
	}
	return out
}

func TestAccretion_SingleDoc(t *testing.T) {
	acc := newAccumulator()
	acc.add(bson.M{"name": "alice", "age": int32(30)})
	schema := acc.build(1)

	if schema.SampledCount != 1 {
		t.Fatalf("SampledCount = %d, want 1", schema.SampledCount)
	}
	if len(schema.Fields) != 2 {
		t.Fatalf("len Fields = %d, want 2", len(schema.Fields))
	}

	byPath := indexByPath(schema.Fields)
	if byPath["name"].Count != 1 {
		t.Errorf("name.Count = %d, want 1", byPath["name"].Count)
	}
	if len(byPath["name"].Types) != 1 || byPath["name"].Types[0].Type != "string" {
		t.Errorf("name.Types = %+v, want [string]", byPath["name"].Types)
	}
	if byPath["age"].Types[0].Type != "int" {
		t.Errorf("age type = %s, want int", byPath["age"].Types[0].Type)
	}
}

func TestAccretion_NestedObject(t *testing.T) {
	acc := newAccumulator()
	acc.add(bson.M{"address": bson.M{"city": "London", "zip": "EC1"}})
	schema := acc.build(1)

	byPath := indexByPath(schema.Fields)
	addr := byPath["address"]
	if len(addr.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(addr.Children))
	}
	childByPath := indexByPath(addr.Children)
	if childByPath["address.city"].Types[0].Type != "string" {
		t.Errorf("address.city type wrong")
	}
}

func TestAccretion_ArrayElementTypes(t *testing.T) {
	acc := newAccumulator()
	acc.add(bson.M{"tags": bson.A{"a", "b", int32(1)}})
	schema := acc.build(1)

	byPath := indexByPath(schema.Fields)
	tags := byPath["tags"]
	if tags.Types[0].Type != "array" {
		t.Errorf("tags top type = %s, want array", tags.Types[0].Type)
	}
	if tags.Types[0].MinLen == nil || *tags.Types[0].MinLen != 3 {
		t.Errorf("array MinLen wrong: %+v", tags.Types[0].MinLen)
	}
	childByPath := indexByPath(tags.Children)
	elem := childByPath["tags[]"]
	if len(elem.Types) != 2 {
		t.Errorf("array element types = %d, want 2", len(elem.Types))
	}
}

func TestAccretion_MixedTypes(t *testing.T) {
	acc := newAccumulator()
	acc.add(bson.M{"v": int32(1)})
	acc.add(bson.M{"v": "two"})
	schema := acc.build(2)

	byPath := indexByPath(schema.Fields)
	v := byPath["v"]
	if len(v.Types) != 2 {
		t.Fatalf("types = %d, want 2", len(v.Types))
	}
	if v.Count != 2 {
		t.Errorf("count = %d, want 2", v.Count)
	}
}

func TestAccretion_MissingFieldNotCounted(t *testing.T) {
	acc := newAccumulator()
	acc.add(bson.M{"a": 1})
	acc.add(bson.M{"b": 2})
	schema := acc.build(2)

	byPath := indexByPath(schema.Fields)
	if byPath["a"].Count != 1 {
		t.Errorf("a.Count = %d, want 1", byPath["a"].Count)
	}
	if byPath["b"].Count != 1 {
		t.Errorf("b.Count = %d, want 1", byPath["b"].Count)
	}
}

func TestAccretion_NullDistinctFromMissing(t *testing.T) {
	acc := newAccumulator()
	acc.add(bson.M{"a": nil})
	acc.add(bson.M{})
	schema := acc.build(2)

	byPath := indexByPath(schema.Fields)
	if byPath["a"].Count != 1 {
		t.Errorf("null-present count = %d, want 1", byPath["a"].Count)
	}
	if byPath["a"].Types[0].Type != "null" {
		t.Errorf("null type = %s", byPath["a"].Types[0].Type)
	}
}

func TestAccretion_DateMinMax(t *testing.T) {
	acc := newAccumulator()
	t1 := bson.NewDateTimeFromTime(time.Unix(1000, 0))
	t2 := bson.NewDateTimeFromTime(time.Unix(5000, 0))
	acc.add(bson.M{"ts": t1})
	acc.add(bson.M{"ts": t2})
	schema := acc.build(2)

	byPath := indexByPath(schema.Fields)
	ts := byPath["ts"].Types[0]
	if ts.Min == nil || ts.Max == nil {
		t.Fatalf("min/max nil")
	}
	if *ts.Min == *ts.Max {
		t.Errorf("min == max")
	}
}

func TestAccretion_StringLengthRunes(t *testing.T) {
	acc := newAccumulator()
	acc.add(bson.M{"s": "abc"})
	acc.add(bson.M{"s": "héllo"})
	schema := acc.build(2)

	byPath := indexByPath(schema.Fields)
	s := byPath["s"].Types[0]
	if s.MinLen == nil || *s.MinLen != 3 {
		t.Errorf("MinLen = %v, want 3", s.MinLen)
	}
	if s.MaxLen == nil || *s.MaxLen != 5 {
		t.Errorf("MaxLen = %v, want 5", s.MaxLen)
	}
}
