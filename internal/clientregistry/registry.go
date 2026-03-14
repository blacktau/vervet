package clientregistry

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"vervet/internal/logging"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type registeredClient struct {
	client   *mongo.Client
	serverID string
	name     string
}

type ConnectedClient struct {
	ServerID string
	Name     string
}

type ClientRegistry struct {
	mu      sync.RWMutex
	ctx     context.Context
	log     *slog.Logger
	clients map[string]registeredClient
}

func NewClientRegistry(log *slog.Logger) *ClientRegistry {
	if log == nil {
		log = slog.Default()
	}
	return &ClientRegistry{
		log:     log.With(slog.String(logging.SourceKey, "ClientRegistry")),
		clients: make(map[string]registeredClient),
	}
}

func (r *ClientRegistry) Init(ctx context.Context) {
	r.ctx = ctx
}

func (r *ClientRegistry) Connect(serverID, name, uri string) (*mongo.Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.clients[serverID]; ok {
		return nil, fmt.Errorf("already connected to server %s", serverID)
	}

	monitor := &event.CommandMonitor{
		Succeeded: func(ctx context.Context, evt *event.CommandSucceededEvent) {
			if evt.CommandName == "hello" || evt.CommandName == "isMaster" {
				r.log.Info("Connected to MongoDB",
					slog.String("ConnectionID", evt.ConnectionID),
					slog.Any("Reply", evt.Reply))
			}
		},
	}

	clientOptions := options.Client().
		ApplyURI(uri).
		SetMonitor(monitor)
	ctx, cancel := context.WithTimeout(r.ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		r.log.Error("Failed to connect to MongoDB",
			slog.String("serverID", serverID), slog.Any("error", err))
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		r.log.Error("Ping failed",
			slog.String("serverID", serverID), slog.Any("error", err))
		_ = client.Disconnect(r.ctx)
		return nil, fmt.Errorf("ping failed, connection invalid: %w", err)
	}

	r.clients[serverID] = registeredClient{
		client:   client,
		serverID: serverID,
		name:     name,
	}

	r.log.Info("Registered client",
		slog.String("serverID", serverID), slog.String("name", name))
	return client, nil
}

func (r *ClientRegistry) GetClient(serverID string) (*mongo.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rc, ok := r.clients[serverID]
	if !ok {
		return nil, fmt.Errorf("no active connection for server %s", serverID)
	}
	return rc.client, nil
}

func (r *ClientRegistry) IsConnected(serverID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.clients[serverID]
	return ok
}

func (r *ClientRegistry) GetAll() []ConnectedClient {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ConnectedClient, 0, len(r.clients))
	for _, rc := range r.clients {
		result = append(result, ConnectedClient{
			ServerID: rc.serverID,
			Name:     rc.name,
		})
	}
	return result
}

func (r *ClientRegistry) Disconnect(serverID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rc, ok := r.clients[serverID]
	if !ok {
		return fmt.Errorf("no active connection for server %s", serverID)
	}

	if err := rc.client.Disconnect(r.ctx); err != nil {
		r.log.Error("Error disconnecting",
			slog.String("serverID", serverID), slog.Any("error", err))
		return fmt.Errorf("error during disconnect: %w", err)
	}

	delete(r.clients, serverID)
	r.log.Info("Disconnected client", slog.String("serverID", serverID))
	return nil
}

func (r *ClientRegistry) DisconnectAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var lastErr error
	for id, rc := range r.clients {
		if err := rc.client.Disconnect(r.ctx); err != nil {
			r.log.Error("Error disconnecting",
				slog.String("serverID", id), slog.Any("error", err))
			lastErr = err
		}
		delete(r.clients, id)
	}

	r.log.Info("Disconnected all clients")
	return lastErr
}
