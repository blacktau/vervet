// Package indexes manages MongoDB index operations
package indexes

import (
	"context"
	"fmt"
	"log/slog"
	"vervet/internal/logging"
	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ClientProvider provides access to active MongoDB connections
type ClientProvider interface {
	GetClient(serverID string) (*mongo.Client, error)
}

type rawIndex struct {
	Name               string `bson:"name"`
	Key                bson.D `bson:"key"`
	Unique             bool   `bson:"unique,omitempty"`
	Sparse             bool   `bson:"sparse,omitempty"`
	ExpireAfterSeconds *int32 `bson:"expireAfterSeconds,omitempty"`
}

type indexStat struct {
	Name     string `bson:"name"`
	Accesses struct {
		Ops int64 `bson:"ops"`
	} `bson:"accesses"`
}

// IndexService handles CRUD operations for MongoDB collection indexes
type IndexService struct {
	ctx     context.Context
	log     *slog.Logger
	clients ClientProvider
}

func NewIndexService(log *slog.Logger, clients ClientProvider) *IndexService {
	return &IndexService{
		log:     log.With(slog.String(logging.SourceKey, "IndexService")),
		clients: clients,
	}
}

func (s *IndexService) Init(ctx context.Context) {
	s.ctx = ctx
}

func (s *IndexService) GetIndexes(serverID, dbName, collectionName string) ([]models.Index, error) {
	s.log.Debug("Getting indexes",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("collectionName", collectionName))

	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return nil, err
	}

	collection := client.Database(dbName).Collection(collectionName)
	cursor, err := collection.Indexes().List(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer cursor.Close(s.ctx)

	var results []rawIndex
	if err := cursor.All(s.ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode indexes: %w", err)
	}

	// Fetch index usage stats via $indexStats aggregation
	usageMap := make(map[string]int64)
	statsCursor, err := collection.Aggregate(s.ctx, mongo.Pipeline{
		{{Key: "$indexStats", Value: bson.D{}}},
	})
	if err == nil {
		defer statsCursor.Close(s.ctx)
		var stats []indexStat
		if err := statsCursor.All(s.ctx, &stats); err == nil {
			for _, s := range stats {
				usageMap[s.Name] = s.Accesses.Ops
			}
		}
	}

	// Fetch index sizes via collStats command
	sizeMap := make(map[string]int64)
	var collStatsResult bson.M
	err = client.Database(dbName).RunCommand(s.ctx, bson.D{
		{Key: "collStats", Value: collectionName},
	}).Decode(&collStatsResult)
	if err == nil {
		if indexSizes, ok := collStatsResult["indexSizes"].(bson.M); ok {
			for name, size := range indexSizes {
				switch v := size.(type) {
				case int32:
					sizeMap[name] = int64(v)
				case int64:
					sizeMap[name] = v
				case float64:
					sizeMap[name] = int64(v)
				}
			}
		}
	}

	var indexes []models.Index
	for _, raw := range results {
		idx := models.Index{
			Name:   raw.Name,
			Unique: raw.Unique,
			Sparse: raw.Sparse,
			TTL:    raw.ExpireAfterSeconds,
			Size:   sizeMap[raw.Name],
			Usage:  usageMap[raw.Name],
		}

		for _, elem := range raw.Key {
			idx.Keys = append(idx.Keys, models.IndexKeyField{
				Field:     elem.Key,
				Direction: elem.Value,
			})
		}

		indexes = append(indexes, idx)
	}

	return indexes, nil
}

func (s *IndexService) CreateIndex(serverID, dbName, collectionName string, request models.CreateIndexRequest) error {
	s.log.Debug("Creating index",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("collectionName", collectionName))

	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return err
	}

	model := s.buildIndexModel(request.Keys, request.Name, request.Unique, request.Sparse, request.TTL)

	collection := client.Database(dbName).Collection(collectionName)
	name, err := collection.Indexes().CreateOne(s.ctx, model)
	if err != nil {
		s.log.Error("Failed to create index",
			slog.String("serverID", serverID),
			slog.String("collectionName", collectionName),
			slog.Any("error", err))
		return fmt.Errorf("failed to create index: %w", err)
	}

	s.log.Info("Created index",
		slog.String("serverID", serverID),
		slog.String("collectionName", collectionName),
		slog.String("indexName", name))
	return nil
}

func (s *IndexService) EditIndex(serverID, dbName, collectionName string, request models.EditIndexRequest) error {
	s.log.Debug("Editing index",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("collectionName", collectionName),
		slog.String("oldName", request.OldName))

	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return err
	}

	collection := client.Database(dbName).Collection(collectionName)
	newModel := s.buildIndexModel(request.Keys, request.Name, request.Unique, request.Sparse, request.TTL)

	sameNameEdit := request.Name == "" || request.Name == request.OldName

	if sameNameEdit {
		// Same name — must drop first to avoid name conflict
		oldSpec, captureErr := s.captureIndex(collection, request.OldName)

		_, err = collection.Indexes().DropOne(s.ctx, request.OldName)
		if err != nil {
			return fmt.Errorf("failed to drop old index: %w", err)
		}

		_, err = collection.Indexes().CreateOne(s.ctx, newModel)
		if err != nil {
			// Attempt to restore the original index
			if captureErr == nil && oldSpec != nil {
				if _, restoreErr := collection.Indexes().CreateOne(s.ctx, *oldSpec); restoreErr != nil {
					s.log.Error("Failed to restore original index after edit failure",
						slog.String("indexName", request.OldName),
						slog.Any("error", restoreErr))
				}
			}
			return fmt.Errorf("failed to create replacement index: %w", err)
		}
	} else {
		// Different name — create first, then drop old
		_, err = collection.Indexes().CreateOne(s.ctx, newModel)
		if err != nil {
			return fmt.Errorf("failed to create new index: %w", err)
		}

		_, err = collection.Indexes().DropOne(s.ctx, request.OldName)
		if err != nil {
			s.log.Error("Created new index but failed to drop old one",
				slog.String("oldName", request.OldName),
				slog.Any("error", err))
			return fmt.Errorf("new index created but failed to drop old index %q: %w", request.OldName, err)
		}
	}

	s.log.Info("Edited index",
		slog.String("serverID", serverID),
		slog.String("collectionName", collectionName),
		slog.String("oldName", request.OldName),
		slog.String("newName", request.Name))
	return nil
}

func (s *IndexService) DropIndex(serverID, dbName, collectionName, indexName string) error {
	s.log.Debug("Dropping index",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("collectionName", collectionName),
		slog.String("indexName", indexName))

	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return err
	}

	collection := client.Database(dbName).Collection(collectionName)
	_, err = collection.Indexes().DropOne(s.ctx, indexName)
	if err != nil {
		s.log.Error("Failed to drop index",
			slog.String("serverID", serverID),
			slog.String("collectionName", collectionName),
			slog.String("indexName", indexName),
			slog.Any("error", err))
		return fmt.Errorf("failed to drop index: %w", err)
	}

	s.log.Info("Dropped index",
		slog.String("serverID", serverID),
		slog.String("collectionName", collectionName),
		slog.String("indexName", indexName))
	return nil
}

func (s *IndexService) buildIndexModel(keys []models.IndexKeyField, name string, unique, sparse bool, ttl *int32) mongo.IndexModel {
	bsonKeys := bson.D{}
	for _, k := range keys {
		dir := k.Direction
		// JSON decodes numbers as float64; MongoDB requires int for direction values
		if f, ok := dir.(float64); ok {
			dir = int(f)
		}
		bsonKeys = append(bsonKeys, bson.E{Key: k.Field, Value: dir})
	}

	indexOpts := options.Index()
	if name != "" {
		indexOpts.SetName(name)
	}
	if unique {
		indexOpts.SetUnique(true)
	}
	if sparse {
		indexOpts.SetSparse(true)
	}
	if ttl != nil {
		indexOpts.SetExpireAfterSeconds(*ttl)
	}

	return mongo.IndexModel{
		Keys:    bsonKeys,
		Options: indexOpts,
	}
}

func (s *IndexService) captureIndex(collection *mongo.Collection, indexName string) (*mongo.IndexModel, error) {
	cursor, err := collection.Indexes().List(s.ctx)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	var results []rawIndex
	if err := cursor.All(s.ctx, &results); err != nil {
		return nil, err
	}

	for _, raw := range results {
		if raw.Name != indexName {
			continue
		}
		opts := options.Index().SetName(raw.Name)
		if raw.Unique {
			opts.SetUnique(true)
		}
		if raw.Sparse {
			opts.SetSparse(true)
		}
		if raw.ExpireAfterSeconds != nil {
			opts.SetExpireAfterSeconds(*raw.ExpireAfterSeconds)
		}
		return &mongo.IndexModel{Keys: raw.Key, Options: opts}, nil
	}

	return nil, fmt.Errorf("index %q not found", indexName)
}
