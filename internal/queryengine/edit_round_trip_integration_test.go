//go:build integration

package queryengine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// The frontend's editableEjson.test.ts checks that each "stored" value is shown
// as "editor"; these tests check that saving "editor" gives back "stored".
const editRoundTripFixture = "../../frontend/src/features/results-document-tree/tests/fixtures/editRoundTrip.json"

type editRoundTripCase struct {
	Name       string          `json:"name"`
	Stored     json.RawMessage `json:"stored"`
	Editor     json.RawMessage `json:"editor"`
	UsableAsID *bool           `json:"usableAsId"`
}

func loadEditRoundTripCases(t *testing.T) []editRoundTripCase {
	t.Helper()
	data, err := os.ReadFile(editRoundTripFixture)
	require.NoError(t, err)
	var cases []editRoundTripCase
	require.NoError(t, json.Unmarshal(data, &cases))
	return cases
}

// ejsonParseScript is what ejsonParseExpression in editableEjson.ts writes.
func ejsonParseScript(t *testing.T, text string) string {
	t.Helper()
	quoted, err := json.Marshal(text)
	require.NoError(t, err)
	return fmt.Sprintf("EJSON.parse(%s, { relaxed: false })", quoted)
}

func storedRaw(t *testing.T, ctx context.Context, coll *mongo.Collection) bson.Raw {
	t.Helper()
	raw, err := coll.FindOne(ctx, bson.D{}).Raw()
	require.NoError(t, err)
	return raw
}

// Saving from the edit dialog without changing anything must leave the stored
// document byte for byte as it was, at the top level and nested.
func TestIntegration_EditRoundTrip_SaveKeepsBytes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient, 0, "")
	coll := testClient.Database(db).Collection("c")

	for _, tc := range loadEditRoundTripCases(t) {
		t.Run(tc.Name, func(t *testing.T) {
			require.NoError(t, coll.Drop(ctx))
			storedDoc := fmt.Sprintf(`{"_id":{"$numberInt":"1"},"v":%s,"nested":{"list":[%s]}}`, tc.Stored, tc.Stored)
			var doc bson.D
			require.NoError(t, bson.UnmarshalExtJSON([]byte(storedDoc), true, &doc))
			_, err := coll.InsertOne(ctx, doc)
			require.NoError(t, err)
			before := storedRaw(t, ctx, coll)

			// The fixture has to be what the frontend really receives.
			result, err := engine.ExecuteQuery(ctx, testURI, db, `db.c.find({ _id: 1 })`)
			require.NoError(t, err)
			require.Len(t, result.Documents, 1)
			received, err := json.Marshal(result.Documents[0])
			require.NoError(t, err)
			assert.JSONEq(t, storedDoc, string(received))

			body := fmt.Sprintf(`{"v":%s,"nested":{"list":[%s]}}`, tc.Editor, tc.Editor)
			script := fmt.Sprintf(`db.getCollection("c").replaceOne(%s, %s)`,
				ejsonParseScript(t, `{"_id":{"$numberInt":"1"}}`), ejsonParseScript(t, body))
			_, err = engine.ExecuteQuery(ctx, testURI, db, script)
			require.NoError(t, err)

			after := storedRaw(t, ctx, coll)
			assert.Equal(t, before.String(), after.String())
			assert.Equal(t, []byte(before), []byte(after))
		})
	}
}

// The edit and delete filters find a document whatever the type of its _id.
func TestIntegration_EditRoundTrip_FilterEveryId(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient, 0, "")
	coll := testClient.Database(db).Collection("c")

	for _, tc := range loadEditRoundTripCases(t) {
		if tc.UsableAsID != nil && !*tc.UsableAsID {
			continue
		}
		t.Run(tc.Name, func(t *testing.T) {
			require.NoError(t, coll.Drop(ctx))
			filter := fmt.Sprintf(`{"_id":%s}`, tc.Stored)
			var doc bson.D
			require.NoError(t, bson.UnmarshalExtJSON([]byte(filter), true, &doc))
			_, err := coll.InsertOne(ctx, doc)
			require.NoError(t, err)

			_, err = engine.ExecuteQuery(ctx, testURI, db,
				fmt.Sprintf(`db.getCollection("c").deleteOne(%s)`, ejsonParseScript(t, filter)))
			require.NoError(t, err)

			count, err := coll.CountDocuments(ctx, bson.D{})
			require.NoError(t, err)
			assert.Zero(t, count)
		})
	}
}

// A field shaped like a broken wrapper can't be read back, so the save has to
// fail and leave the document alone rather than storing something else.
func TestIntegration_EditRoundTrip_BadWrapperFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient, 0, "")
	coll := testClient.Database(db).Collection("c")
	_, err := coll.InsertOne(ctx, bson.D{{Key: "_id", Value: int32(1)}, {Key: "keep", Value: true}})
	require.NoError(t, err)

	for _, body := range []string{`{"x":{"$numberInt":"db.c.drop()"}}`, `{"x":{"$date":"nope"}}`, `{"x":{"$date":"2024-01-01","name":"x"}}`} {
		_, err := engine.ExecuteQuery(ctx, testURI, db, fmt.Sprintf(`db.getCollection("c").replaceOne(%s, %s)`,
			ejsonParseScript(t, `{"_id":{"$numberInt":"1"}}`), ejsonParseScript(t, body)))
		assert.Error(t, err, body)
	}

	var got bson.D
	require.NoError(t, coll.FindOne(ctx, bson.D{}).Decode(&got))
	assert.Equal(t, bson.D{{Key: "_id", Value: int32(1)}, {Key: "keep", Value: true}}, got)
}

// The dialog reads these counts to tell a save that matched nothing from a real one.
func TestIntegration_EditRoundTrip_MissReportsZero(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient, 0, "")

	for script, field := range map[string]string{
		`db.getCollection("c").replaceOne({ _id: 1 }, { a: 1 })`: "matchedCount",
		`db.getCollection("c").deleteOne({ _id: 1 })`:            "deletedCount",
	} {
		result, err := engine.ExecuteQuery(ctx, testURI, db, script)
		require.NoError(t, err)
		require.Len(t, result.Documents, 1, script)
		received, err := json.Marshal(result.Documents[0])
		require.NoError(t, err)
		var doc map[string]any
		require.NoError(t, json.Unmarshal(received, &doc))
		assert.EqualValues(t, 0, doc[field], string(received))
	}
}

// A strict parse gives values scripts can still use.
func TestIntegration_EJSON_StrictParseValuesStayUsable(t *testing.T) {
	engine, db, ctx := setupNumberData(t)
	got := runScript(t, engine, ctx, db, `
		var v = EJSON.parse('{"d":{"$numberDouble":"5.0"},"s":{"$symbol":"z"},"r":{"$regularExpression":{"pattern":"a","options":"i"}}}', { relaxed: false });
		print(v.d + 1, String(v.d), String(v.s), String(v.r));
	`)
	assert.Equal(t, "6 5 z /a/i", got)
}
