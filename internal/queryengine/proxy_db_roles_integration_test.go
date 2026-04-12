//go:build integration

package queryengine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_CreateRole_GetRole_DropRole(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient)

	script := `db.createRole({ role: "reader1", privileges: [], roles: [{ role: "read", db: "` + db + `" }] })`
	_, err := engine.ExecuteQuery(ctx, testURI, db, script)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.getRole("reader1")`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "reader1")

	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.dropRole("reader1")`)
	require.NoError(t, err)

	result, err = engine.ExecuteQuery(ctx, testURI, db, `db.getRole("reader1")`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "null")
}

func TestIntegration_GetRoles(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.createRole({ role: "r1", privileges: [], roles: [] })`)
	require.NoError(t, err)
	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.createRole({ role: "r2", privileges: [], roles: [] })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.getRoles()`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "r1")
	assert.Contains(t, result.RawOutput, "r2")
}

func TestIntegration_UpdateRole(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.createRole({ role: "r3", privileges: [], roles: [] })`)
	require.NoError(t, err)

	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.updateRole("r3", { roles: [{ role: "read", db: "`+db+`" }] })`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.getRole("r3")`)
	require.NoError(t, err)
	assert.Contains(t, result.RawOutput, "r3")
}

func TestIntegration_GrantRevokePrivilegesToRole(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.createRole({ role: "r4", privileges: [], roles: [] })`)
	require.NoError(t, err)

	priv := `[{ resource: { db: "` + db + `", collection: "" }, actions: ["find"] }]`
	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.grantPrivilegesToRole("r4", `+priv+`)`)
	require.NoError(t, err)

	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.revokePrivilegesFromRole("r4", `+priv+`)`)
	require.NoError(t, err)
}

func TestIntegration_GrantRevokeRolesToRole(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.createRole({ role: "r5", privileges: [], roles: [] })`)
	require.NoError(t, err)

	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.grantRolesToRole("r5", [{ role: "read", db: "`+db+`" }])`)
	require.NoError(t, err)

	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.revokeRolesFromRole("r5", [{ role: "read", db: "`+db+`" }])`)
	require.NoError(t, err)
}

func TestIntegration_DropAllRoles(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := dbName(t)
	defer testClient.Database(db).Drop(ctx)
	engine := NewGojaEngine(testClient)

	_, err := engine.ExecuteQuery(ctx, testURI, db, `db.createRole({ role: "r6", privileges: [], roles: [] })`)
	require.NoError(t, err)
	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.createRole({ role: "r7", privileges: [], roles: [] })`)
	require.NoError(t, err)

	_, err = engine.ExecuteQuery(ctx, testURI, db, `db.dropAllRoles()`)
	require.NoError(t, err)

	result, err := engine.ExecuteQuery(ctx, testURI, db, `db.getRoles()`)
	require.NoError(t, err)
	assert.NotContains(t, result.RawOutput, "\"r6\"")
	assert.NotContains(t, result.RawOutput, "\"r7\"")
}
