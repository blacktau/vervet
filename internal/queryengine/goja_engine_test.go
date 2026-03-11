package queryengine

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupRuntime creates a Goja runtime with a database proxy.
// Uses nil client — only suitable for tests that don't execute queries.
func setupRuntime(t *testing.T) (*goja.Runtime, *execContext) {
	t.Helper()
	rt := goja.New()
	ec := &execContext{ctx: context.Background(), client: nil, dbName: "testdb", rt: rt}
	db := newDatabaseProxy(ec)
	err := rt.Set("db", db)
	require.NoError(t, err)
	return rt, ec
}

func TestDatabaseProxy_CollectionAccess_ReturnsNonNil(t *testing.T) {
	rt, _ := setupRuntime(t)
	val, err := rt.RunString(`db.users`)
	require.NoError(t, err)
	assert.NotNil(t, val.Export())
}

func TestDatabaseProxy_GetName_ReturnsDatabaseName(t *testing.T) {
	rt, _ := setupRuntime(t)
	val, err := rt.RunString(`db.getName()`)
	require.NoError(t, err)
	assert.Equal(t, "testdb", val.Export())
}

func TestDatabaseProxy_GetCollection_ReturnsProxy(t *testing.T) {
	rt, _ := setupRuntime(t)
	val, err := rt.RunString(`typeof db.getCollection('movies').find`)
	require.NoError(t, err)
	assert.Equal(t, "function", val.Export())
}

func TestCollectionProxy_Find_ReturnsCursor(t *testing.T) {
	rt, _ := setupRuntime(t)
	val, err := rt.RunString(`
		const cursor = db.users.find({ name: "alice" });
		typeof cursor.limit === 'function' &&
		typeof cursor.skip === 'function' &&
		typeof cursor.sort === 'function' &&
		typeof cursor.toArray === 'function' &&
		typeof cursor.forEach === 'function' &&
		typeof cursor.count === 'function'
	`)
	require.NoError(t, err)
	assert.Equal(t, true, val.Export())
}

func TestCollectionProxy_Find_CursorChaining(t *testing.T) {
	rt, _ := setupRuntime(t)
	val, err := rt.RunString(`
		const cursor = db.users.find({}).limit(10).skip(5).sort({ name: 1 });
		typeof cursor.toArray === 'function'
	`)
	require.NoError(t, err)
	assert.Equal(t, true, val.Export())
}

func TestCollectionProxy_Find_CursorHasLazyCursor(t *testing.T) {
	rt, _ := setupRuntime(t)
	val, err := rt.RunString(`db.users.find({})`)
	require.NoError(t, err)
	cursor := extractLazyCursor(val)
	require.NotNil(t, cursor)
	assert.False(t, cursor.resolved)
	assert.Equal(t, "users", cursor.collection)
}

func TestCollectionProxy_Find_CursorChainingUpdatesState(t *testing.T) {
	rt, _ := setupRuntime(t)
	val, err := rt.RunString(`db.users.find({ age: 25 }).limit(10).skip(5)`)
	require.NoError(t, err)
	cursor := extractLazyCursor(val)
	require.NotNil(t, cursor)
	assert.Equal(t, int64(10), cursor.limit)
	assert.Equal(t, int64(5), cursor.skip)
}

func TestCollectionProxy_FindOne_ReturnsCursorWithFindOneFlag(t *testing.T) {
	rt, _ := setupRuntime(t)
	val, err := rt.RunString(`db.users.findOne({ name: "alice" })`)
	require.NoError(t, err)
	cursor := extractLazyCursor(val)
	require.NotNil(t, cursor)
	assert.True(t, cursor.isFindOne)
}

func TestCollectionProxy_EagerMethods_Exist(t *testing.T) {
	rt, _ := setupRuntime(t)
	methods := []string{
		"insertOne", "insertMany", "updateOne", "updateMany",
		"deleteOne", "deleteMany", "replaceOne", "countDocuments",
		"aggregate", "distinct", "drop", "createIndex", "listIndexes",
	}
	for _, m := range methods {
		val, err := rt.RunString(`typeof db.users.` + m)
		require.NoError(t, err, "method %s", m)
		assert.Equal(t, "function", val.Export(), "method %s should be a function", m)
	}
}

func TestCollectionProxy_EagerMethod_PanicsWithoutClient(t *testing.T) {
	rt, _ := setupRuntime(t)
	// With nil client, eager methods should panic (caught by Goja as exception)
	_, err := rt.RunString(`db.users.insertOne({ name: "bob" })`)
	assert.Error(t, err, "insertOne with nil client should error")
}

func TestPlainExpression_ReturnsValue(t *testing.T) {
	rt, _ := setupRuntime(t)
	val, err := rt.RunString(`const x = 42; x`)
	require.NoError(t, err)
	cursor := extractLazyCursor(val)
	assert.Nil(t, cursor, "plain expression should not be a cursor")
	assert.Equal(t, int64(42), val.Export())
}

func TestPrint_CapturesOutput(t *testing.T) {
	rt := goja.New()
	ec := &execContext{ctx: context.Background(), client: nil, dbName: "testdb", rt: rt}
	db := newDatabaseProxy(ec)
	_ = rt.Set("db", db)

	var printed []string
	_ = rt.Set("print", func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			printed = append(printed, arg.String())
		}
		return goja.Undefined()
	})

	_, err := rt.RunString(`print("hello")`)
	require.NoError(t, err)
	require.Len(t, printed, 1)
	assert.Equal(t, "hello", printed[0])
}

func TestMultiStatement_VariableThenCursor(t *testing.T) {
	rt, _ := setupRuntime(t)
	val, err := rt.RunString(`const filter = { status: "active" }; db.users.find(filter)`)
	require.NoError(t, err)
	cursor := extractLazyCursor(val)
	require.NotNil(t, cursor)
	assert.Equal(t, "users", cursor.collection)
}
