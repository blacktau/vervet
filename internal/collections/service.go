// Package collections manages MongoDB collection operations
package collections

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"vervet/internal/logging"
	"vervet/internal/models"
	"vervet/internal/queryengine"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ClientProvider provides access to active MongoDB connections
type ClientProvider interface {
	GetClient(serverID string) (*mongo.Client, error)
}

// CollectionsService handles operations on MongoDB collections
type CollectionsService struct {
	ctx     context.Context
	log     *slog.Logger
	clients ClientProvider
}

func NewCollectionsService(log *slog.Logger, clients ClientProvider) *CollectionsService {
	return &CollectionsService{
		log:     log.With(slog.String(logging.SourceKey, "CollectionsService")),
		clients: clients,
	}
}

func (s *CollectionsService) Init(ctx context.Context) {
	s.ctx = ctx
}

func (s *CollectionsService) GetServerStatistics(serverID string) (map[string]any, error) {
	s.log.Debug("Getting server statistics",
		slog.String("serverID", serverID),
	)

	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return nil, err
	}

	var result bson.M
	err = client.Database("admin").RunCommand(s.ctx, bson.D{
		{Key: "serverStatus", Value: 1},
	}).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *CollectionsService) GetDatabaseStatistics(serverID, dbName string) (map[string]any, error) {
	s.log.Debug("Getting database statistics",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
	)

	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return nil, err
	}

	var result bson.M
	err = client.Database(dbName).RunCommand(s.ctx, bson.D{
		{Key: "dbStats", Value: 1},
	}).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *CollectionsService) GetStatistics(serverID, dbName, collectionName string) (map[string]any, error) {
	s.log.Debug("Getting collection statistics",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("collectionName", collectionName),
	)

	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return nil, err
	}

	var result bson.M
	err = client.Database(dbName).RunCommand(s.ctx, bson.D{
		{Key: "collStats", Value: collectionName},
	}).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *CollectionsService) GetCollections(serverID, dbName string) ([]string, error) {
	s.log.Debug("Getting collections",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName))

	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return nil, err
	}

	db := client.Database(dbName)
	names, err := db.ListCollectionNames(s.ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	slices.Sort(names)
	return names, nil
}

func (s *CollectionsService) GetViews(serverID, dbName string) ([]string, error) {
	s.log.Debug("Getting views",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName))

	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return nil, err
	}

	db := client.Database(dbName)
	filter := bson.D{{Key: "type", Value: "view"}}
	names, err := db.ListCollectionNames(s.ctx, filter)
	if err != nil {
		return nil, err
	}
	slices.Sort(names)
	return names, nil
}

func (s *CollectionsService) CreateCollection(serverID, dbName, collectionName string) error {
	s.log.Debug("Creating collection",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("collectionName", collectionName))

	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return err
	}

	err = client.Database(dbName).CreateCollection(s.ctx, collectionName)
	if err != nil {
		s.log.Error("Failed to create collection",
			slog.String("serverID", serverID),
			slog.String("dbName", dbName),
			slog.String("collectionName", collectionName),
			slog.Any("error", err))
		return fmt.Errorf("failed to create collection: %w", err)
	}

	s.log.Info("Created collection",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("collectionName", collectionName))
	return nil
}

func (s *CollectionsService) GetCollectionSchema(serverID, dbName, collName string) (models.CollectionSchema, error) {
	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return models.CollectionSchema{}, err
	}
	return queryengine.SampleSchema(s.ctx, client, dbName, collName)
}

func (s *CollectionsService) RenameCollection(serverID, dbName, oldName, newName string) error {
	if newName == "" {
		return fmt.Errorf("new collection name cannot be empty")
	}
	if oldName == newName {
		return fmt.Errorf("new name must differ from old name")
	}

	s.log.Debug("Renaming collection",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("oldName", oldName),
		slog.String("newName", newName))

	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return err
	}

	// MongoDB renames via the admin command: renameCollection
	cmd := bson.D{
		{Key: "renameCollection", Value: dbName + "." + oldName},
		{Key: "to", Value: dbName + "." + newName},
	}
	err = client.Database("admin").RunCommand(s.ctx, cmd).Err()
	if err != nil {
		s.log.Error("Failed to rename collection",
			slog.String("serverID", serverID),
			slog.String("dbName", dbName),
			slog.String("oldName", oldName),
			slog.String("newName", newName),
			slog.Any("error", err))
		return fmt.Errorf("failed to rename collection: %w", err)
	}

	s.log.Info("Renamed collection",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("oldName", oldName),
		slog.String("newName", newName))
	return nil
}

func (s *CollectionsService) DropCollection(serverID, dbName, collectionName string) error {
	s.log.Debug("Dropping collection",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("collectionName", collectionName))

	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return err
	}

	err = client.Database(dbName).Collection(collectionName).Drop(s.ctx)
	if err != nil {
		s.log.Error("Failed to drop collection",
			slog.String("serverID", serverID),
			slog.String("dbName", dbName),
			slog.String("collectionName", collectionName),
			slog.Any("error", err))
		return fmt.Errorf("failed to drop collection: %w", err)
	}

	s.log.Info("Dropped collection",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("collectionName", collectionName))
	return nil
}
