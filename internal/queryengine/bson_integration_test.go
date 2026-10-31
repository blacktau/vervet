//go:build integration

package queryengine

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	testClient *mongo.Client
	testURI    string
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Disable the Ryuk reaper container — it fails under rootless Podman
	// because it needs Docker-socket-level access. Cleanup is handled by
	// defer calls in tests and the TerminateContainer below.
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	mongoContainer, err := mongodb.Run(ctx, "mongo:7")
	if err != nil {
		log.Fatalf("failed to start MongoDB container: %v", err)
	}

	testURI, err = mongoContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}

	testClient, err = mongo.Connect(options.Client().ApplyURI(testURI))
	if err != nil {
		log.Fatalf("failed to connect to MongoDB: %v", err)
	}

	// Cleanup runs before os.Exit, which executes no deferred calls —
	// deferring the terminate here leaked a mongo:7 container per run.
	code := m.Run()

	if err := testClient.Disconnect(ctx); err != nil {
		log.Printf("failed to disconnect: %v", err)
	}
	if err := testcontainers.TerminateContainer(mongoContainer); err != nil {
		log.Printf("failed to terminate container: %v", err)
	}

	os.Exit(code)
}

// dbName returns a unique database name for each test to ensure isolation.
func dbName(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("test_%s", t.Name())
}

// insertAndReadBack runs an insertOne query via the GojaEngine, then reads
// the document back directly via the driver and returns it.
func insertAndReadBack(t *testing.T, query string) bson.M {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0, "")
	_, err := engine.ExecuteQuery(ctx, testURI, db, query)
	require.NoError(t, err)

	var doc bson.M
	err = testClient.Database(db).Collection("test").FindOne(ctx, bson.M{}).Decode(&doc)
	require.NoError(t, err)
	return doc
}

// --- Issue #124: UUID() without arguments ---

func TestIntegration_Issue124_InsertOneWithUUID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0, "")
	query := `db.getCollection("test-collection").insertOne({
		_id: UUID(),
		CheckType: "CustomerOnly",
		ContactId: null,
		CreatedAt: ISODate(),
	});`

	_, err := engine.ExecuteQuery(ctx, testURI, db, query)
	require.NoError(t, err)

	var doc bson.M
	err = testClient.Database(db).Collection("test-collection").FindOne(ctx, bson.M{}).Decode(&doc)
	require.NoError(t, err)

	bin, ok := doc["_id"].(bson.Binary)
	require.True(t, ok, "expected _id to be bson.Binary, got %T", doc["_id"])
	assert.Equal(t, byte(0x04), bin.Subtype)
	assert.Len(t, bin.Data, 16)
	assert.Equal(t, "CustomerOnly", doc["CheckType"])
	assert.Nil(t, doc["ContactId"])
	assert.IsType(t, bson.DateTime(0), doc["CreatedAt"])
}

// --- BSON type constructor tests ---

func TestIntegration_UUID_NoArgs_GeneratesRandomUUID(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ _id: UUID() })`)
	bin, ok := doc["_id"].(bson.Binary)
	require.True(t, ok, "expected bson.Binary, got %T", doc["_id"])
	assert.Equal(t, byte(0x04), bin.Subtype)
	assert.Len(t, bin.Data, 16)
}

func TestIntegration_UUID_WithString_StoresCorrectBytes(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ _id: UUID("550e8400-e29b-41d4-a716-446655440000") })`)
	bin, ok := doc["_id"].(bson.Binary)
	require.True(t, ok, "expected bson.Binary, got %T", doc["_id"])
	assert.Equal(t, byte(0x04), bin.Subtype)
	assert.Equal(t, "550e8400e29b41d4a716446655440000",
		fmt.Sprintf("%x", bin.Data))
}

func TestIntegration_ObjectId_NoArgs(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ _id: ObjectId() })`)
	_, ok := doc["_id"].(bson.ObjectID)
	assert.True(t, ok, "expected bson.ObjectID, got %T", doc["_id"])
}

func TestIntegration_ObjectId_WithHex(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ _id: ObjectId("507f1f77bcf86cd799439011") })`)
	oid, ok := doc["_id"].(bson.ObjectID)
	require.True(t, ok, "expected bson.ObjectID, got %T", doc["_id"])
	assert.Equal(t, "507f1f77bcf86cd799439011", oid.Hex())
}

func TestIntegration_ISODate_NoArgs(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ ts: ISODate() })`)
	_, ok := doc["ts"].(bson.DateTime)
	assert.True(t, ok, "expected bson.DateTime, got %T", doc["ts"])
}

func TestIntegration_ISODate_WithString(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ ts: ISODate("2024-06-15T12:00:00Z") })`)
	dt, ok := doc["ts"].(bson.DateTime)
	require.True(t, ok, "expected bson.DateTime, got %T", doc["ts"])
	expected := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, expected.UnixMilli(), int64(dt))
}

func TestIntegration_NumberInt(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ val: NumberInt(42) })`)
	val, ok := doc["val"].(int32)
	require.True(t, ok, "expected int32, got %T", doc["val"])
	assert.Equal(t, int32(42), val)
}

func TestIntegration_NumberLong(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ val: NumberLong("9007199254740993") })`)
	val, ok := doc["val"].(int64)
	require.True(t, ok, "expected int64, got %T", doc["val"])
	assert.Equal(t, int64(9007199254740993), val)
}

// Plain JS numbers are typed as js-bson types them: int32 when they fit,
// double otherwise (issue #305).
func TestIntegration_PlainNumbersStoreAsInt32OrDouble(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ small: 5, neg: -2147483648, big: 2147483648, frac: 2.5 })`)
	assert.Equal(t, int32(5), doc["small"])
	assert.Equal(t, int32(-2147483648), doc["neg"])
	assert.Equal(t, float64(2147483648), doc["big"])
	assert.Equal(t, 2.5, doc["frac"])
}

// Reading a document and writing it back must not turn its NumberInt fields
// into longs (issue #305). A whole-number double comes back as int32, the
// same limitation mongosh has: the script sees it as a plain number.
func TestIntegration_ReadModifyWriteKeepsNumberTypes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0, "")
	_, err := engine.ExecuteQuery(ctx, testURI, db,
		`db.src.insertOne({ _id: 1, i: NumberInt(5), d: Double(2), f: 2.5, l: NumberLong("7") })`)
	require.NoError(t, err)
	_, err = engine.ExecuteQuery(ctx, testURI, db,
		`db.src.find().forEach(function (doc) { db.dst.insertOne(doc); })`)
	require.NoError(t, err)

	var doc bson.M
	require.NoError(t, testClient.Database(db).Collection("dst").FindOne(ctx, bson.M{}).Decode(&doc))
	assert.Equal(t, int32(1), doc["_id"])
	assert.Equal(t, int32(5), doc["i"])
	assert.Equal(t, int32(2), doc["d"])
	assert.Equal(t, 2.5, doc["f"])
	assert.Equal(t, int64(7), doc["l"])
}

// Documents a command returns from a collection keep their long fields when a
// script writes them back: findAndModify's document and a runCommand cursor's
// batches convert as find() results do.
func TestIntegration_CommandReturnedDocumentsKeepLongs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0, "")
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.src.insertMany([{ _id: 1, l: NumberLong(7) }, { _id: 2, l: NumberLong(8) }])`)
	require.NoError(t, err)
	_, err = engine.ExecuteQuery(ctx, testURI, db, `
		var f = db.src.findAndModify({ query: { _id: 1 }, update: { $set: { seen: true } } });
		db.fam.insertOne(f);
		var r = db.runCommand({ find: 'src', filter: { _id: 1 } });
		db.first.insertOne(r.cursor.firstBatch[0]);
		var c = db.runCommand({ find: 'src', batchSize: 1 });
		var m = db.runCommand({ getMore: c.cursor.id, collection: 'src' });
		db.next.insertOne(m.cursor.nextBatch[0]);
		r.ok + 1`)
	require.NoError(t, err)

	for coll, want := range map[string]int64{"fam": 7, "first": 7, "next": 8} {
		var doc bson.M
		require.NoError(t, testClient.Database(db).Collection(coll).FindOne(ctx, bson.M{}).Decode(&doc))
		assert.Equal(t, want, doc["l"], coll)
	}
}

// Later pages and the count run the same filter as page 1: a long in the
// filter keeps its exact value and type through the PageContext round trip.
func TestIntegration_PagedFilterKeepsLongs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 2, "")
	_, err := engine.ExecuteQuery(ctx, testURI, db, `
		for (var i = 0; i < 5; i++) { db.p.insertOne({ k: NumberLong("9007199254740993"), i: i }); }
		db.p.insertOne({ k: NumberLong("9007199254740992"), i: 99 })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.p.find({ k: NumberLong("9007199254740993") }, { _id: 0, k: 0 }).sort({ i: 1 })`)
	require.NoError(t, err)
	require.NotNil(t, result.PageContext)
	count, _, err := engine.CountForPage(ctx, *result.PageContext)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
	page3, err := engine.FetchPage(ctx, *result.PageContext, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, `[{"i":{"$numberInt":"4"}}]`, resultText(page3))
}

// findAndModify with no matching document returns null, as in mongosh.
func TestIntegration_FindAndModifyNoMatchIsNull(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0, "")
	result, err := engine.ExecuteQuery(ctx, testURI, db, `String(db.none.findAndModify({ query: { _id: 1 }, update: { $set: { a: 1 } } }))`)
	require.NoError(t, err)
	assert.Equal(t, "null", result.RawOutput)
}

// findOne returns the document itself, or null when nothing matches, as in
// mongosh (#303), so a script can read its fields and write it back.
func TestIntegration_FindOneReturnsDocument(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.c.insertOne({ _id: 1, name: "alice" })`)
	require.NoError(t, err)

	for _, tc := range []struct{ script, want string }{
		{`typeof db.c.findOne({ _id: 1 }).name`, "string"},
		{`Object.keys(db.c.findOne({ _id: 1 })).join(",")`, "_id,name"},
		{`String(db.c.findOne({ _id: 999 }) === null)`, "true"},
		{`db.c.replaceOne({ _id: 1 }, db.c.findOne({ _id: 1 })).matchedCount`, "1"},
		{`Object.keys(db.c.findOne({ _id: 1 }, { name: 0 })).join(",")`, "_id"},
		{`db.c.findOne({ _id: 1 })`, `[{"_id":{"$numberInt":"1"},"name":"alice"}]`},
		{`db.c.findOne({ _id: 999 })`, "null"},
	} {
		result, err := engine.ExecuteQuery(ctx, testURI, db, tc.script)
		require.NoError(t, err, tc.script)
		assert.Equal(t, tc.want, resultText(result), tc.script)
	}
}

// findOne takes the find options mongosh accepts in its third argument.
func TestIntegration_FindOneOptions(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.c.insertMany([{ _id: 1, name: "b" }, { _id: 2, name: "a" }, { _id: 3, name: "c" }])`)
	require.NoError(t, err)

	for _, tc := range []struct{ script, want string }{
		{`db.c.findOne({}, null, { sort: { name: 1 } })._id`, "2"},
		{`db.c.findOne({}, null, { sort: { name: 1 }, skip: 1 })._id`, "1"},
		{`Object.keys(db.c.findOne({}, null, { projection: { name: 0 } })).join(",")`, "_id"},
		{`db.c.findOne({}, { _id: 0 }, { projection: { name: 0 } }).name`, "b"},
		{`db.c.findOne({}, null, { sort: { name: -1 }, comment: "x", maxTimeMS: 5000 })._id`, "3"},
		{`db.c.findOne({ name: "A" }, null, { collation: { locale: "en", strength: 2 } })._id`, "2"},
		{`db.c.findOne({}, null, { sort: { name: 1 }, skip: NumberLong(2), maxTimeMS: NumberInt(5000) })._id`, "3"},
		{`db.c.findOne({}, null, { sort: { name: 1 }, limit: 5, batchSize: 10, noCursorTimeout: true })._id`, "2"},
	} {
		result, err := engine.ExecuteQuery(ctx, testURI, db, tc.script)
		require.NoError(t, err, tc.script)
		assert.Equal(t, tc.want, resultText(result), tc.script)
	}

	for _, tc := range []struct{ script, want string }{
		{`db.c.findOne({}, null, { skip: -1 })`, "findOne skip must not be negative"},
		{`db.c.findOne({}, null, { skip: "1" })`, "findOne skip must be a number"},
		{`db.c.findOne({}, null, { allowDiskUse: true })`, `findOne option "allowDiskUse" is not supported`},
		{`db.c.findOne({}, null, { comment: 5 })`, "findOne comment must be a string"},
		{`db.c.findOne({}, null, { skip: -1, foo: 1 })`, "findOne skip must not be negative"},
		{`db.c.findOne({}, null, { foo: 1, skip: -1 })`, `findOne option "foo" is not supported`},
		{`db.c.findOne({}, null, { skip: 1e19 })`, "findOne skip is too large"},
		{`db.c.findOne({}, null, { skip: Infinity })`, "findOne skip is too large"},
		{`db.c.findOne({}, null, { maxTimeMS: 1e13 })`, "maxTimeMS must be at most 2147483647"},
		{`db.c.find().maxTimeMS(1e13).toArray()`, "maxTimeMS must be at most 2147483647"},
	} {
		_, err := engine.ExecuteQuery(ctx, testURI, db, tc.script)
		assert.ErrorContains(t, err, tc.want, tc.script)
	}

	// A server error is reported once, as findOne's.
	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.c.findOne({ $bogus: 1 })`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "findOne failed: ")
	assert.NotContains(t, err.Error(), "find failed")
}

// A findOne document left as the script's final value is shown with the types
// the server stored, which a whole-number double loses on its way through JS.
// Once the script changes the document, it is shown as the script left it.
// Like a cursor, an unchanged document is shown even when the script printed.
func TestIntegration_FindOneShowsStoredTypes(t *testing.T) {
	ctx, db, engine := keyOrderEngine(t)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.c.insertOne({ _id: 1, d: Double(5) })`)
	require.NoError(t, err)
	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.c.insertOne({ _id: ObjectId("650000000000000000000001"), d: Double(5), l: Long(7), at: new Date(0) })`)
	require.NoError(t, err)
	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.c.insertOne({ _id: 2, d: Double(5), n: NaN })`)
	require.NoError(t, err)

	for _, tc := range []struct{ script, want string }{
		{`db.c.findOne({ _id: 1 })`, `[{"_id":{"$numberInt":"1"},"d":{"$numberDouble":"5.0"}}]`},
		{`db.c.findOne({ l: { $exists: true } })`, `[{"_id":{"$oid":"650000000000000000000001"},"d":{"$numberDouble":"5.0"},"l":{"$numberLong":"7"},"at":{"$date":{"$numberLong":"0"}}}]`},
		{`var d = db.c.findOne({ _id: 1 }); d.d; d`, `[{"_id":{"$numberInt":"1"},"d":{"$numberDouble":"5.0"}}]`},
		{`var d = db.c.findOne({ _id: 1 }); d.x = "y"; d`, `[{"_id":1,"d":5,"x":"y"}]`},
		{`db.c.findOne({ _id: 2 })`, `[{"_id":{"$numberInt":"2"},"d":{"$numberDouble":"5.0"},"n":{"$numberDouble":"NaN"}}]`},
		{`print("checking"); db.c.findOne({ _id: 1 })`, `[{"_id":{"$numberInt":"1"},"d":{"$numberDouble":"5.0"}}]`},
		{`var d = db.c.findOne({ _id: 1 }); d.x = "y"; print("changed"); d`, "changed"},
	} {
		result, err := engine.ExecuteQuery(ctx, testURI, db, tc.script)
		require.NoError(t, err, tc.script)
		assert.Equal(t, tc.want, resultText(result), tc.script)
	}
}

// Every long in a command reply is a Long, metadata included (#315): a capped
// collection created with a long size lists that size as a Long, which still
// does arithmetic.
func TestIntegration_CommandReplyLongsAreLongs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0, "")
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.runCommand({ create: 'capped', capped: true, size: NumberLong("5000000000") })`)
	require.NoError(t, err)
	result, err := engine.ExecuteQuery(ctx, testURI, db, `
		var o = db.runCommand({ listCollections: 1, filter: { name: 'capped' } }).cursor.firstBatch[0].options;
		typeof o.size + " " + (o.size + 1)`)
	require.NoError(t, err)
	assert.Equal(t, "object 5000000001", result.RawOutput)
}

// A server long beyond ±2^53 keeps its exact value and type through a script
// (review of #314): EJSON.parse values, and cursor ids passed to getMore.
func TestIntegration_LongsBeyondSafeIntegerRoundTrip(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne(EJSON.parse('{"n": {"$numberLong": "9007199254740993"}}'))`)
	assert.Equal(t, int64(9007199254740993), doc["n"])

	// No deadline, as in the app: the driver turns a deadline into maxTimeMS,
	// which the server rejects on getMore.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := dbName(t) + "_cursor"
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0, "")
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.c.insertMany([{ n: 1 }, { n: 2 }, { n: 3 }])`)
	require.NoError(t, err)
	result, err := engine.ExecuteQuery(ctx, testURI, db, `
		var r = db.runCommand({ find: 'c', batchSize: 1 });
		var m = db.runCommand({ getMore: r.cursor.id, collection: 'c' });
		String(m.ok) + ' ' + m.cursor.nextBatch.length`)
	require.NoError(t, err)
	assert.Equal(t, "1 2", result.RawOutput)
}

func TestIntegration_NumberDecimal(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ val: NumberDecimal("123.456") })`)
	_, ok := doc["val"].(bson.Decimal128)
	assert.True(t, ok, "expected bson.Decimal128, got %T", doc["val"])
}

func TestIntegration_Timestamp(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ ts: Timestamp(1700000000, 1) })`)
	ts, ok := doc["ts"].(bson.Timestamp)
	require.True(t, ok, "expected bson.Timestamp, got %T", doc["ts"])
	assert.Equal(t, uint32(1700000000), ts.T)
	assert.Equal(t, uint32(1), ts.I)
}

func TestIntegration_MinKey(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ val: MinKey() })`)
	_, ok := doc["val"].(bson.MinKey)
	assert.True(t, ok, "expected bson.MinKey, got %T", doc["val"])
}

func TestIntegration_MaxKey(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ val: MaxKey() })`)
	_, ok := doc["val"].(bson.MaxKey)
	assert.True(t, ok, "expected bson.MaxKey, got %T", doc["val"])
}

func TestIntegration_BinData(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ data: BinData(0, "aGVsbG8=") })`)
	bin, ok := doc["data"].(bson.Binary)
	require.True(t, ok, "expected bson.Binary, got %T", doc["data"])
	assert.Equal(t, byte(0x00), bin.Subtype)
	assert.Equal(t, []byte("hello"), bin.Data)
}

// --- Regex in queries ---

func TestIntegration_Regex_FindMatchesCorrectly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0, "")

	// Insert test documents
	setup := `db.test.insertMany([
		{ name: "Alice" },
		{ name: "Bob" },
		{ name: "alice_smith" },
	])`
	_, err := engine.ExecuteQuery(ctx, testURI, db, setup)
	require.NoError(t, err)

	// Find with case-insensitive regex
	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.find({ name: /alice/i })`)
	require.NoError(t, err)
	assert.Len(t, result.Documents, 2, "expected 2 documents matching /alice/i")
}

// --- Issue #148: distinct on int64 field must preserve EJSON structure ---

func TestIntegration_Issue148_DistinctLongs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0, "")

	setup := `db.test.insertMany([
		{ AField: 1, LongId: NumberLong("7151") },
		{ AField: 1, LongId: NumberLong("11788") },
		{ AField: 1, LongId: NumberLong("7151") },
		{ AField: 2, LongId: NumberLong("99999") },
	])`
	_, err := engine.ExecuteQuery(ctx, testURI, db, setup)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.getCollection("test").distinct("LongId", { AField: 1 })`)
	require.NoError(t, err)

	assert.Empty(t, result.RawOutput, "distinct result must not be stringified into RawOutput")
	require.Len(t, result.Documents, 2, "distinct returns the values themselves, as mongosh does")

	for _, v := range result.Documents {
		m, ok := asMap(v)
		require.True(t, ok, "expected EJSON document for long, got %T", v)
		_, hasLong := m["$numberLong"]
		assert.True(t, hasLong, "expected $numberLong key, got %v", m)
	}
}

// distinct hands a script the bare array mongosh returns, so ids stringify to
// their digits instead of collapsing to "[object Object]".
func TestIntegration_Issue148_DistinctLongsInScript(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0, "")
	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.insertMany([
		{ AField: 1, LongId: NumberLong("7151") },
		{ AField: 1, LongId: NumberLong("11788") },
	])`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `
		const ids = db.test.distinct("LongId", { AField: 1 });
		print(Array.isArray(ids) + ":" + ids.map(v => String(v)).sort().join(","))
	`)
	require.NoError(t, err)
	assert.Equal(t, "true:11788,7151", result.RawOutput)
}

// --- Write-method results are single objects in scripts, matching mongosh ---

// A multi-line script that stores an insertMany result in a variable and
// reads a field off it (as mongosh does). If the eager-method return were
// wrapped in an array, `result.insertedIds` would be undefined and the print
// would report 0 even though the inserts happened.
func TestIntegration_InsertManyResultInScript(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0, "")

	query := `
		const items = [1, 2, 3];
		const payload = items.map(id => ({ id: id, status: "active" }));
		const result = db.myCollection.insertMany(payload);
		print("Inserted " + (result.insertedIds ? Object.keys(result.insertedIds).length : 0));
	`

	res, err := engine.ExecuteQuery(ctx, testURI, db, query)
	require.NoError(t, err)

	count, err := testClient.Database(db).Collection("myCollection").CountDocuments(ctx, bson.M{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), count, "three documents should be inserted")

	assert.Equal(t, "Inserted 3", res.RawOutput,
		"script must see result.insertedIds — regressed if this says 'Inserted 0'")
}

// Guards the single-object shape for the most common write/read-single ops
// so future changes to toGojaValue/singleToResult don't silently regress to
// an array wrapper.
func TestIntegration_WriteResultsAreObjectsNotArrays(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0, "")

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.c.insertMany([{n:1},{n:2},{n:3}])`)
	require.NoError(t, err)

	cases := []struct {
		name  string
		query string
	}{
		{"insertOne", `const r = db.c.insertOne({n: 99}); print(typeof r.acknowledged + ":" + (r.insertedId ? "id" : "nil"))`},
		{"updateOne", `const r = db.c.updateOne({n:1}, {$set:{n:10}}); print(typeof r.acknowledged + ":" + r.matchedCount)`},
		{"deleteOne", `const r = db.c.deleteOne({n:10}); print(typeof r.acknowledged + ":" + r.deletedCount)`},
		// countDocuments and distinct return a number and an array in mongosh,
		// not documents to read fields off.
		{"countDocuments", `const r = db.c.countDocuments({}); print(typeof r + ":" + r)`},
		{"distinct", `const r = db.c.distinct("n"); print(Array.isArray(r) + ":" + r.length)`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := engine.ExecuteQuery(ctx, testURI, db, tc.query)
			require.NoError(t, err)
			assert.NotEmpty(t, res.RawOutput)
			assert.NotContains(t, res.RawOutput, "undefined",
				"field access on result returned undefined — result is likely still array-wrapped")
		})
	}
}

// Write-result counts are plain numbers, as in mongosh, so a zero count is
// falsy and === compares by value.
func TestIntegration_WriteResultCountsAreNumbers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0, "")
	result, err := engine.ExecuteQuery(ctx, testURI, db, `
		const counts = [
			db.c.deleteOne({ x: 1 }).deletedCount,
			db.c.deleteMany({ x: 1 }).deletedCount,
			db.c.updateOne({ x: 1 }, { $set: { y: 1 } }).matchedCount,
			db.c.updateMany({ x: 1 }, { $set: { y: 1 } }).modifiedCount,
			db.c.bulkWrite([{ insertOne: { document: { x: 2 } } }]).insertedCount,
		];
		counts.map(n => typeof n + ":" + (n === 0) + ":" + !!n).join(" ")`)
	require.NoError(t, err)
	assert.Equal(t, "number:true:false number:true:false number:true:false number:true:false number:false:true", result.RawOutput)
}

func TestIntegration_Regex_NestedInOperator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient, 0, "")

	setup := `db.test.insertMany([
		{ name: "foo-bar" },
		{ name: "baz-qux" },
		{ name: "foo-baz" },
	])`
	_, err := engine.ExecuteQuery(ctx, testURI, db, setup)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.test.find({ name: { $regex: /^foo/ } })`)
	require.NoError(t, err)
	assert.Len(t, result.Documents, 2, "expected 2 documents matching /^foo/")
}
