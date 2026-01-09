// Package connections contains code to manage active connections to mongo instances
package connections

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"vervet/internal/connectionStrings"
	"vervet/internal/logging"
	"vervet/internal/models"

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
	ctx               context.Context
	activeConnections map[string]ActiveConnection
	mu                sync.RWMutex
	log               *slog.Logger
	store             connectionStrings.Store
}

func NewManager(log *slog.Logger, store connectionStrings.Store) *ConnectionManager {
	log = log.With(slog.String(logging.SourceKey, "ConnectionManager"))
	return &ConnectionManager{
		activeConnections: make(map[string]ActiveConnection),
		mu:                sync.RWMutex{},
		log:               log,
		store:             store,
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
		_ = client.Disconnect(cm.ctx)
		return models.Connection{}, fmt.Errorf("ping failed, connection invalid: %w", err)
	}

	activeConnection := newActiveConnection(serverID)
	activeConnection.client = client
	// activeConnection.ctx = ctx
	cm.activeConnections[serverID] = activeConnection

	cm.log.Info("Successfully connected to MongoDB", slog.String("serverID", serverID))
	runtime.EventsEmit(cm.ctx, ConnectedEvent, serverID)

	connection := models.Connection{
		ServerID: serverID,
		Name:     serverID,
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
		cm.log.Error("Failed to connect to MongoDB:", slog.String("uri", uri), slog.Any("error", err))
		return false, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		cm.log.Error("Ping failed:", slog.String("uri", uri), slog.Any("error", err))
		return false, fmt.Errorf("failed to connection to database: %w", err)
	}

	if _, err = client.ListDatabases(ctx, bson.D{}, nil); err != nil {
		_ = client.Disconnect(ctx)
		cm.log.Error("Failed to retrieve list of databases", slog.Any("error", err))
		return false, fmt.Errorf("failed to retrieve list of databases: %w", err)
	}

	_ = client.Disconnect(ctx)

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

// GetConnectedClientIDs returns a list od IDs for the currently active connections
// this is exposed to wails
func (cm *ConnectionManager) GetConnectedClientIDs() []string {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ids := make([]string, 0, len(cm.activeConnections))
	for id := range cm.activeConnections {
		ids = append(ids, id)
	}
	return ids
}

func (cm *ConnectionManager) getClient(serverID string) (*ActiveConnection, error) {
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
	connection, err := cm.getClient(serverID)
	if err != nil {
		return nil, err
	}

	return connection.client.ListDatabaseNames(cm.ctx, bson.D{})
}
