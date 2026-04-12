package queryengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectionProxy_InfoMethodsExist(t *testing.T) {
	rt, _ := setupRuntime(t)
	val, err := rt.RunString(`
		const c = db.users;
		typeof c.stats === 'function' &&
		typeof c.isCapped === 'function' &&
		typeof c.dataSize === 'function' &&
		typeof c.storageSize === 'function' &&
		typeof c.totalSize === 'function' &&
		typeof c.totalIndexSize === 'function' &&
		typeof c.getIndexes === 'function' &&
		typeof c.count === 'function' &&
		typeof c.renameCollection === 'function' &&
		typeof c.validate === 'function' &&
		typeof c.findAndModify === 'function'
	`)
	require.NoError(t, err)
	assert.Equal(t, true, val.Export())
}

func TestCollectionProxy_InfoMethods_PanicWithoutClient(t *testing.T) {
	rt, _ := setupRuntime(t)
	_, err := rt.RunString(`db.users.stats()`)
	assert.Error(t, err)
}

func TestCollectionProxy_RenameCollection_RequiresName(t *testing.T) {
	rt, _ := setupRuntime(t)
	_, err := rt.RunString(`db.users.renameCollection()`)
	assert.Error(t, err)
}

func TestCollectionProxy_FindAndModify_RequiresSpec(t *testing.T) {
	rt, _ := setupRuntime(t)
	_, err := rt.RunString(`db.users.findAndModify()`)
	assert.Error(t, err)
}

func TestToInt64(t *testing.T) {
	assert.Equal(t, int64(5), toInt64(int64(5)))
	assert.Equal(t, int64(5), toInt64(int32(5)))
	assert.Equal(t, int64(5), toInt64(int(5)))
	assert.Equal(t, int64(5), toInt64(float64(5.7)))
	assert.Equal(t, int64(0), toInt64("not a number"))
	assert.Equal(t, int64(0), toInt64(nil))
}
