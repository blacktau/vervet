package connections

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type ActiveConnection struct {
	serverID string
	client   *mongo.Client
}

func newActiveConnection(serverID string) ActiveConnection {
	return ActiveConnection{
		serverID: serverID,
	}
}

func (ac *ActiveConnection) Disconnect(ctx context.Context) error {
	err := ac.client.Disconnect(ctx)
	if err != nil {
		return fmt.Errorf("error during mongo disconnection: %w", err)
	}

	ac.client = nil
	return nil
}

func (ac *ActiveConnection) Query(ctx context.Context, dbName string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	db := ac.client.Database(dbName)
	collections, err := db.ListCollectionNames(ctx, struct{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections in database %v: %w", dbName, err)
	}
	return collections, err
}
