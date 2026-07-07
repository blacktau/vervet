//go:build integration

package schema

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var testClient *mongo.Client

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

func TestIntegration_Sample_Aggregate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := "test_sample_aggregate"
	defer testClient.Database(db).Drop(ctx)

	coll := testClient.Database(db).Collection("c1")
	docs := []any{
		bson.M{"name": "a", "age": 30},
		bson.M{"name": "b", "age": 31},
		bson.M{"name": "c", "age": "not a number"},
	}
	if _, err := coll.InsertMany(ctx, docs); err != nil {
		t.Fatalf("insert: %v", err)
	}

	schema, err := Sample(ctx, testClient, db, "c1", 100)
	if err != nil {
		t.Fatalf("Sample: %v", err)
	}
	if schema.SampledCount == 0 {
		t.Fatal("SampledCount = 0")
	}
	if schema.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", schema.TotalCount)
	}
}

func TestIntegration_Sample_OnView(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := "test_sample_view"
	defer testClient.Database(db).Drop(ctx)

	database := testClient.Database(db)
	if _, err := database.Collection("base").InsertOne(ctx, bson.M{"x": 1}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	err := database.RunCommand(ctx, bson.D{
		{Key: "create", Value: "view1"},
		{Key: "viewOn", Value: "base"},
		{Key: "pipeline", Value: bson.A{}},
	}).Err()
	if err != nil {
		t.Fatalf("create view: %v", err)
	}

	schema, err := Sample(ctx, testClient, db, "view1", 100)
	if err != nil {
		t.Fatalf("Sample: %v", err)
	}
	if schema.SampledCount == 0 {
		t.Fatal("expected docs from view")
	}
}

func TestIntegration_Sample_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := "test_sample_cancel"
	defer testClient.Database(db).Drop(ctx)

	coll := testClient.Database(db).Collection("c2")
	bulk := make([]any, 1000)
	for i := range bulk {
		bulk[i] = bson.M{"i": i}
	}
	if _, err := coll.InsertMany(ctx, bulk); err != nil {
		t.Fatalf("insert: %v", err)
	}

	cancelCtx, cancelFn := context.WithCancel(ctx)
	cancelFn()

	_, err := Sample(cancelCtx, testClient, db, "c2", 1000)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}
