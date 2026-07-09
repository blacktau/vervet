//go:build integration

package collections

import (
	"context"
	"log"
	"log/slog"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

func newService(t *testing.T) *CollectionsService {
	t.Helper()
	svc := NewCollectionsService(slog.Default(), stubProvider{client: testClient})
	svc.Init(context.Background())
	return svc
}

func seedColl(t *testing.T, dbName, collName string) {
	t.Helper()
	ctx := context.Background()
	_, err := testClient.Database(dbName).Collection(collName).InsertOne(ctx, bson.M{"x": 1})
	require.NoError(t, err)
	t.Cleanup(func() { testClient.Database(dbName).Drop(ctx) })
}

func TestIntegration_GetCollections_ListsAndSorts(t *testing.T) {
	db := "coll_list"
	seedColl(t, db, "zebra")
	seedColl(t, db, "alpha")

	names, err := newService(t).GetCollections("srv", db)
	require.NoError(t, err)

	assert.Contains(t, names, "alpha")
	assert.Contains(t, names, "zebra")
	assert.True(t, slices.IsSorted(names), "want sorted, got %v", names)
}

func TestIntegration_GetViews_ReturnsOnlyViews(t *testing.T) {
	ctx := context.Background()
	db := "coll_views"
	seedColl(t, db, "base")

	err := testClient.Database(db).RunCommand(ctx, bson.D{
		{Key: "create", Value: "myview"},
		{Key: "viewOn", Value: "base"},
		{Key: "pipeline", Value: bson.A{}},
	}).Err()
	require.NoError(t, err)

	views, err := newService(t).GetViews("srv", db)
	require.NoError(t, err)

	assert.Equal(t, []string{"myview"}, views)
	assert.NotContains(t, views, "base")
}

func TestIntegration_CreateCollection_ThenListed(t *testing.T) {
	ctx := context.Background()
	db := "coll_create"
	t.Cleanup(func() { testClient.Database(db).Drop(ctx) })

	svc := newService(t)
	require.NoError(t, svc.CreateCollection("srv", db, "fresh"))

	names, err := svc.GetCollections("srv", db)
	require.NoError(t, err)
	assert.Contains(t, names, "fresh")
}

// MongoDB's create command is idempotent when the existing collection's options
// match, so re-creating is success rather than a NamespaceExists error.
func TestIntegration_CreateCollection_DuplicateIsNoOp(t *testing.T) {
	db := "coll_dup"
	seedColl(t, db, "dup")

	assert.NoError(t, newService(t).CreateCollection("srv", db, "dup"))
}

func TestIntegration_CreateCollection_InvalidNameErrors(t *testing.T) {
	db := "coll_invalid"

	err := newService(t).CreateCollection("srv", db, "bad$name")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create collection")
}

func TestIntegration_RenameCollection_MovesDocuments(t *testing.T) {
	ctx := context.Background()
	db := "coll_rename"
	seedColl(t, db, "oldname")

	svc := newService(t)
	require.NoError(t, svc.RenameCollection("srv", db, "oldname", "newname"))

	names, err := svc.GetCollections("srv", db)
	require.NoError(t, err)
	assert.Contains(t, names, "newname")
	assert.NotContains(t, names, "oldname")

	count, err := testClient.Database(db).Collection("newname").CountDocuments(ctx, bson.D{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "documents must survive the rename")
}

func TestIntegration_RenameCollection_RejectsEmptyName(t *testing.T) {
	err := newService(t).RenameCollection("srv", "any", "old", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestIntegration_RenameCollection_RejectsIdenticalName(t *testing.T) {
	err := newService(t).RenameCollection("srv", "any", "same", "same")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must differ")
}

func TestIntegration_DropCollection_RemovesIt(t *testing.T) {
	db := "coll_drop"
	seedColl(t, db, "goner")

	svc := newService(t)
	require.NoError(t, svc.DropCollection("srv", db, "goner"))

	names, err := svc.GetCollections("srv", db)
	require.NoError(t, err)
	assert.NotContains(t, names, "goner")
}

func TestIntegration_GetStatistics_ReturnsCollStats(t *testing.T) {
	db := "coll_stats"
	seedColl(t, db, "c")

	stats, err := newService(t).GetStatistics("srv", db, "c")
	require.NoError(t, err)

	assert.Contains(t, stats, "count")
	assert.Contains(t, stats, "size")
}

func TestIntegration_GetServerStatistics_ReturnsServerStatus(t *testing.T) {
	stats, err := newService(t).GetServerStatistics("srv")
	require.NoError(t, err)

	assert.Contains(t, stats, "host")
	assert.Contains(t, stats, "version")
}

func TestIntegration_SampleSchema_InfersFields(t *testing.T) {
	db := "coll_sample"
	seedColl(t, db, "c")

	// SampleSchema takes its own ctx rather than using s.ctx.
	schema, err := newService(t).SampleSchema(context.Background(), "srv", db, "c", 100)
	require.NoError(t, err)

	assert.Equal(t, int64(1), schema.TotalCount)
	assert.NotZero(t, schema.SampledCount)
}

func TestIntegration_GetCollections_PropagatesProviderError(t *testing.T) {
	svc := NewCollectionsService(slog.Default(), stubProvider{err: assert.AnError})
	svc.Init(context.Background())

	_, err := svc.GetCollections("srv", "any")
	assert.ErrorIs(t, err, assert.AnError)
}
