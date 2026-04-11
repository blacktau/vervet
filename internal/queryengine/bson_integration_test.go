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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	defer func() {
		if err := testcontainers.TerminateContainer(mongoContainer); err != nil {
			log.Printf("failed to terminate container: %v", err)
		}
	}()

	testURI, err = mongoContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}

	testClient, err = mongo.Connect(ctx, options.Client().ApplyURI(testURI))
	if err != nil {
		log.Fatalf("failed to connect to MongoDB: %v", err)
	}
	defer testClient.Disconnect(ctx)

	os.Exit(m.Run())
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

	engine := NewGojaEngine(testClient)
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

	engine := NewGojaEngine(testClient)
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

	bin, ok := doc["_id"].(primitive.Binary)
	require.True(t, ok, "expected _id to be primitive.Binary, got %T", doc["_id"])
	assert.Equal(t, byte(0x04), bin.Subtype)
	assert.Len(t, bin.Data, 16)
	assert.Equal(t, "CustomerOnly", doc["CheckType"])
	assert.Nil(t, doc["ContactId"])
	assert.IsType(t, primitive.DateTime(0), doc["CreatedAt"])
}

// --- BSON type constructor tests ---

func TestIntegration_UUID_NoArgs_GeneratesRandomUUID(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ _id: UUID() })`)
	bin, ok := doc["_id"].(primitive.Binary)
	require.True(t, ok, "expected primitive.Binary, got %T", doc["_id"])
	assert.Equal(t, byte(0x04), bin.Subtype)
	assert.Len(t, bin.Data, 16)
}

func TestIntegration_UUID_WithString_StoresCorrectBytes(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ _id: UUID("550e8400-e29b-41d4-a716-446655440000") })`)
	bin, ok := doc["_id"].(primitive.Binary)
	require.True(t, ok, "expected primitive.Binary, got %T", doc["_id"])
	assert.Equal(t, byte(0x04), bin.Subtype)
	assert.Equal(t, "550e8400e29b41d4a716446655440000",
		fmt.Sprintf("%x", bin.Data))
}

func TestIntegration_ObjectId_NoArgs(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ _id: ObjectId() })`)
	_, ok := doc["_id"].(primitive.ObjectID)
	assert.True(t, ok, "expected primitive.ObjectID, got %T", doc["_id"])
}

func TestIntegration_ObjectId_WithHex(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ _id: ObjectId("507f1f77bcf86cd799439011") })`)
	oid, ok := doc["_id"].(primitive.ObjectID)
	require.True(t, ok, "expected primitive.ObjectID, got %T", doc["_id"])
	assert.Equal(t, "507f1f77bcf86cd799439011", oid.Hex())
}

func TestIntegration_ISODate_NoArgs(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ ts: ISODate() })`)
	_, ok := doc["ts"].(primitive.DateTime)
	assert.True(t, ok, "expected primitive.DateTime, got %T", doc["ts"])
}

func TestIntegration_ISODate_WithString(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ ts: ISODate("2024-06-15T12:00:00Z") })`)
	dt, ok := doc["ts"].(primitive.DateTime)
	require.True(t, ok, "expected primitive.DateTime, got %T", doc["ts"])
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

func TestIntegration_NumberDecimal(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ val: NumberDecimal("123.456") })`)
	_, ok := doc["val"].(primitive.Decimal128)
	assert.True(t, ok, "expected primitive.Decimal128, got %T", doc["val"])
}

func TestIntegration_Timestamp(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ ts: Timestamp(1700000000, 1) })`)
	ts, ok := doc["ts"].(primitive.Timestamp)
	require.True(t, ok, "expected primitive.Timestamp, got %T", doc["ts"])
	assert.Equal(t, uint32(1700000000), ts.T)
	assert.Equal(t, uint32(1), ts.I)
}

func TestIntegration_MinKey(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ val: MinKey() })`)
	_, ok := doc["val"].(primitive.MinKey)
	assert.True(t, ok, "expected primitive.MinKey, got %T", doc["val"])
}

func TestIntegration_MaxKey(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ val: MaxKey() })`)
	_, ok := doc["val"].(primitive.MaxKey)
	assert.True(t, ok, "expected primitive.MaxKey, got %T", doc["val"])
}

func TestIntegration_BinData(t *testing.T) {
	doc := insertAndReadBack(t, `db.test.insertOne({ data: BinData(0, "aGVsbG8=") })`)
	bin, ok := doc["data"].(primitive.Binary)
	require.True(t, ok, "expected primitive.Binary, got %T", doc["data"])
	assert.Equal(t, byte(0x00), bin.Subtype)
	assert.Equal(t, []byte("hello"), bin.Data)
}

// --- Regex in queries ---

func TestIntegration_Regex_FindMatchesCorrectly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)

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

func TestIntegration_Regex_NestedInOperator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)

	engine := NewGojaEngine(testClient)

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
