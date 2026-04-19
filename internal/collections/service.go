// Package collections manages MongoDB collection operations
package collections

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"vervet/internal/models"
	"vervet/internal/queryengine"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const operationTimeout = 30 * time.Second

// ClientProvider provides access to active MongoDB connections
type ClientProvider interface {
	GetClient(serverID string) (*mongo.Client, error)
}

// CollectionsService handles operations on MongoDB collections
type CollectionsService struct {
	log     *slog.Logger
	ctx     context.Context
	clients ClientProvider
}

func NewCollectionsService(log *slog.Logger, clients ClientProvider) *CollectionsService {
	return &CollectionsService{
		log:     log,
		clients: clients,
	}
}

func (s *CollectionsService) Init(ctx context.Context) {
	s.ctx = ctx
}

func (s *CollectionsService) GetServerStatistics(serverID string) (map[string]any, error) {
	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(s.ctx, operationTimeout)
	defer cancel()

	var result bson.M
	err = client.Database("admin").RunCommand(ctx, bson.D{
		{Key: "serverStatus", Value: 1},
	}).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *CollectionsService) GetStatistics(serverID, dbName, collectionName string) (map[string]any, error) {
	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(s.ctx, operationTimeout)
	defer cancel()

	var result bson.M
	err = client.Database(dbName).RunCommand(ctx, bson.D{
		{Key: "collStats", Value: collectionName},
	}).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *CollectionsService) GetCollections(serverID, dbName string) ([]string, error) {
	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(s.ctx, operationTimeout)
	defer cancel()

	db := client.Database(dbName)
	names, err := db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	slices.Sort(names)
	return names, nil
}

func (s *CollectionsService) GetViews(serverID, dbName string) ([]string, error) {
	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(s.ctx, operationTimeout)
	defer cancel()

	db := client.Database(dbName)
	filter := bson.D{{Key: "type", Value: "view"}}
	names, err := db.ListCollectionNames(ctx, filter)
	if err != nil {
		return nil, err
	}
	slices.Sort(names)
	return names, nil
}

func (s *CollectionsService) CreateCollection(serverID, dbName, collectionName string) error {
	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(s.ctx, operationTimeout)
	defer cancel()

	err = client.Database(dbName).CreateCollection(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

func (s *CollectionsService) GetCollectionSchema(serverID, dbName, collName string) (models.CollectionSchema, error) {
	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return models.CollectionSchema{}, err
	}

	ctx, cancel := context.WithTimeout(s.ctx, operationTimeout)
	defer cancel()

	return queryengine.SampleSchema(ctx, client, dbName, collName)
}

func (s *CollectionsService) RenameCollection(serverID, dbName, oldName, newName string) error {
	if newName == "" {
		return fmt.Errorf("new collection name cannot be empty")
	}
	if oldName == newName {
		return fmt.Errorf("new name must differ from old name")
	}

	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(s.ctx, operationTimeout)
	defer cancel()

	// MongoDB renames via the admin command: renameCollection
	cmd := bson.D{
		{Key: "renameCollection", Value: dbName + "." + oldName},
		{Key: "to", Value: dbName + "." + newName},
	}
	err = client.Database("admin").RunCommand(ctx, cmd).Err()
	if err != nil {
		return fmt.Errorf("failed to rename collection: %w", err)
	}

	return nil
}

func (s *CollectionsService) DropCollection(serverID, dbName, collectionName string) error {
	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(s.ctx, operationTimeout)
	defer cancel()

	err = client.Database(dbName).Collection(collectionName).Drop(ctx)
	if err != nil {
		return fmt.Errorf("failed to drop collection: %w", err)
	}

	return nil
}
