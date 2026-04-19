package clientregistry

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"vervet/internal/logging"
	"vervet/internal/models"
	"vervet/internal/oidc"

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
	mu           sync.RWMutex
	ctx          context.Context
	log          *slog.Logger
	clients      map[string]registeredClient
	tokenManager *oidc.TokenManager
}

func NewClientRegistry(log *slog.Logger, tokenManager *oidc.TokenManager) *ClientRegistry {
	if log == nil {
		log = slog.Default()
	}
	return &ClientRegistry{
		log:          log.With(slog.String(logging.SourceKey, "ClientRegistry")),
		clients:      make(map[string]registeredClient),
		tokenManager: tokenManager,
	}
}

func (r *ClientRegistry) Init(ctx context.Context) {
	r.ctx = ctx
}

func (r *ClientRegistry) Connect(serverID, name, uri string) (*mongo.Client, error) {
	r.mu.Lock()
	if _, ok := r.clients[serverID]; ok {
		r.mu.Unlock()
		return nil, fmt.Errorf("already connected to server %s", serverID)
	}
	r.mu.Unlock()

	monitor := &event.CommandMonitor{
		Succeeded: func(ctx context.Context, evt *event.CommandSucceededEvent) {
			// No logging for every hello/isMaster — too chatty
		},
	}

	clientOptions := options.Client().
		ApplyURI(uri).
		SetMonitor(monitor)
	ctx, cancel := context.WithTimeout(r.ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(r.ctx)
		return nil, fmt.Errorf("ping failed, connection invalid: %w", err)
	}

	r.mu.Lock()
	if _, ok := r.clients[serverID]; ok {
		r.mu.Unlock()
		_ = client.Disconnect(r.ctx)
		return nil, fmt.Errorf("already connected to server %s", serverID)
	}
	r.clients[serverID] = registeredClient{
		client:   client,
		serverID: serverID,
		name:     name,
	}
	r.mu.Unlock()

	r.log.Debug("Registered client",
		slog.String("serverID", serverID), slog.String("name", name))
	return client, nil
}

func (r *ClientRegistry) ConnectWithConfig(serverID, name string, cfg models.ConnectionConfig) (*mongo.Client, error) {
	r.mu.Lock()
	if _, ok := r.clients[serverID]; ok {
		r.mu.Unlock()
		return nil, fmt.Errorf("already connected to server %s", serverID)
	}
	r.mu.Unlock()

	monitor := &event.CommandMonitor{
		Succeeded: func(ctx context.Context, evt *event.CommandSucceededEvent) {
			// No logging for every hello/isMaster — too chatty
		},
	}

	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetMonitor(monitor)

	if cfg.AuthMethod == models.AuthOIDC {
		credential := options.Credential{
			AuthMechanism: "MONGODB-OIDC",
			AuthMechanismProperties: map[string]string{
				"ALLOWED_HOSTS": "*",
			},
		}
		if cfg.OIDCConfig != nil && cfg.OIDCConfig.WorkloadIdentity {
			credential.OIDCMachineCallback = r.tokenManager.MachineCallback(serverID)
		} else {
			credential.OIDCHumanCallback = r.tokenManager.HumanCallback(serverID, cfg.OIDCConfig)
		}
		clientOptions.SetAuth(credential)
	}

	connectTimeout := 10 * time.Second
	if cfg.AuthMethod == models.AuthOIDC {
		connectTimeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(r.ctx, connectTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		if cfg.AuthMethod == models.AuthOIDC {
			r.tokenManager.CleanupServer(serverID)
		}
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(r.ctx)
		if cfg.AuthMethod == models.AuthOIDC {
			r.tokenManager.CleanupServer(serverID)
		}
		return nil, fmt.Errorf("ping failed, connection invalid: %w", err)
	}

	r.mu.Lock()
	if _, ok := r.clients[serverID]; ok {
		r.mu.Unlock()
		_ = client.Disconnect(r.ctx)
		return nil, fmt.Errorf("already connected to server %s", serverID)
	}
	r.clients[serverID] = registeredClient{
		client:   client,
		serverID: serverID,
		name:     name,
	}
	r.mu.Unlock()

	r.log.Debug("Registered client",
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

	// Always remove the client from the map, even if Disconnect() fails.
	// A broken client reference is worse than a missing one — the user
	// must be able to reconnect.
	delete(r.clients, serverID)

	if rc.client == nil {
		r.log.Warn("Client was nil; removed from registry without disconnecting",
			slog.String("serverID", serverID))
		return nil
	}

	if err := rc.client.Disconnect(r.ctx); err != nil {
		return fmt.Errorf("error during disconnect: %w", err)
	}

	r.log.Debug("Disconnected client", slog.String("serverID", serverID))
	return nil
}

func (r *ClientRegistry) DisconnectAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var lastErr error
	for id, rc := range r.clients {
		if rc.client == nil {
			r.log.Warn("Client was nil; removed from registry without disconnecting",
				slog.String("serverID", id))
			continue
		}
		if err := rc.client.Disconnect(r.ctx); err != nil {
			lastErr = err
		}
		delete(r.clients, id)
	}

	r.log.Debug("Disconnected all clients")
	return lastErr
}
