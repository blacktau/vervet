//go:build integration

package queryengine

import (
	"context"
	"encoding/json"
	"testing"

	"vervet/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Edits and deletes from the results grid go to the namespace a result reports,
// so a result that can't vouch for its documents must not report one.
func TestResultNamespace(t *testing.T) {
	ctx := context.Background()
	db := dbName(t)
	sibling := db + "_sibling"
	_, err := testClient.Database(db).Collection("customers").InsertOne(ctx, bson.D{
		{Key: "_id", Value: 1}, {Key: "score", Value: 5.0},
	})
	require.NoError(t, err)
	_, err = testClient.Database(sibling).Collection("orders").InsertOne(ctx, bson.D{{Key: "_id", Value: 1}})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = testClient.Database(db).Drop(ctx)
		_ = testClient.Database(sibling).Drop(ctx)
	})

	whole := &models.ResultNamespace{Database: db, Collection: "customers", WholeDocuments: true}
	partial := &models.ResultNamespace{Database: db, Collection: "customers"}
	inSibling := &models.ResultNamespace{Database: sibling, Collection: "orders", WholeDocuments: true}

	cases := []struct {
		name   string
		script string
		want   *models.ResultNamespace
	}{
		{"paged find", `db.customers.find()`, whole},
		{"find with a projection", `db.customers.find({}, { score: 1 })`, partial},
		{"find with a cursor projection", `db.customers.find().projection({ score: 1 })`, partial},
		{"find with returnKey", `db.customers.find().returnKey(true)`, partial},
		{"find on a sibling database", `db.getSiblingDB("` + sibling + `").orders.find()`, inSibling},
		{"partly read cursor", `var c = db.customers.find(); c.hasNext(); c`, whole},
		{"toArray", `db.customers.find().toArray()`, whole},
		{"toArray with a projection", `db.customers.find({}, { score: 1 }).toArray()`, partial},
		{"findOne", `db.customers.findOne()`, whole},
		{"findOne with a projection", `db.customers.findOne({}, { score: 1 })`, partial},
		{"findOne on a sibling database", `db.getSiblingDB("` + sibling + `").orders.findOne()`, inSibling},
		{"aggregate with whole documents", `db.customers.aggregate([{ $match: {} }, { $sort: { _id: 1 } }, { $skip: 0 }, { $limit: 5 }])`, whole},
		{"aggregate given stages as arguments", `db.customers.aggregate({ $match: {} }, { $limit: 5 })`, whole},
		{"aggregate toArray", `db.customers.aggregate([]).toArray()`, whole},
		{"aggregate on a sibling database", `db.getSiblingDB("` + sibling + `").orders.aggregate([])`, inSibling},
		{"aggregate with $group", `db.customers.aggregate([{ $group: { _id: "$_id" } }])`, nil},
		{"aggregate with $project", `db.customers.aggregate([{ $project: { score: 1 } }])`, nil},
		{"aggregate with $addFields", `db.customers.aggregate([{ $addFields: { extra: 1 } }])`, nil},
		{"aggregate explain", `db.customers.aggregate([], { explain: true })`, nil},
		{"database aggregate", `db.aggregate([{ $documents: [{ _id: 1 }] }])`, nil},
		{"array built in a script", `[{ _id: 1 }]`, nil},
		{"object built in a script", `({ _id: 1 })`, nil},
		{"changed findOne document", `var d = db.customers.findOne(); d.score = 6; d`, nil},
		{"changed aggregate document", `var a = db.customers.aggregate([]).toArray(); a[0].score = 6; a`, nil},
		{"mapped cursor", `db.customers.find().map(d => d)`, nil},
		{"document taken from an aggregate", `db.customers.aggregate([]).toArray()[0]`, nil},
		{"write result", `db.customers.updateOne({ _id: 999 }, { $set: { a: 1 } })`, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			engine := NewGojaEngine(testClient, 25, "")
			result, err := engine.ExecuteQuery(ctx, testURI, db, c.script)
			require.NoError(t, err)
			assert.Equal(t, c.want, result.Namespace)
		})
	}
}

// Saving replaces the document with what the grid showed, so a whole-number
// double read through aggregate must still be shown as a double.
func TestAggregateFinalValueKeepsStoredTypes(t *testing.T) {
	ctx := context.Background()
	db := dbName(t)
	_, err := testClient.Database(db).Collection("c").InsertOne(ctx, bson.D{
		{Key: "_id", Value: 1}, {Key: "score", Value: 5.0},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = testClient.Database(db).Drop(ctx) })

	engine := NewGojaEngine(testClient, 25, "")
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.c.aggregate([{ $match: {} }])`)
	require.NoError(t, err)
	data, err := json.Marshal(result.Documents)
	require.NoError(t, err)
	assert.Contains(t, string(data), `{"$numberDouble":"5.0"}`)
}
