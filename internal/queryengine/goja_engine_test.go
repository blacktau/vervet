package queryengine

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRuntime(t *testing.T) *goja.Runtime {
	t.Helper()
	rt := goja.New()
	db := newDatabaseProxy(rt, "testdb")
	err := rt.Set("db", db)
	require.NoError(t, err)
	return rt
}

func TestDatabaseProxy_CollectionAccess_ReturnsNonNil(t *testing.T) {
	rt := setupRuntime(t)

	val, err := rt.RunString(`db.users`)
	require.NoError(t, err)
	assert.NotNil(t, val.Export(), "db.users should return a non-nil value")
}

func TestDatabaseProxy_GetName_ReturnsDatabaseName(t *testing.T) {
	rt := setupRuntime(t)

	val, err := rt.RunString(`db.getName()`)
	require.NoError(t, err)
	assert.Equal(t, "testdb", val.Export())
}

func TestCollectionProxy_Find_CapturesOp(t *testing.T) {
	rt := setupRuntime(t)

	val, err := rt.RunString(`db.users.find({ name: "alice" })`)
	require.NoError(t, err)

	exported := val.Export()
	op, ok := exported.(*CapturedOp)
	require.True(t, ok, "expected *CapturedOp, got %T", exported)
	assert.Equal(t, "users", op.Collection)
	assert.Equal(t, "find", op.Method)
	assert.Len(t, op.Args, 1)

	filter, ok := op.Args[0].(map[string]any)
	require.True(t, ok, "expected map[string]any, got %T", op.Args[0])
	assert.Equal(t, "alice", filter["name"])
}

func TestCollectionProxy_Find_WithProjection_CapturesTwoArgs(t *testing.T) {
	rt := setupRuntime(t)

	val, err := rt.RunString(`db.users.find({ age: { $gt: 21 } }, { name: 1, email: 1 })`)
	require.NoError(t, err)

	exported := val.Export()
	op, ok := exported.(*CapturedOp)
	require.True(t, ok, "expected *CapturedOp, got %T", exported)
	assert.Equal(t, "users", op.Collection)
	assert.Equal(t, "find", op.Method)
	assert.Len(t, op.Args, 2)

	filter, ok := op.Args[0].(map[string]any)
	require.True(t, ok)
	ageFilter, ok := filter["age"].(map[string]any)
	require.True(t, ok, "filter.age should be a map")
	assert.Equal(t, int64(21), ageFilter["$gt"])

	projection, ok := op.Args[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, int64(1), projection["name"])
	assert.Equal(t, int64(1), projection["email"])
}

func TestCollectionProxy_FindOne_CapturesOp(t *testing.T) {
	rt := setupRuntime(t)

	val, err := rt.RunString(`db.users.findOne({ _id: "abc123" })`)
	require.NoError(t, err)

	exported := val.Export()
	op, ok := exported.(*CapturedOp)
	require.True(t, ok, "expected *CapturedOp, got %T", exported)
	assert.Equal(t, "users", op.Collection)
	assert.Equal(t, "findOne", op.Method)
}

func TestCollectionProxy_InsertOne_CapturesOp(t *testing.T) {
	rt := setupRuntime(t)

	val, err := rt.RunString(`db.users.insertOne({ name: "bob", age: 30 })`)
	require.NoError(t, err)

	exported := val.Export()
	op, ok := exported.(*CapturedOp)
	require.True(t, ok, "expected *CapturedOp, got %T", exported)
	assert.Equal(t, "users", op.Collection)
	assert.Equal(t, "insertOne", op.Method)
	assert.Len(t, op.Args, 1)

	doc, ok := op.Args[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "bob", doc["name"])
	assert.Equal(t, int64(30), doc["age"])
}

func TestCollectionProxy_Aggregate_CapturesOp(t *testing.T) {
	rt := setupRuntime(t)

	val, err := rt.RunString(`db.orders.aggregate([{ $match: { status: "A" } }])`)
	require.NoError(t, err)

	exported := val.Export()
	op, ok := exported.(*CapturedOp)
	require.True(t, ok, "expected *CapturedOp, got %T", exported)
	assert.Equal(t, "orders", op.Collection)
	assert.Equal(t, "aggregate", op.Method)
	assert.Len(t, op.Args, 1)

	pipeline, ok := op.Args[0].([]any)
	require.True(t, ok, "expected []any for pipeline, got %T", op.Args[0])
	assert.Len(t, pipeline, 1)
}

func TestCollectionProxy_DeleteOne_CapturesOp(t *testing.T) {
	rt := setupRuntime(t)

	val, err := rt.RunString(`db.users.deleteOne({ name: "alice" })`)
	require.NoError(t, err)

	exported := val.Export()
	op, ok := exported.(*CapturedOp)
	require.True(t, ok, "expected *CapturedOp, got %T", exported)
	assert.Equal(t, "users", op.Collection)
	assert.Equal(t, "deleteOne", op.Method)
}

func TestCollectionProxy_UpdateOne_CapturesOp(t *testing.T) {
	rt := setupRuntime(t)

	val, err := rt.RunString(`db.users.updateOne({ name: "alice" }, { $set: { age: 31 } })`)
	require.NoError(t, err)

	exported := val.Export()
	op, ok := exported.(*CapturedOp)
	require.True(t, ok, "expected *CapturedOp, got %T", exported)
	assert.Equal(t, "users", op.Collection)
	assert.Equal(t, "updateOne", op.Method)
	assert.Len(t, op.Args, 2)
}

func TestCollectionProxy_CountDocuments_CapturesOp(t *testing.T) {
	rt := setupRuntime(t)

	val, err := rt.RunString(`db.users.countDocuments({ active: true })`)
	require.NoError(t, err)

	exported := val.Export()
	op, ok := exported.(*CapturedOp)
	require.True(t, ok, "expected *CapturedOp, got %T", exported)
	assert.Equal(t, "users", op.Collection)
	assert.Equal(t, "countDocuments", op.Method)
}

func TestMultiStatement_VariableThenQuery_CapturesOp(t *testing.T) {
	rt := setupRuntime(t)

	val, err := rt.RunString(`const filter = { status: "active" }; db.users.find(filter)`)
	require.NoError(t, err)

	exported := val.Export()
	op, ok := exported.(*CapturedOp)
	require.True(t, ok, "expected *CapturedOp, got %T", exported)
	assert.Equal(t, "users", op.Collection)
	assert.Equal(t, "find", op.Method)
	assert.Len(t, op.Args, 1)

	filter, ok := op.Args[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "active", filter["status"])
}

func TestPlainExpression_ReturnsValue_NotCapturedOp(t *testing.T) {
	rt := setupRuntime(t)

	val, err := rt.RunString(`const x = 42; x`)
	require.NoError(t, err)

	exported := val.Export()
	_, ok := exported.(*CapturedOp)
	assert.False(t, ok, "plain expression should not be a CapturedOp")
	assert.Equal(t, int64(42), exported)
}

func TestPrint_CapturesOutput(t *testing.T) {
	rt := goja.New()
	db := newDatabaseProxy(rt, "testdb")
	err := rt.Set("db", db)
	require.NoError(t, err)

	var printed []string
	err = rt.Set("print", func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			printed = append(printed, arg.String())
		}
		return goja.Undefined()
	})
	require.NoError(t, err)

	_, err = rt.RunString(`print("hello")`)
	require.NoError(t, err)

	require.Len(t, printed, 1)
	assert.Equal(t, "hello", printed[0])
}

func TestDifferentCollections_CaptureCorrectName(t *testing.T) {
	rt := setupRuntime(t)

	val, err := rt.RunString(`db.products.find({})`)
	require.NoError(t, err)

	op, ok := val.Export().(*CapturedOp)
	require.True(t, ok)
	assert.Equal(t, "products", op.Collection)

	val, err = rt.RunString(`db.orders.findOne({ orderId: 123 })`)
	require.NoError(t, err)

	op, ok = val.Export().(*CapturedOp)
	require.True(t, ok)
	assert.Equal(t, "orders", op.Collection)
}

func TestCollectionProxy_InsertMany_CapturesOp(t *testing.T) {
	rt := setupRuntime(t)

	val, err := rt.RunString(`db.users.insertMany([{ name: "a" }, { name: "b" }])`)
	require.NoError(t, err)

	exported := val.Export()
	op, ok := exported.(*CapturedOp)
	require.True(t, ok, "expected *CapturedOp, got %T", exported)
	assert.Equal(t, "users", op.Collection)
	assert.Equal(t, "insertMany", op.Method)
	assert.Len(t, op.Args, 1)

	docs, ok := op.Args[0].([]any)
	require.True(t, ok)
	assert.Len(t, docs, 2)
}

func TestCollectionProxy_ReplaceOne_CapturesOp(t *testing.T) {
	rt := setupRuntime(t)

	val, err := rt.RunString(`db.users.replaceOne({ name: "alice" }, { name: "alice", age: 32 })`)
	require.NoError(t, err)

	exported := val.Export()
	op, ok := exported.(*CapturedOp)
	require.True(t, ok, "expected *CapturedOp, got %T", exported)
	assert.Equal(t, "users", op.Collection)
	assert.Equal(t, "replaceOne", op.Method)
	assert.Len(t, op.Args, 2)
}
