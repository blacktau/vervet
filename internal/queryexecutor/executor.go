package queryexecutor

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
	"vervet/internal/clientregistry"
	"vervet/internal/connectionStrings"
	"vervet/internal/logging"
	"vervet/internal/models"
	"vervet/internal/queryengine"
	"vervet/internal/shell"
)

// SettingsProvider allows QueryExecutor to read app settings without depending on the full settings package.
type SettingsProvider interface {
	GetSettings() (models.Settings, error)
}

// QueryExecutor executes queries against connected servers.
// Each query spawns a one-shot mongosh process or uses the built-in goja engine.
type QueryExecutor struct {
	mu       sync.Mutex
	ctx      context.Context
	log      *slog.Logger
	registry *clientregistry.ClientRegistry
	store    connectionStrings.Store
	cancels  map[string]context.CancelFunc // serverID -> cancel for in-flight query
	cfg      shell.Config
	settings SettingsProvider
}

func NewQueryExecutor(log *slog.Logger, registry *clientregistry.ClientRegistry, store connectionStrings.Store, settings SettingsProvider) *QueryExecutor {
	return &QueryExecutor{
		log:      log.With(slog.String(logging.SourceKey, "QueryExecutor")),
		registry: registry,
		store:    store,
		cancels:  make(map[string]context.CancelFunc),
		settings: settings,
		cfg: shell.Config{
			Timeout: 30 * time.Second,
		},
	}
}

// Init stores the Wails application context, used as the parent for query contexts.
func (qe *QueryExecutor) Init(ctx context.Context) {
	qe.ctx = ctx
}

// ExecuteQuery runs a query against the given server and database.
// Only one query per server runs at a time; a new call cancels any in-flight query.
// The engine (built-in goja or mongosh) is selected based on the user's settings.
func (qe *QueryExecutor) ExecuteQuery(serverID, dbName, query string) (models.QueryResult, error) {
	qe.log.Debug("Executing query", slog.String("serverID", serverID), slog.String("dbName", dbName))

	// Cancel any in-flight query for this server
	qe.mu.Lock()
	if cancel, ok := qe.cancels[serverID]; ok {
		qe.log.Debug("Cancelling previous in-flight query", slog.String("serverID", serverID))
		cancel()
	}
	queryCtx, cancel := context.WithCancel(qe.ctx)
	qe.cancels[serverID] = cancel
	qe.mu.Unlock()

	defer func() {
		cancel()
		qe.mu.Lock()
		delete(qe.cancels, serverID)
		qe.mu.Unlock()
	}()

	cfg, _ := qe.settings.GetSettings()
	if cfg.Editor.QueryEngine == "builtin" {
		return qe.executeWithGoja(queryCtx, serverID, dbName, query)
	}
	return qe.executeWithMongosh(queryCtx, serverID, dbName, query)
}

func (qe *QueryExecutor) executeWithGoja(ctx context.Context, serverID, dbName, query string) (models.QueryResult, error) {
	client, err := qe.registry.GetClient(serverID)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("no active connection: %w", err)
	}

	engine := queryengine.NewGojaEngine(client)
	result, err := engine.ExecuteQuery(ctx, "", dbName, query)
	if err != nil {
		qe.log.Error("Goja query failed", slog.String("serverID", serverID), slog.Any("error", err))
		return models.QueryResult{}, err
	}

	qe.log.Debug("Query executed successfully (builtin)", slog.String("serverID", serverID))
	return result, nil
}

func (qe *QueryExecutor) executeWithMongosh(ctx context.Context, serverID, dbName, query string) (models.QueryResult, error) {
	cfg, err := qe.store.GetConnectionConfig(serverID)
	if err != nil {
		qe.log.Error("Failed to get connection config", slog.String("serverID", serverID), slog.Any("error", err))
		return models.QueryResult{}, err
	}

	uri := cfg.URI
	if dbName != "" {
		if !validDBName.MatchString(dbName) {
			return models.QueryResult{}, fmt.Errorf("invalid database name: %q", dbName)
		}
		uri = appendDatabase(uri, dbName)
	}

	var result models.QueryResult
	if cfg.AuthMethod == models.AuthOIDC {
		result, err = shell.ExecuteWithOIDC(ctx, uri, query, qe.cfg)
	} else {
		result, err = shell.Execute(ctx, uri, query, qe.cfg)
	}

	if err != nil {
		qe.log.Error("Query execution failed", slog.String("serverID", serverID), slog.Any("error", err))
		return models.QueryResult{}, err
	}

	qe.log.Debug("Query executed successfully (mongosh)", slog.String("serverID", serverID))
	return result, nil
}

// CancelQuery cancels any in-flight query for the given server.
func (qe *QueryExecutor) CancelQuery(serverID string) {
	qe.mu.Lock()
	defer qe.mu.Unlock()

	if cancel, ok := qe.cancels[serverID]; ok {
		qe.log.Info("Cancelling query", slog.String("serverID", serverID))
		cancel()
		delete(qe.cancels, serverID)
	}
}

// CloseAll cancels all in-flight queries.
func (qe *QueryExecutor) CloseAll() {
	qe.mu.Lock()
	defer qe.mu.Unlock()

	for id, cancel := range qe.cancels {
		cancel()
		delete(qe.cancels, id)
	}
}

// CheckMongosh returns true if mongosh is available in PATH.
func (qe *QueryExecutor) CheckMongosh() bool {
	return shell.CheckMongosh()
}

func (qe *QueryExecutor) getURI(serverID, dbName string) (string, error) {
	uri, err := qe.store.GetRegisteredServerURI(serverID)
	if err != nil {
		return "", fmt.Errorf("failed to get URI: %w", err)
	}

	if dbName != "" {
		if !validDBName.MatchString(dbName) {
			return "", fmt.Errorf("invalid database name: %q", dbName)
		}
		uri = appendDatabase(uri, dbName)
	}

	return uri, nil
}

// validDBName matches valid MongoDB database names: letters, digits, underscores, hyphens, dots.
var validDBName = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)

// appendDatabase appends the database name to a MongoDB URI.
// It handles both mongodb:// and mongodb+srv:// schemes.
func appendDatabase(uri, dbName string) string {
	// Split on '?' to preserve query parameters
	parts := strings.SplitN(uri, "?", 2)
	base := parts[0]

	// Remove trailing slash if present
	base = strings.TrimRight(base, "/")

	// Strip existing database name if present.
	// After the scheme (mongodb:// or mongodb+srv://) and host(s), the path
	// segment is the database name. We find the end of the host portion by
	// looking for the third slash (scheme has two).
	if idx := strings.Index(base, "://"); idx >= 0 {
		hostStart := idx + 3
		if slashIdx := strings.Index(base[hostStart:], "/"); slashIdx >= 0 {
			// There's already a database path — remove it
			base = base[:hostStart+slashIdx]
		}
	}

	// Append database name
	base = base + "/" + dbName

	// Re-attach query parameters if any
	if len(parts) == 2 {
		return base + "?" + parts[1]
	}

	return base
}
