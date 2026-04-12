//go:build integration

package queryengine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_CreateUser_GetUser_DropUser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient)

	script := `db.createUser({ user: "alice", pwd: "pw123456", roles: [{ role: "read", db: "` + db + `" }] })`
	_, err := engine.ExecuteQuery(ctx, testURI, db, script)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.getUser("alice")`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "alice")

	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.dropUser("alice")`)
	require.NoError(t, err)

	result, err = engine.ExecuteQuery(ctx, testURI, db, `db.getUser("alice")`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "null")
}

func TestIntegration_GetUsers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.createUser({ user: "u1", pwd: "pw123456", roles: [] })`)
	require.NoError(t, err)
	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.createUser({ user: "u2", pwd: "pw123456", roles: [] })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.getUsers()`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "u1")
	assert.Contains(t, result.RawOutput, "u2")
}

func TestIntegration_UpdateUser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.createUser({ user: "bob", pwd: "pw123456", roles: [] })`)
	require.NoError(t, err)

	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.updateUser("bob", { roles: [{ role: "read", db: "`+db+`" }] })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.getUser("bob")`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "read")
}

func TestIntegration_ChangeUserPassword(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.createUser({ user: "carol", pwd: "pw123456", roles: [] })`)
	require.NoError(t, err)

	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.changeUserPassword("carol", "newpw123456")`)
	require.NoError(t, err)
}

func TestIntegration_GrantRevokeRolesToUser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.createUser({ user: "dan", pwd: "pw123456", roles: [] })`)
	require.NoError(t, err)

	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.grantRolesToUser("dan", [{ role: "read", db: "`+db+`" }])`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.getUser("dan")`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "read")

	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.revokeRolesFromUser("dan", [{ role: "read", db: "`+db+`" }])`)
	require.NoError(t, err)
}

func TestIntegration_DropAllUsers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.createUser({ user: "e1", pwd: "pw123456", roles: [] })`)
	require.NoError(t, err)
	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.createUser({ user: "e2", pwd: "pw123456", roles: [] })`)
	require.NoError(t, err)

	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.dropAllUsers()`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.getUsers()`)
	require.NoError(t, err)
	assert.NotContains(t, result.RawOutput, "e1")
	assert.NotContains(t, result.RawOutput, "e2")
}
