//go:build integration

package databases

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

// stubProvider satisfies ClientProvider, handing every caller the container client.
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

func newService(t *testing.T) *DatabasesService {
	t.Helper()
	svc := NewDatabasesService(slog.Default(), stubProvider{client: testClient})
	svc.Init(context.Background())
	return svc
}

func seed(t *testing.T, dbName string) {
	t.Helper()
	ctx := context.Background()
	_, err := testClient.Database(dbName).Collection("seed").InsertOne(ctx, bson.M{"x": 1})
	require.NoError(t, err)
	t.Cleanup(func() { testClient.Database(dbName).Drop(ctx) })
}

func TestIntegration_GetDatabases_ListsAndSorts(t *testing.T) {
	seed(t, "zz_db_list")
	seed(t, "aa_db_list")

	names, err := newService(t).GetDatabases("srv")
	require.NoError(t, err)

	assert.Contains(t, names, "aa_db_list")
	assert.Contains(t, names, "zz_db_list")
	assert.True(t, slices.IsSorted(names), "GetDatabases must return sorted names, got %v", names)
}

func TestIntegration_GetDatabases_PropagatesProviderError(t *testing.T) {
	svc := NewDatabasesService(slog.Default(), stubProvider{err: assert.AnError})
	svc.Init(context.Background())

	_, err := svc.GetDatabases("srv")
	assert.ErrorIs(t, err, assert.AnError)
}

func TestIntegration_GetDatabaseStatistics_ReturnsDbStats(t *testing.T) {
	seed(t, "db_stats_test")

	stats, err := newService(t).GetDatabaseStatistics("srv", "db_stats_test")
	require.NoError(t, err)

	assert.Equal(t, "db_stats_test", stats["db"])
	assert.Contains(t, stats, "collections")
	assert.Contains(t, stats, "dataSize")
}

func TestIntegration_DropDatabase_RemovesIt(t *testing.T) {
	ctx := context.Background()
	seed(t, "db_to_drop")

	svc := newService(t)
	require.NoError(t, svc.DropDatabase("srv", "db_to_drop"))

	names, err := testClient.ListDatabaseNames(ctx, bson.D{})
	require.NoError(t, err)
	assert.NotContains(t, names, "db_to_drop")
}

func TestIntegration_DropDatabase_NonExistentIsNoError(t *testing.T) {
	// MongoDB treats dropping an absent database as success.
	assert.NoError(t, newService(t).DropDatabase("srv", "never_existed_xyz"))
}
