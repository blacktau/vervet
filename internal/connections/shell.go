package connections

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"vervet/internal/logging"
	"vervet/internal/shell"
)

// ShellManager executes mongosh queries against connected servers.
// Each query spawns a one-shot mongosh process (no persistent subprocess).
type ShellManager struct {
	mu      sync.Mutex
	ctx     context.Context
	log     *slog.Logger
	cm      *ConnectionManager
	cancels map[string]context.CancelFunc // serverID -> cancel for in-flight query
	cfg     shell.Config
}

func NewShellManager(log *slog.Logger, cm *ConnectionManager) *ShellManager {
	return &ShellManager{
		log:     log.With(slog.String(logging.SourceKey, "ShellManager")),
		cm:      cm,
		cancels: make(map[string]context.CancelFunc),
		cfg: shell.Config{
			Timeout: 30 * time.Second,
		},
	}
}

// Init stores the Wails application context, used as the parent for query contexts.
func (sm *ShellManager) Init(ctx context.Context) {
	sm.ctx = ctx
}

// ExecuteQuery runs a mongosh query against the given server and database.
// Only one query per server runs at a time; a new call cancels any in-flight query.
func (sm *ShellManager) ExecuteQuery(serverID, dbName, query string) (string, error) {
	sm.log.Debug("Executing query", slog.String("serverID", serverID), slog.String("dbName", dbName))

	uri, err := sm.getURI(serverID, dbName)
	if err != nil {
		sm.log.Error("Failed to get URI for query", slog.String("serverID", serverID), slog.Any("error", err))
		return "", err
	}

	// Cancel any in-flight query for this server
	sm.mu.Lock()
	if cancel, ok := sm.cancels[serverID]; ok {
		sm.log.Debug("Cancelling previous in-flight query", slog.String("serverID", serverID))
		cancel()
	}
	queryCtx, cancel := context.WithCancel(sm.ctx)
	sm.cancels[serverID] = cancel
	sm.mu.Unlock()

	defer func() {
		cancel()
		sm.mu.Lock()
		delete(sm.cancels, serverID)
		sm.mu.Unlock()
	}()

	result, err := shell.Execute(queryCtx, uri, query, sm.cfg)
	if err != nil {
		sm.log.Error("Query execution failed", slog.String("serverID", serverID), slog.String("dbName", dbName), slog.Any("error", err))
		return "", err
	}

	sm.log.Debug("Query executed successfully", slog.String("serverID", serverID), slog.String("dbName", dbName))
	return result, nil
}

// CancelQuery cancels any in-flight query for the given server.
func (sm *ShellManager) CancelQuery(serverID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if cancel, ok := sm.cancels[serverID]; ok {
		sm.log.Info("Cancelling query", slog.String("serverID", serverID))
		cancel()
		delete(sm.cancels, serverID)
	}
}

// CloseAll cancels all in-flight queries.
func (sm *ShellManager) CloseAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for id, cancel := range sm.cancels {
		cancel()
		delete(sm.cancels, id)
	}
}

// CheckMongosh returns true if mongosh is available in PATH.
func (sm *ShellManager) CheckMongosh() bool {
	return shell.CheckMongosh()
}

func (sm *ShellManager) getURI(serverID, dbName string) (string, error) {
	uri, err := sm.cm.store.GetRegisteredServerURI(serverID)
	if err != nil {
		return "", fmt.Errorf("failed to get URI: %w", err)
	}

	if dbName != "" {
		uri = appendDatabase(uri, dbName)
	}

	return uri, nil
}

// appendDatabase appends the database name to a MongoDB URI.
// It handles both mongodb:// and mongodb+srv:// schemes.
func appendDatabase(uri, dbName string) string {
	// Split on '?' to preserve query parameters
	parts := strings.SplitN(uri, "?", 2)
	base := parts[0]

	// Remove trailing slash if present
	base = strings.TrimRight(base, "/")

	// Append database name
	base = base + "/" + dbName

	// Re-attach query parameters if any
	if len(parts) == 2 {
		return base + "?" + parts[1]
	}

	return base
}
