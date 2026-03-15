// Package collections manages MongoDB collection operations
package collections

import (
	"context"
	"log/slog"
	"vervet/internal/logging"

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
