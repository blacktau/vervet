// Package collections manages MongoDB collection operations
package collections

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"vervet/internal/models"
	"vervet/internal/schema"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
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
	// Views are listed under their own tree folder via GetViews; without this
	// filter they would appear under Collections as well.
	names, err := db.ListCollectionNames(ctx, bson.D{{Key: "type", Value: bson.D{{Key: "$ne", Value: "view"}}}})
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

// inventoryWorkers bounds how many databases are listed concurrently when
// building the namespace inventory, so a server with many databases does not
// open an unbounded number of simultaneous operations.
const inventoryWorkers = 8

// GetNamespaceInventory lists every database on the server along with its
// collections and views. It is used to build the data browser's search index
// up front, so find works on namespaces the user has never expanded.
//
// A database that fails to list yields an empty entry rather than failing the
// whole call: one restricted database must not cost the user the entire index.
func (s *CollectionsService) GetNamespaceInventory(serverID string) (models.NamespaceInventory, error) {
	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return models.NamespaceInventory{}, err
	}

	ctx, cancel := context.WithTimeout(s.ctx, operationTimeout)
	defer cancel()

	dbNames, err := client.ListDatabaseNames(ctx, bson.D{})
	if err != nil {
		return models.NamespaceInventory{}, err
	}
	slices.Sort(dbNames)

	databases := make([]models.DatabaseNamespaces, len(dbNames))
	sem := make(chan struct{}, inventoryWorkers)
	var wg sync.WaitGroup

	for i, dbName := range dbNames {
		wg.Add(1)
		go func(i int, dbName string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			collections, views, err := s.listNamespaces(ctx, client, dbName)
			if err != nil {
				s.log.Warn("list namespaces failed", "database", dbName, "error", err)
				databases[i] = models.DatabaseNamespaces{
					Name:        dbName,
					Collections: []string{},
					Views:       []string{},
				}
				return
			}

			databases[i] = models.DatabaseNamespaces{
				Name:        dbName,
				Collections: collections,
				Views:       views,
			}
		}(i, dbName)
	}
	wg.Wait()

	return models.NamespaceInventory{ServerID: serverID, Databases: databases}, nil
}

// listNamespaces splits one database's namespaces by type in a single round
// trip. ListCollections returns the type field, so this costs half what a
// separate GetCollections plus GetViews pair would.
func (s *CollectionsService) listNamespaces(
	ctx context.Context,
	client *mongo.Client,
	dbName string,
) ([]string, []string, error) {
	cursor, err := client.Database(dbName).ListCollections(ctx, bson.D{})
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)

	collections := []string{}
	views := []string{}

	for cursor.Next(ctx) {
		var entry struct {
			Name string `bson:"name"`
			Type string `bson:"type"`
		}
		if err := cursor.Decode(&entry); err != nil {
			return nil, nil, err
		}
		if entry.Type == "view" {
			views = append(views, entry.Name)
			continue
		}
		collections = append(collections, entry.Name)
	}

	if err := cursor.Err(); err != nil {
		return nil, nil, err
	}

	slices.Sort(collections)
	slices.Sort(views)
	return collections, views, nil
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

func (s *CollectionsService) SampleSchema(ctx context.Context, serverID, dbName, collName string, size int) (models.CollectionSchema, error) {
	client, err := s.clients.GetClient(serverID)
	if err != nil {
		return models.CollectionSchema{}, err
	}
	return schema.Sample(ctx, client, dbName, collName, size)
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
