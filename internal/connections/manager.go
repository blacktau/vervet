package connections

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"sync"
	"time"
	"vervet/internal/configuration"
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
func (cm *ConnectionManager) Connect(connectionID int) (bool, string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, ok := cm.activeConnections[connectionID]; ok {
		return true, "Already connected to this MongoDB instance"
	}

	// 1. Retrieve the connection URI securely.
	uri, err := configuration.GetConnectionURI(connectionID)
	if err != nil {
		return false, fmt.Sprintf("Error retrieving connection URI: %v", err)
	}

	clientOptions := options.Client().ApplyURI(uri)
	ctx, cancel := context.WithTimeout(cm.ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Printf("Failed to connect to MongoDB: %v", err)
		return false, fmt.Sprintf("Failed to connect to database: %v", err)
	}

	// 4. Ping the database to ensure connection is valid.
	if err = client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(cm.ctx)
		return false, fmt.Sprintf("Ping failed, connection invalid: %v", err)
	}

	activeConnection := newActiveConnection(connectionID)
	activeConnection.client = client
	// activeConnection.ctx = ctx
	cm.activeConnections[connectionID] = activeConnection

	log.Printf("Successfully connected to MongoDB via ID: %d", connectionID)

	return true, "Successfully connected"
}

// Disconnect closes the active MongoDB connection.
// This method is exposed to Wails.
func (cm *ConnectionManager) Disconnect(connectionID int) (bool, string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if connection, ok := cm.activeConnections[connectionID]; ok {
		err := connection.Disconnect(cm.ctx)
		if err != nil {
			log.Printf("Error while diconnecting from mongo for connectionID %d: %v", connectionID, err)
			return false, fmt.Sprintf("Error diconnecting: %v", err)
		}

		delete(cm.activeConnections, connectionID)
		log.Printf("Disconnected from mongo for connectionID: %v", connectionID)
		return true, "Disconnection successful"

	}

	return false, "Connection not found or not active"
}

// DisconnectAll disconnects all the active connections
// this is exposed to wails
func (cm *ConnectionManager) DisconnectAll() (bool, string) {
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
	return true, "All connections disconnected"
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
