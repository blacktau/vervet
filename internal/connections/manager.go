// Package connections contains code to manage active connections to mongo instances
package connections

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
	"vervet/internal/connectionStrings"
	"vervet/internal/logging"
	"vervet/internal/models"
	"vervet/internal/queryengine"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.mongodb.org/mongo-driver/event"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	ConnectedEvent    = "connection-connected"
	DisconnectedEvent = "connection-disconnected"
)

type ConnectionManager struct {
	mu                sync.RWMutex
	ctx               context.Context
	store             connectionStrings.Store
	serverProvider    ServerProvider
	log               *slog.Logger
	activeConnections map[string]activeConnection
}

type ServerProvider interface {
	GetServer(id string) (*models.RegisteredServer, error)
}

func NewManager(log *slog.Logger, store connectionStrings.Store, provider ServerProvider) *ConnectionManager {
	log = log.With(slog.String(logging.SourceKey, "ConnectionManager"))
	return &ConnectionManager{
		activeConnections: make(map[string]activeConnection),
		mu:                sync.RWMutex{},
		log:               log,
		store:             store,
		serverProvider:    provider,
	}
}

func (cm *ConnectionManager) Init(ctx context.Context) error {
	cm.log.Debug("Initializing Connection ConnectionManager")
	cm.ctx = ctx
	return nil
}

// Connect establishes a connection to a MongoDB database using a securely stored URI.
// This method is exposed to Wails.
func (cm *ConnectionManager) Connect(serverID string) (models.Connection, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.log.Debug("Connecting to Mongo Instance", slog.String("serverID", serverID))

	if _, ok := cm.activeConnections[serverID]; ok {
		cm.log.Warn("already connected to Mongo Instance", slog.String("serverID", serverID))
		return models.Connection{}, fmt.Errorf("already connected to this Mongo Instance")
	}

	server, err := cm.serverProvider.GetServer(serverID)
	if err != nil {
		cm.log.Error("Error retrieving server", slog.String("serverID", serverID))
		return models.Connection{}, fmt.Errorf("error retrieving server: %w", err)
	}

	uri, err := cm.store.GetRegisteredServerURI(serverID)
	if err != nil {
		cm.log.Error("Error retrieving connection URI", slog.String("serverID", serverID))
		return models.Connection{}, fmt.Errorf("error retrieving connection URI: %w", err)
	}

	monitor := &event.CommandMonitor{
		Succeeded: func(ctx context.Context, evt *event.CommandSucceededEvent) {
			if evt.CommandName == "hello" || evt.CommandName == "isMaster" {
				cm.log.Info(
					"Connected to MongoDB",
					slog.String("ConnectionID", evt.ConnectionID),
					slog.Any("Reply", evt.Reply))
			}
		},
	}

	clientOptions := options.Client().
		ApplyURI(uri).
		SetMonitor(monitor)
	ctx, cancel := context.WithTimeout(cm.ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		cm.log.Error("Failed to connect to MongoDB", slog.String("serverID", serverID), slog.Any("error", err))
		return models.Connection{}, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		cm.log.Error("Ping failed for server", slog.String("serverID", serverID), slog.Any("error", err))
		err2 := client.Disconnect(cm.ctx)
		if err2 != nil {
			cm.log.Error("Error disconnecting from mongo server", slog.String("serverID", serverID), slog.Any("error", err2))
		}
		return models.Connection{}, fmt.Errorf("ping failed, connection invalid: %w", err)
	}

	ac := newActiveConnection(serverID, server.Name)
	ac.client = client
	// ac.ctx = ctx
	cm.activeConnections[serverID] = ac

	cm.log.Info("Successfully connected to MongoDB", slog.String("serverID", serverID))
	runtime.EventsEmit(cm.ctx, ConnectedEvent, serverID)

	connection := models.Connection{
		ServerID: serverID,
		Name:     server.Name,
	}

	return connection, nil
}

func (cm *ConnectionManager) TestConnection(uri string) (bool, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	clientOptions := options.Client().ApplyURI(uri)
	ctx, cancel := context.WithTimeout(cm.ctx, 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		scrubbed := cleanConnectionString(uri)
		cm.log.Error("Failed to connect to MongoDB:", slog.String("uri", scrubbed), slog.Any("error", err))
		return false, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		err2 := client.Disconnect(cm.ctx)
		scrubbed := cleanConnectionString(uri)
		if err2 != nil {
			cm.log.Error("Error disconnecting from mongo server", slog.String("uri", scrubbed), slog.Any("error", err2))
		}
		cm.log.Error("Ping failed:", slog.String("uri", scrubbed), slog.Any("error", err))
		return false, fmt.Errorf("failed to connection to database: %w", err)
	}

	if _, err = client.ListDatabases(ctx, bson.D{}, nil); err != nil {
		scrubbed := cleanConnectionString(uri)
		err2 := client.Disconnect(cm.ctx)
		if err2 != nil {
			cm.log.Error("Error disconnecting from mongo server", slog.String("uri", scrubbed), slog.Any("error", err2))
		}

		cm.log.Error("Failed to retrieve list of databases", slog.Any("error", err))
		return false, fmt.Errorf("failed to retrieve list of databases: %w", err)
	}

	err = client.Disconnect(ctx)
	if err != nil {
		scrubbed := cleanConnectionString(uri)
		cm.log.Error("Error disconnecting from mongo server", slog.String("uri", scrubbed), slog.Any("error", err))
	}

	return true, nil
}

func (cm *ConnectionManager) Disconnect(serverID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if connection, ok := cm.activeConnections[serverID]; ok {
		err := connection.Disconnect(cm.ctx)
		if err != nil {
			cm.log.Error("Error disconnecting from mongo server", slog.String("serverID", serverID), slog.Any("error", err))
			return fmt.Errorf("error disconnecting: %w", err)
		}

		delete(cm.activeConnections, serverID)
		runtime.EventsEmit(cm.ctx, DisconnectedEvent, serverID)
		cm.log.Info("Disconnected from mongo server", slog.String("serverID", serverID))
		return nil
	}

	cm.log.Warn("Connection not found or not active", slog.String("serverID", serverID))
	return fmt.Errorf("connection not found or not active")
}

// DisconnectAll disconnects all the active connections
// this is exposed to wails
func (cm *ConnectionManager) DisconnectAll() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for id, connection := range cm.activeConnections {
		err := connection.Disconnect(cm.ctx)
		if err != nil {
			cm.log.Error("Error when disconnecting from mongo", slog.String("serverID", connection.serverID), slog.Any("error", err))
		}

		runtime.EventsEmit(cm.ctx, DisconnectedEvent, id)

		delete(cm.activeConnections, id)
	}

	cm.log.Info("Disconnected from all mongo servers")
	return nil
}

func (cm *ConnectionManager) GetConnections() []models.Connection {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ids := make([]models.Connection, 0, len(cm.activeConnections))
	for id := range cm.activeConnections {
		connection := cm.activeConnections[id]
		ids = append(ids, models.Connection{
			ServerID: connection.serverID,
			Name:     connection.name,
		})
	}
	return ids
}

func (cm *ConnectionManager) getClient(serverID string) (*activeConnection, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	connection, ok := cm.activeConnections[serverID]
	if !ok {
		cm.log.Warn("No active connection to mongo instance", slog.String("serverID", serverID))
		return nil, fmt.Errorf("no active connection to mongo instance for ID: %v", serverID)
	}

	return &connection, nil
}

func (cm *ConnectionManager) GetDatabases(serverID string) ([]string, error) {
	cm.log.Debug("Getting databases for mongo instance", slog.String("serverID", serverID))
	connection, err := cm.getClient(serverID)
	if err != nil {
		cm.log.Error("Failed to get client", slog.String("serverID", serverID), slog.Any("error", err))
		return nil, err
	}

	names, err := connection.client.ListDatabaseNames(cm.ctx, bson.D{})
	if err != nil {
		cm.log.Error("Failed to list databases", slog.String("serverID", serverID), slog.Any("error", err))
		return nil, err
	}
	slices.Sort(names)
	cm.log.Debug("Got databases", slog.String("serverID", serverID), slog.Any("databases", names))
	return names, nil
}

func (cm *ConnectionManager) GetCollections(serverID string, dbName string) ([]string, error) {
	cm.log.Debug("Getting collections for mongo instance", slog.String("serverID", serverID), slog.String("dbName", dbName))
	connection, err := cm.getClient(serverID)
	if err != nil {
		return nil, err
	}

	db := connection.client.Database(dbName)
	names, err := db.ListCollectionNames(cm.ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	slices.Sort(names)
	return names, nil
}

func (cm *ConnectionManager) GetViews(serverID string, dbName string) ([]string, error) {
	cm.log.Debug("Getting views for mongo instance", slog.String("serverID", serverID), slog.String("dbName", dbName))
	connection, err := cm.getClient(serverID)
	if err != nil {
		return nil, err
	}

	db := connection.client.Database(dbName)
	filter := bson.D{{Key: "type", Value: "view"}}
	names, err := db.ListCollectionNames(cm.ctx, filter)
	if err != nil {
		return nil, err
	}
	slices.Sort(names)
	return names, nil
}

func (cm *ConnectionManager) GetCollectionSchema(serverID, dbName, collName string) (models.CollectionSchema, error) {
	connection, err := cm.getClient(serverID)
	if err != nil {
		return models.CollectionSchema{}, err
	}
	return queryengine.SampleSchema(cm.ctx, connection.client, dbName, collName)
}

func (cm *ConnectionManager) CreateCollection(serverID string, dbName string, collectionName string) error {
	cm.log.Debug("Creating collection",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("collectionName", collectionName))

	connection, err := cm.getClient(serverID)
	if err != nil {
		return err
	}

	db := connection.client.Database(dbName)
	err = db.CreateCollection(cm.ctx, collectionName)
	if err != nil {
		cm.log.Error("Failed to create collection",
			slog.String("serverID", serverID),
			slog.String("dbName", dbName),
			slog.String("collectionName", collectionName),
			slog.Any("error", err))
		return fmt.Errorf("failed to create collection: %w", err)
	}

	cm.log.Info("Created collection",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("collectionName", collectionName))
	return nil
}

func (cm *ConnectionManager) GetIndexes(serverID string, dbName string, collectionName string) ([]models.Index, error) {
	cm.log.Debug("Getting indexes",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("collectionName", collectionName))

	connection, err := cm.getClient(serverID)
	if err != nil {
		return nil, err
	}

	collection := connection.client.Database(dbName).Collection(collectionName)
	cursor, err := collection.Indexes().List(cm.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer cursor.Close(cm.ctx)

	type rawIndex struct {
		Name               string `bson:"name"`
		Key                bson.D `bson:"key"`
		Unique             bool   `bson:"unique,omitempty"`
		Sparse             bool   `bson:"sparse,omitempty"`
		ExpireAfterSeconds *int32 `bson:"expireAfterSeconds,omitempty"`
	}

	var results []rawIndex
	if err := cursor.All(cm.ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode indexes: %w", err)
	}

	var indexes []models.Index
	for _, raw := range results {
		idx := models.Index{
			Name:   raw.Name,
			Unique: raw.Unique,
			Sparse: raw.Sparse,
			TTL:    raw.ExpireAfterSeconds,
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

func (cm *ConnectionManager) CreateIndex(serverID string, dbName string, collectionName string, request models.CreateIndexRequest) error {
	cm.log.Debug("Creating index",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("collectionName", collectionName))

	connection, err := cm.getClient(serverID)
	if err != nil {
		return err
	}

	keys := bson.D{}
	for _, k := range request.Keys {
		keys = append(keys, bson.E{Key: k.Field, Value: k.Direction})
	}

	indexOpts := options.Index()
	if request.Name != "" {
		indexOpts.SetName(request.Name)
	}
	if request.Unique {
		indexOpts.SetUnique(true)
	}
	if request.Sparse {
		indexOpts.SetSparse(true)
	}
	if request.TTL != nil {
		indexOpts.SetExpireAfterSeconds(*request.TTL)
	}

	model := mongo.IndexModel{
		Keys:    keys,
		Options: indexOpts,
	}

	collection := connection.client.Database(dbName).Collection(collectionName)
	name, err := collection.Indexes().CreateOne(cm.ctx, model)
	if err != nil {
		cm.log.Error("Failed to create index",
			slog.String("serverID", serverID),
			slog.String("collectionName", collectionName),
			slog.Any("error", err))
		return fmt.Errorf("failed to create index: %w", err)
	}

	cm.log.Info("Created index",
		slog.String("serverID", serverID),
		slog.String("collectionName", collectionName),
		slog.String("indexName", name))
	return nil
}

func (cm *ConnectionManager) DropIndex(serverID string, dbName string, collectionName string, indexName string) error {
	cm.log.Debug("Dropping index",
		slog.String("serverID", serverID),
		slog.String("dbName", dbName),
		slog.String("collectionName", collectionName),
		slog.String("indexName", indexName))

	connection, err := cm.getClient(serverID)
	if err != nil {
		return err
	}

	collection := connection.client.Database(dbName).Collection(collectionName)
	_, err = collection.Indexes().DropOne(cm.ctx, indexName)
	if err != nil {
		cm.log.Error("Failed to drop index",
			slog.String("serverID", serverID),
			slog.String("collectionName", collectionName),
			slog.String("indexName", indexName),
			slog.Any("error", err))
		return fmt.Errorf("failed to drop index: %w", err)
	}

	cm.log.Info("Dropped index",
		slog.String("serverID", serverID),
		slog.String("collectionName", collectionName),
		slog.String("indexName", indexName))
	return nil
}

func cleanConnectionString(uri string) string {
	idx := strings.Index(uri, "@")
	if idx == -1 {
		return uri
	}

	front := uri[0:idx]
	rest := uri[idx+1:]

	idx = strings.LastIndex(front, ":")
	front = front[0:idx]

	return front + ":***" + rest
}
