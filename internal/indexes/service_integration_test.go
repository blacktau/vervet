//go:build integration

package indexes

import (
	"context"
	"log"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"vervet/internal/models"
)

var testClient *mongo.Client

type stubProvider struct {
	client *mongo.Client
	err    error
}

func (s stubProvider) GetClient(string) (*mongo.Client, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.client, nil
}

func TestMain(m *testing.M) {
	ctx := context.Background()
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	container, err := mongodb.Run(ctx, "mongo:7")
	if err != nil {
		log.Fatalf("start container: %v", err)
	}
	defer func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			log.Printf("terminate: %v", err)
		}
	}()

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("conn string: %v", err)
	}

	testClient, err = mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer testClient.Disconnect(ctx)

	os.Exit(m.Run())
}

func newService(t *testing.T) *IndexService {
	t.Helper()
	svc := NewIndexService(slog.Default(), stubProvider{client: testClient})
	svc.Init(context.Background())
	return svc
}

// seedIdx creates a database with one collection holding one document, and
// registers cleanup. Returns the db name.
func seedIdx(t *testing.T, dbName string) string {
	t.Helper()
	ctx := context.Background()
	_, err := testClient.Database(dbName).Collection("c").InsertOne(ctx, bson.M{"email": "a@b.c", "age": 1})
	require.NoError(t, err)
	t.Cleanup(func() { testClient.Database(dbName).Drop(ctx) })
	return dbName
}

func findIndex(t *testing.T, list []models.Index, name string) models.Index {
	t.Helper()
	for _, idx := range list {
		if idx.Name == name {
			return idx
		}
	}
	t.Fatalf("index %q not found in %+v", name, list)
	return models.Index{}
}

func TestIntegration_GetIndexes_IncludesDefaultIdIndex(t *testing.T) {
	db := seedIdx(t, "idx_default")

	list, err := newService(t).GetIndexes("srv", db, "c")
	require.NoError(t, err)

	idIdx := findIndex(t, list, "_id_")
	require.Len(t, idIdx.Keys, 1)
	assert.Equal(t, "_id", idIdx.Keys[0].Field)
}

func TestIntegration_CreateIndex_UniqueSparseAndKeys(t *testing.T) {
	db := seedIdx(t, "idx_create")
	svc := newService(t)

	err := svc.CreateIndex("srv", db, "c", models.CreateIndexRequest{
		Name:   "email_unique",
		Keys:   []models.IndexKeyField{{Field: "email", Direction: 1}},
		Unique: true,
		Sparse: true,
	})
	require.NoError(t, err)

	list, err := svc.GetIndexes("srv", db, "c")
	require.NoError(t, err)

	idx := findIndex(t, list, "email_unique")
	assert.True(t, idx.Unique)
	assert.True(t, idx.Sparse)
	require.Len(t, idx.Keys, 1)
	assert.Equal(t, "email", idx.Keys[0].Field)
}

// buildIndexModel coerces float64 directions to int, because JSON decodes all
// numbers as float64. Guard that conversion.
func TestIntegration_CreateIndex_CoercesFloat64Direction(t *testing.T) {
	db := seedIdx(t, "idx_float_dir")
	svc := newService(t)

	err := svc.CreateIndex("srv", db, "c", models.CreateIndexRequest{
		Name: "age_desc",
		Keys: []models.IndexKeyField{{Field: "age", Direction: float64(-1)}},
	})
	require.NoError(t, err, "float64(-1) direction must be coerced to int(-1)")

	list, err := svc.GetIndexes("srv", db, "c")
	require.NoError(t, err)
	findIndex(t, list, "age_desc")
}

func TestIntegration_CreateIndex_TTL(t *testing.T) {
	db := seedIdx(t, "idx_ttl")
	svc := newService(t)

	ttl := int32(3600)
	err := svc.CreateIndex("srv", db, "c", models.CreateIndexRequest{
		Name: "ttl_idx",
		Keys: []models.IndexKeyField{{Field: "createdAt", Direction: 1}},
		TTL:  &ttl,
	})
	require.NoError(t, err)

	list, err := svc.GetIndexes("srv", db, "c")
	require.NoError(t, err)

	idx := findIndex(t, list, "ttl_idx")
	require.NotNil(t, idx.TTL, "TTL must round-trip from expireAfterSeconds")
	assert.Equal(t, int32(3600), *idx.TTL)
}

// This is the bug test. It fails against current main: collStats decodes
// indexSizes as bson.D, not bson.M, so the type assertion in GetIndexes never
// fires and every Size stays 0.
func TestIntegration_GetIndexes_ReportsSize(t *testing.T) {
	db := seedIdx(t, "idx_size")

	list, err := newService(t).GetIndexes("srv", db, "c")
	require.NoError(t, err)

	idIdx := findIndex(t, list, "_id_")
	assert.Greater(t, idIdx.Size, int64(0),
		"index Size must be populated from collStats.indexSizes; got 0, meaning the type assertion failed")
}

func TestIntegration_EditIndex_SameNameRecreates(t *testing.T) {
	db := seedIdx(t, "idx_edit_same")
	svc := newService(t)

	require.NoError(t, svc.CreateIndex("srv", db, "c", models.CreateIndexRequest{
		Name: "e", Keys: []models.IndexKeyField{{Field: "email", Direction: 1}},
	}))

	err := svc.EditIndex("srv", db, "c", models.EditIndexRequest{
		OldName: "e",
		Name:    "e",
		Keys:    []models.IndexKeyField{{Field: "email", Direction: 1}},
		Unique:  true,
	})
	require.NoError(t, err)

	list, err := svc.GetIndexes("srv", db, "c")
	require.NoError(t, err)
	assert.True(t, findIndex(t, list, "e").Unique, "edit must apply the new unique flag")
}

func TestIntegration_EditIndex_RenameCreatesThenDrops(t *testing.T) {
	db := seedIdx(t, "idx_edit_rename")
	svc := newService(t)

	require.NoError(t, svc.CreateIndex("srv", db, "c", models.CreateIndexRequest{
		Name: "old_idx", Keys: []models.IndexKeyField{{Field: "email", Direction: 1}},
	}))

	err := svc.EditIndex("srv", db, "c", models.EditIndexRequest{
		OldName: "old_idx",
		Name:    "new_idx",
		Keys:    []models.IndexKeyField{{Field: "age", Direction: 1}},
	})
	require.NoError(t, err)

	list, err := svc.GetIndexes("srv", db, "c")
	require.NoError(t, err)

	findIndex(t, list, "new_idx")
	for _, idx := range list {
		assert.NotEqual(t, "old_idx", idx.Name, "old index must be dropped after rename")
	}
}

// When same-name recreation fails, EditIndex restores the original index.
// A unique index over duplicate values cannot be built, forcing that path.
func TestIntegration_EditIndex_RestoresOriginalOnFailure(t *testing.T) {
	ctx := context.Background()
	db := seedIdx(t, "idx_edit_rollback")
	svc := newService(t)

	// Two docs sharing an email make a unique index impossible.
	_, err := testClient.Database(db).Collection("c").InsertOne(ctx, bson.M{"email": "a@b.c", "age": 2})
	require.NoError(t, err)

	require.NoError(t, svc.CreateIndex("srv", db, "c", models.CreateIndexRequest{
		Name: "email_idx", Keys: []models.IndexKeyField{{Field: "email", Direction: 1}},
	}))

	err = svc.EditIndex("srv", db, "c", models.EditIndexRequest{
		OldName: "email_idx",
		Name:    "email_idx",
		Keys:    []models.IndexKeyField{{Field: "email", Direction: 1}},
		Unique:  true,
	})
	require.Error(t, err, "unique index over duplicate emails must fail")
	assert.Contains(t, err.Error(), "failed to create replacement index")

	list, err := svc.GetIndexes("srv", db, "c")
	require.NoError(t, err)
	restored := findIndex(t, list, "email_idx")
	assert.False(t, restored.Unique, "original non-unique index must be restored")
}

func TestIntegration_DropIndex_RemovesIt(t *testing.T) {
	db := seedIdx(t, "idx_drop")
	svc := newService(t)

	require.NoError(t, svc.CreateIndex("srv", db, "c", models.CreateIndexRequest{
		Name: "doomed", Keys: []models.IndexKeyField{{Field: "email", Direction: 1}},
	}))
	require.NoError(t, svc.DropIndex("srv", db, "c", "doomed"))

	list, err := svc.GetIndexes("srv", db, "c")
	require.NoError(t, err)
	for _, idx := range list {
		assert.NotEqual(t, "doomed", idx.Name)
	}
}

func TestIntegration_DropIndex_UnknownErrors(t *testing.T) {
	db := seedIdx(t, "idx_drop_unknown")

	err := newService(t).DropIndex("srv", db, "c", "no_such_index")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to drop index")
}

func TestIntegration_GetIndexes_PropagatesProviderError(t *testing.T) {
	svc := NewIndexService(slog.Default(), stubProvider{err: assert.AnError})
	svc.Init(context.Background())

	_, err := svc.GetIndexes("srv", "any", "c")
	assert.ErrorIs(t, err, assert.AnError)
}
