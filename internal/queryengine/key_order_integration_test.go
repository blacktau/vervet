//go:build integration

package queryengine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Key order is significant to MongoDB in more places than command names. These
// tests guard against script objects losing their key order on the way to the
// driver (issue #301). Each repeats the operation because a randomly ordered
// map can produce the right order by chance.

func keyOrderEngine(t *testing.T) (context.Context, string, *GojaEngine) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	db := dbName(t)
	t.Cleanup(func() { _ = testClient.Database(db).Drop(context.Background()) })
	return ctx, db, NewGojaEngine(testClient, 0, "")
}

func TestIntegration_KeyOrder_CompoundIndexKeys(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	for i := 0; i < 10; i++ {
		_, err := engine.ExecuteQuery(ctx, testURI, db, `db.idx.createIndex({ c: 1, a: -1, b: 1 })`)
		require.NoError(t, err)
		_, err = engine.ExecuteQuery(ctx, testURI, db, `db.idx.createIndexes([{ z: 1, y: 1, x: 1 }])`)
		require.NoError(t, err)
	}

	specs, err := testClient.Database(db).Collection("idx").Indexes().ListSpecifications(ctx)
	require.NoError(t, err)
	names := make([]string, 0, len(specs))
	for _, s := range specs {
		names = append(names, s.Name)
	}
	assert.ElementsMatch(t, []string{"_id_", "c_1_a_-1_b_1", "z_1_y_1_x_1"}, names)
}

func TestIntegration_KeyOrder_CompoundSort(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.s.insertMany([
		{ n: 1, g: "b", v: 1 }, { n: 2, g: "a", v: 2 },
		{ n: 3, g: "b", v: 3 }, { n: 4, g: "a", v: 4 }
	])`)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		result, err := engine.ExecuteQuery(ctx, testURI, db,
			`db.s.find({}, { _id: 0, n: 1 }).sort({ g: 1, v: -1 }).toArray().map(function (d) { return d.n; }).join(",")`)
		require.NoError(t, err)
		require.Equal(t, "4,2,3,1", resultText(result))

		result, err = engine.ExecuteQuery(ctx, testURI, db,
			`db.s.aggregate([{ $sort: { g: 1, v: -1 } }]).toArray().map(function (d) { return d.n; }).join(",")`)
		require.NoError(t, err)
		require.Equal(t, "4,2,3,1", resultText(result))
	}
}

func TestIntegration_KeyOrder_InsertedFieldOrder(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db,
		`db.f.insertOne({ _id: 1, zebra: 1, apple: 2, mango: { y: 1, x: 2, w: 3 } })`)
	require.NoError(t, err)

	var raw bson.Raw
	require.NoError(t, testClient.Database(db).Collection("f").FindOne(ctx, bson.D{}).Decode(&raw))
	var doc bson.D
	require.NoError(t, bson.Unmarshal(raw, &doc))
	assert.Equal(t, []string{"_id", "zebra", "apple", "mango"}, docKeys(doc))
	assert.Equal(t, []string{"y", "x", "w"}, docKeys(doc[3].Value.(bson.D)))

	// An exact embedded-document match only succeeds with the same field order.
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.f.countDocuments({ mango: { y: 1, x: 2, w: 3 } })`)
	require.NoError(t, err)
	assert.Contains(t, resultText(result), "1")
}
