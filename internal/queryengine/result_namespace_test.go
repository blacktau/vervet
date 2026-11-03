package queryengine

import (
	"math"
	"testing"

	"vervet/internal/models"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestKeepsDocumentsWhole(t *testing.T) {
	stage := func(name string) bson.D { return bson.D{{Key: name, Value: bson.D{}}} }
	cases := []struct {
		name     string
		pipeline any
		want     bool
	}{
		{"empty pipeline", bson.A{}, true},
		{"selecting and ordering stages", bson.A{stage("$match"), stage("$sort"), stage("$skip"), stage("$limit")}, true},
		{"$group makes new documents", bson.A{stage("$match"), stage("$group")}, false},
		{"$project hides fields", bson.A{stage("$project")}, false},
		{"$unset hides fields", bson.A{stage("$unset")}, false},
		{"$addFields adds fields", bson.A{stage("$addFields")}, false},
		{"$lookup adds fields", bson.A{stage("$lookup")}, false},
		{"$unionWith reads another collection", bson.A{stage("$unionWith")}, false},
		{"stage with two keys is not one we know", bson.A{bson.D{{Key: "$match", Value: 1}, {Key: "$group", Value: 1}}}, false},
		{"not a pipeline", "$match", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, keepsDocumentsWhole(c.pipeline))
		})
	}
}

func TestFindNamespaceWithComputedID(t *testing.T) {
	cases := []struct {
		name       string
		projection bson.D
		kept       bool
	}{
		{"_id included", bson.D{{Key: "_id", Value: int32(1)}}, true},
		{"_id excluded", bson.D{{Key: "_id", Value: int32(0)}}, true},
		{"_id as a boolean", bson.D{{Key: "_id", Value: true}}, true},
		{"_id as a double", bson.D{{Key: "_id", Value: 1.0}}, true},
		{"_id as a long", bson.D{{Key: "_id", Value: int64(1)}}, true},
		{"_id from another field", bson.D{{Key: "_id", Value: "$customerId"}}, false},
		{"_id from an expression", bson.D{{Key: "_id", Value: bson.D{{Key: "$literal", Value: 1}}}}, false},
		{"part of _id", bson.D{{Key: "_id.a", Value: int32(1)}}, false},
		{"a field that starts the same", bson.D{{Key: "_idx", Value: int32(1)}}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ns := findNamespace("shop", "orders", c.projection, findExtras{})
			assert.Equal(t, c.kept, ns != nil)
		})
	}
}

func TestFindNamespaceWholeDocuments(t *testing.T) {
	projection := bson.D{{Key: "name", Value: 1}}
	cases := []struct {
		name       string
		projection any
		extras     findExtras
		want       bool
	}{
		{"no projection", nil, findExtras{}, true},
		{"empty projection", bson.D{}, findExtras{}, true},
		{"projection", projection, findExtras{}, false},
		{"returnKey", nil, findExtras{ReturnKey: true}, false},
		{"showRecordId", nil, findExtras{ShowRecordID: true}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ns := findNamespace("shop", "orders", c.projection, c.extras)
			assert.Equal(t, "shop", ns.Database)
			assert.Equal(t, "orders", ns.Collection)
			assert.Equal(t, c.want, ns.WholeDocuments)
		})
	}
}

// Every BSON type a read can return must compare as unchanged, or the result
// silently loses its namespace and exact types. A change must never do so.
func TestReadUnchanged(t *testing.T) {
	id := bson.NewObjectID()
	doc := bson.D{
		{Key: "_id", Value: id},
		{Key: "double", Value: 5.0},
		{Key: "nan", Value: math.NaN()},
		{Key: "int", Value: int32(1)},
		{Key: "long", Value: int64(5000000000)},
		{Key: "decimal", Value: mustDecimal(t, "1.5")},
		{Key: "date", Value: bson.DateTime(1700000000000)},
		{Key: "binary", Value: bson.Binary{Subtype: 4, Data: make([]byte, 16)}},
		{Key: "timestamp", Value: bson.Timestamp{T: 1, I: 2}},
		{Key: "regex", Value: bson.Regex{Pattern: "a", Options: "i"}},
		{Key: "null", Value: nil},
		{Key: "min", Value: bson.MinKey{}},
		{Key: "max", Value: bson.MaxKey{}},
		{Key: "code", Value: bson.JavaScript("1")},
		{Key: "nested", Value: bson.D{{Key: "list", Value: bson.A{"a", int32(1), bson.D{{Key: "b", Value: id}}}}}},
		{Key: "empty", Value: bson.A{}},
	}
	setup := func(t *testing.T) (*execContext, models.QueryResult) {
		rt := goja.New()
		require.NoError(t, registerBSONTypes(rt))
		return &execContext{rt: rt}, docsToResult([]bson.D{doc})
	}

	t.Run("unchanged list", func(t *testing.T) {
		ec, result := setup(t)
		val := toGojaValue(ec.rt, result)
		assert.True(t, ec.readUnchanged(result, exportValue(val)))
	})

	t.Run("unchanged single document", func(t *testing.T) {
		ec, result := setup(t)
		result.Single = true
		val := toGojaValue(ec.rt, result)
		assert.True(t, ec.readUnchanged(result, exportValue(val)))
	})

	for name, change := range map[string]string{
		"changed number":    `v[0].int = 2`,
		"changed id":        `v[0]._id = ObjectId()`,
		"changed date":      `v[0].date = new Date(0)`,
		"changed long":      `v[0].long = NumberLong(1)`,
		"nested change":     `v[0].nested.list[2].b = 1`,
		"added field":       `v[0].extra = 1`,
		"removed field":     `delete v[0].double`,
		"reordered fields":  `var d = v[0].double; delete v[0].double; v[0].double = d`,
		"added document":    `v.push({ _id: 1 })`,
		"removed document":  `v.pop()`,
		"swapped for plain": `v[0].nested = 1`,
	} {
		t.Run(name, func(t *testing.T) {
			ec, result := setup(t)
			require.NoError(t, ec.rt.Set("v", toGojaValue(ec.rt, result)))
			_, err := ec.rt.RunString(change)
			require.NoError(t, err)
			assert.False(t, ec.readUnchanged(result, exportValue(ec.rt.Get("v"))))
		})
	}
}

func mustDecimal(t *testing.T, s string) bson.Decimal128 {
	t.Helper()
	d, err := bson.ParseDecimal128(s)
	require.NoError(t, err)
	return d
}
