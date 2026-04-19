// Package connections contains code to manage active connections to mongo instances
package connections

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"vervet/internal/clientregistry"
	"vervet/internal/connectionStrings"
	"vervet/internal/logging"
	"vervet/internal/models"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	ConnectedEvent    = "connection-connected"
	DisconnectedEvent = "connection-disconnected"
)

type ConnectionManager struct {
	ctx            context.Context
	registry       *clientregistry.ClientRegistry
	store          connectionStrings.Store
	serverProvider ServerProvider
	log            *slog.Logger
}

type ServerProvider interface {
	GetServer(id string) (*models.RegisteredServer, error)
}

func NewManager(log *slog.Logger, registry *clientregistry.ClientRegistry, store connectionStrings.Store, provider ServerProvider) *ConnectionManager {
	log = log.With(slog.String(logging.SourceKey, "ConnectionManager"))
	return &ConnectionManager{
		log:            log,
		registry:       registry,
		store:          store,
		serverProvider: provider,
	}
}

func (cm *ConnectionManager) Init(ctx context.Context) error {
	cm.ctx = ctx
	return nil
}

// Connect establishes a connection to a MongoDB database using a securely stored URI.
// This method is exposed to Wails.
func (cm *ConnectionManager) Connect(serverID string) (models.Connection, error) {
	if cm.registry.IsConnected(serverID) {
		cm.log.Warn("already connected to Mongo Instance", slog.String("serverID", serverID))
		return models.Connection{}, fmt.Errorf("already connected to this Mongo Instance")
	}

	server, err := cm.serverProvider.GetServer(serverID)
	if err != nil {
		return models.Connection{}, fmt.Errorf("error retrieving server: %w", err)
	}

	cfg, err := cm.store.GetConnectionConfig(serverID)
	if err != nil {
		return models.Connection{}, fmt.Errorf("error retrieving connection config: %w", err)
	}

	if cfg.AuthMethod == models.AuthOIDC {
		_, err = cm.registry.ConnectWithConfig(serverID, server.Name, cfg)
	} else {
		_, err = cm.registry.Connect(serverID, server.Name, cfg.URI)
	}

	if err != nil {
		return models.Connection{}, err
	}

	cm.log.Debug("Successfully connected to MongoDB", slog.String("serverID", serverID))
	runtime.EventsEmit(cm.ctx, ConnectedEvent, serverID)

	return models.Connection{
		ServerID: serverID,
		Name:     server.Name,
	}, nil
}

func (cm *ConnectionManager) TestConnection(uri string) (bool, error) {
	clientOptions := options.Client().ApplyURI(uri)
	ctx, cancel := context.WithTimeout(cm.ctx, 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return false, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(cm.ctx)
		return false, fmt.Errorf("failed to connection to database: %w", err)
	}

	if _, err = client.ListDatabases(ctx, bson.D{}, nil); err != nil {
		_ = client.Disconnect(cm.ctx)
		return false, fmt.Errorf("failed to retrieve list of databases: %w", err)
	}

	_ = client.Disconnect(ctx)

	return true, nil
}

func (cm *ConnectionManager) TestConnectionWithConfig(ctx context.Context, cfg models.ConnectionConfig) (bool, error) {
	if cfg.AuthMethod == models.AuthOIDC {
		return false, fmt.Errorf("test connection not supported for OIDC — save the server first, then connect")
	}

	clientOptions := options.Client().ApplyURI(cfg.URI)
	connectCtx, connectCancel := context.WithTimeout(ctx, 30*time.Second)
	defer connectCancel()

	client, err := mongo.Connect(connectCtx, clientOptions)
	if err != nil {
		return false, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = client.Ping(connectCtx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return false, fmt.Errorf("failed to connect to database: %w", err)
	}

	_ = client.Disconnect(ctx)

	return true, nil
}

func (cm *ConnectionManager) Disconnect(serverID string) error {
	err := cm.registry.Disconnect(serverID)

	// Always emit the event: the registry removes the client from its map
	// even when the underlying driver disconnect fails, so the frontend
	// must know the connection is gone.
	runtime.EventsEmit(cm.ctx, DisconnectedEvent, serverID)

	if err != nil {
		return err
	}

	cm.log.Debug("Disconnected from mongo server", slog.String("serverID", serverID))
	return nil
}

// DisconnectAll disconnects all the active connections
// this is exposed to wails
func (cm *ConnectionManager) DisconnectAll() error {
	all := cm.registry.GetAll()
	err := cm.registry.DisconnectAll()

	for _, c := range all {
		runtime.EventsEmit(cm.ctx, DisconnectedEvent, c.ServerID)
	}

	if err != nil {
		return err
	}

	cm.log.Debug("Disconnected from all mongo servers")
	return nil
}

func (cm *ConnectionManager) GetConnections() []models.Connection {
	all := cm.registry.GetAll()
	connections := make([]models.Connection, len(all))
	for i, c := range all {
		connections[i] = models.Connection{
			ServerID: c.ServerID,
			Name:     c.Name,
		}
	}
	return connections
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
