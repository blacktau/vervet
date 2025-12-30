// Package connections contains code to manage active connections to mongo instances
package connections

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"vervet/internal/connectionStrings"

	"github.com/wailsapp/wails/v2/pkg/logger"
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

type Manager interface {
	Init(ctx context.Context) error
	Connect(serverID string) (Connection, error)
	TestConnection(uri string) (bool, error)
	Disconnect(serverID string) error
	DisconnectAll() error
	GetConnectedClientIDs() []string
	GetClient(serverID string) (*ActiveConnection, error)
	GetDatabases(serverID string) ([]string, error)
}

type connectionManager struct {
	ctx               context.Context
	activeConnections map[string]ActiveConnection
	mu                sync.RWMutex
	log               logger.Logger
	store             connectionStrings.Store
}

type Connection struct {
	serverID string
	name     string
}

func NewManager(log logger.Logger, store connectionStrings.Store) Manager {
	return &connectionManager{
		activeConnections: make(map[string]ActiveConnection),
		mu:                sync.RWMutex{},
		log:               log,
		store:             store,
	}
}

func (cm *connectionManager) Init(ctx context.Context) error {
	cm.ctx = ctx
	return nil
}

// Connect establishes a connection to a MongoDB database using a securely stored URI.
// This method is exposed to Wails.
func (cm *connectionManager) Connect(serverID string) (Connection, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, ok := cm.activeConnections[serverID]; ok {
		return Connection{}, fmt.Errorf("already connected to this Mongo Instance")
	}

	uri, err := cm.store.GetRegisteredServerURI(serverID)
	if err != nil {
		return Connection{}, fmt.Errorf("error retrieving connection URI: %w", err)
	}

	monitor := &event.CommandMonitor{
		Succeeded: func(ctx context.Context, evt *event.CommandSucceededEvent) {
			if evt.CommandName == "hello" || evt.CommandName == "isMaster" {
				cm.log.Info(fmt.Sprintf("Connected to MongoDB: %s", evt.ConnectionID))
				cm.log.Info(fmt.Sprintf("Server Response: %v", evt.Reply))
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
		log.Printf("Failed to connect to MongoDB: %v", err)
		return Connection{}, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(cm.ctx)
		return Connection{}, fmt.Errorf("ping failed, connection invalid: %w", err)
	}

	activeConnection := newActiveConnection(serverID)
	activeConnection.client = client
	// activeConnection.ctx = ctx
	cm.activeConnections[serverID] = activeConnection

	cm.log.Info(fmt.Sprintf("Successfully connected to MongoDB via ID: %d", serverID))
	runtime.EventsEmit(cm.ctx, ConnectedEvent, serverID)

	connection := Connection{
		serverID: serverID,
		name:     serverID,
	}

	return connection, nil
}

func (cm *connectionManager) TestConnection(uri string) (bool, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	clientOptions := options.Client().ApplyURI(uri)
	ctx, cancel := context.WithTimeout(cm.ctx, 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Printf("Failed to connect to MongoDB: %v", err)
		return false, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return false, fmt.Errorf("failed to connection to database: %w", err)
	}

	if _, err = client.ListDatabases(ctx, bson.D{}, nil); err != nil {
		_ = client.Disconnect(ctx)
		return false, fmt.Errorf("failed to retrive list of databases: %w", err)
	}

	_ = client.Disconnect(ctx)

	return true, nil
}

// Disconnect closes the active MongoDB connection.
// This method is exposed to Wails.
func (cm *connectionManager) Disconnect(serverID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if connection, ok := cm.activeConnections[serverID]; ok {
		err := connection.Disconnect(cm.ctx)
		if err != nil {
			log.Printf("Error while disconnecting from mongo for serverID %d: %v", serverID, err)
			return fmt.Errorf("error disconnecting: %w", err)
		}

		delete(cm.activeConnections, serverID)
		runtime.EventsEmit(cm.ctx, DisconnectedEvent, serverID)
		log.Printf("Disconnected from mongo for serverID: %v", serverID)
		return nil
	}

	return fmt.Errorf("connection not found or not active")
}

// DisconnectAll disconnects all the active connections
// this is exposed to wails
func (cm *connectionManager) DisconnectAll() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for id, connection := range cm.activeConnections {
		err := connection.Disconnect(cm.ctx)
		if err != nil {
			log.Printf("Error when disconnecting connection with ID: %v", err)
		}

		runtime.EventsEmit(cm.ctx, DisconnectedEvent, id)

		delete(cm.activeConnections, id)
	}
	log.Print("All active mongo DB connections disconnected")
	return nil
}

// GetConnectedClientIDs returns a list od IDs for the currently active connections
// this is exposed to wails
func (cm *connectionManager) GetConnectedClientIDs() []string {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ids := make([]string, 0, len(cm.activeConnections))
	for id := range cm.activeConnections {
		ids = append(ids, id)
	}
	return ids
}

func (cm *connectionManager) GetClient(serverID string) (*ActiveConnection, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	connection, ok := cm.activeConnections[serverID]
	if !ok {
		return nil, fmt.Errorf("no active connection to mongo instance for ID: %v", serverID)
	}

	return &connection, nil
}

func (cm *connectionManager) GetDatabases(serverID string) ([]string, error) {
	connection, err := cm.GetClient(serverID)
	if err != nil {
		return nil, err
	}

	return connection.client.ListDatabaseNames(cm.ctx, bson.D{})
}
