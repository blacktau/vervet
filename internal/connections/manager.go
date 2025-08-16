// Package connections contains code to manage active connections to mongo instances
package connections

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"vervet/internal/configuration"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ConnectionManager struct {
	ctx               context.Context
	activeConnections map[int]ActiveConnection
	mu                sync.RWMutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		activeConnections: make(map[int]ActiveConnection),
		mu:                sync.RWMutex{},
	}
}

func (cm *ConnectionManager) Init(ctx context.Context) error {
	cm.ctx = ctx
	return nil
}

// Connect establishes a connection to a MongoDB database using a securely stored URI.
// This method is exposed to Wails.
func (cm *ConnectionManager) Connect(registeredServerID int) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, ok := cm.activeConnections[registeredServerID]; ok {
		return fmt.Errorf("already connected to this Mongo Instance")
	}

	// 1. Retrieve the connection URI securely.
	uri, err := configuration.GetRegisteredServerURI(registeredServerID)
	if err != nil {
		return fmt.Errorf("error retrieving connection URI: %w", err)
	}

	clientOptions := options.Client().ApplyURI(uri)
	ctx, cancel := context.WithTimeout(cm.ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Printf("Failed to connect to MongoDB: %v", err)
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// 4. Ping the database to ensure connection is valid.
	if err = client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(cm.ctx)
		return fmt.Errorf("ping failed, connection invalid: %w", err)
	}

	activeConnection := newActiveConnection(registeredServerID)
	activeConnection.client = client
	// activeConnection.ctx = ctx
	cm.activeConnections[registeredServerID] = activeConnection

	log.Printf("Successfully connected to MongoDB via ID: %d", registeredServerID)

	return nil
}

// Disconnect closes the active MongoDB connection.
// This method is exposed to Wails.
func (cm *ConnectionManager) Disconnect(connectionID int) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if connection, ok := cm.activeConnections[connectionID]; ok {
		err := connection.Disconnect(cm.ctx)
		if err != nil {
			log.Printf("Error while diconnecting from mongo for connectionID %d: %v", connectionID, err)
			return fmt.Errorf("error diconnecting: %w", err)
		}

		delete(cm.activeConnections, connectionID)
		log.Printf("Disconnected from mongo for connectionID: %v", connectionID)
		return nil
	}

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
			log.Printf("Error when disconnecting connection with ID: %v", err)
		}

		delete(cm.activeConnections, id)
	}
	log.Print("All active mongo DB connections disconnected")
	return nil
}

// GetConnectedClientIDs returns a list od IDs for the currently active connections
// this is exposed to wails
func (cm *ConnectionManager) GetConnectedClientIDs() []int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ids := make([]int, 0, len(cm.activeConnections))
	for id := range cm.activeConnections {
		ids = append(ids, id)
	}
	return ids
}

func (cm *ConnectionManager) GetClient(connectionID int) (*ActiveConnection, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	connection, ok := cm.activeConnections[connectionID]
	if !ok {
		return nil, fmt.Errorf("no active connection to mongo instance for ID: %v", connectionID)
	}

	return &connection, nil
}
