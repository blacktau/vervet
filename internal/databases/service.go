// Package databases manages MongoDB database-level operations
package databases

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"vervet/internal/logging"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ClientProvider provides access to active MongoDB connections
type ClientProvider interface {
	GetClient(serverID string) (*mongo.Client, error)
}

// DatabasesService handles database-level operations
type DatabasesService struct {
	ctx     context.Context
	log     *slog.Logger
	clients ClientProvider
}

func NewDatabasesService(log *slog.Logger, clients ClientProvider) *DatabasesService {
	return &DatabasesService{
		log:     log.With(slog.String(logging.SourceKey, "DatabasesService")),
		clients: clients,
	}
}

func (s *DatabasesService) Init(ctx context.Context) {
	s.ctx = ctx
}

func (s *DatabasesService) GetDatabases(serverID string) ([]string, error) {
	s.log.Debug("Getting databases for mongo instance", slog.String("serverID", serverID))
	client, err := s.clients.GetClient(serverID)
	if err != nil {
		s.log.Error("Failed to get client", slog.String("serverID", serverID), slog.Any("error", err))
		return nil, err
	}

	names, err := client.ListDatabaseNames(s.ctx, bson.D{})
	if err != nil {
		s.log.Error("Failed to list databases", slog.String("serverID", serverID), slog.Any("error", err))
		return nil, err
	}
	slices.Sort(names)
	s.log.Debug("Got databases", slog.String("serverID", serverID), slog.Any("databases", names))
	return names, nil
}

func (s *DatabasesService) GetDatabaseStatistics(serverID, dbName string) (map[string]any, error) {
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

func (s *DatabasesService) DropDatabase(serverID, dbName string) error {
	s.log.Debug("Dropping database",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName))

	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return err
	}

	err = client.Database(dbName).Drop(s.ctx)
	if err != nil {
		s.log.Error("Failed to drop database",
			slog.String("serverID", serverID),
			slog.String("dbName", dbName),
			slog.Any("error", err))
		return fmt.Errorf("failed to drop database: %w", err)
	}

	s.log.Info("Dropped database",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName))
	return nil
}
