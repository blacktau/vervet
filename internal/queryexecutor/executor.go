package queryexecutor

import (
	"context"
	"errors"
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

// ErrPagingUnsupported is returned by FetchPage / CountForPage when the active
// query engine doesn't support server-side paging (e.g. mongosh).
var ErrPagingUnsupported = errors.New("paging is supported only for the builtin engine")

// SettingsProvider allows QueryExecutor to read app settings without depending on the full settings package.
type SettingsProvider interface {
	GetSettings() (models.Settings, error)
}

// queryKey identifies a single in-flight query. Keying by both serverID and
// queryID lets multiple queries run concurrently against the same connection
// while still allowing a specific query to be cancelled.
type queryKey struct {
	serverID string
	queryID  string
}

// QueryExecutor executes queries against connected servers.
// Each query spawns a one-shot mongosh process or uses the built-in goja engine.
type QueryExecutor struct {
	mu       sync.Mutex
	ctx      context.Context
	log      *slog.Logger
	registry *clientregistry.ClientRegistry
	store    connectionStrings.Store
	cancels  map[queryKey]context.CancelFunc // (serverID, queryID) -> cancel for in-flight query
	cfg      shell.Config
	settings SettingsProvider
}

func NewQueryExecutor(log *slog.Logger, registry *clientregistry.ClientRegistry, store connectionStrings.Store, settings SettingsProvider) *QueryExecutor {
	return &QueryExecutor{
		log:      log.With(slog.String(logging.SourceKey, "QueryExecutor")),
		registry: registry,
		store:    store,
		cancels:  make(map[queryKey]context.CancelFunc),
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

// registerQuery records the cancel func for an in-flight query, keyed by
// (serverID, queryID). It deliberately does NOT cancel other queries for the
// same server — concurrent queries against one connection are supported.
func (qe *QueryExecutor) registerQuery(serverID, queryID string, cancel context.CancelFunc) {
	qe.mu.Lock()
	qe.cancels[queryKey{serverID, queryID}] = cancel
	qe.mu.Unlock()
}

// unregisterQuery removes the tracked cancel func for a finished query.
func (qe *QueryExecutor) unregisterQuery(serverID, queryID string) {
	qe.mu.Lock()
	delete(qe.cancels, queryKey{serverID, queryID})
	qe.mu.Unlock()
}

// ExecuteQuery runs a query against the given server and database.
// Multiple queries may run concurrently against the same server; each is
// tracked by queryID so it can be cancelled independently.
// The engine (built-in goja or mongosh) is selected based on the user's settings.
func (qe *QueryExecutor) ExecuteQuery(serverID, queryID, dbName, query string) (models.QueryResult, error) {
	queryCtx, cancel := context.WithCancel(qe.ctx)
	qe.registerQuery(serverID, queryID, cancel)

	defer func() {
		cancel()
		qe.unregisterQuery(serverID, queryID)
	}()

	cfg, _ := qe.settings.GetSettings()
	if cfg.Query.QueryEngine == "builtin" {
		return qe.executeWithGoja(queryCtx, serverID, dbName, query)
	}
	return qe.executeWithMongosh(queryCtx, serverID, dbName, query)
}

func (qe *QueryExecutor) executeWithGoja(ctx context.Context, serverID, dbName, query string) (models.QueryResult, error) {
	client, err := qe.registry.GetClient(serverID)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("no active connection: %w", err)
	}

	cfg, _ := qe.settings.GetSettings()
	engine := queryengine.NewGojaEngine(client, int64(cfg.Query.DefaultPageSize))
	result, err := engine.ExecuteQuery(ctx, "", dbName, query)
	if err != nil {
		return models.QueryResult{}, err
	}

	return result, nil
}

func (qe *QueryExecutor) executeWithMongosh(ctx context.Context, serverID, dbName, query string) (models.QueryResult, error) {
	cfg, err := qe.store.GetConnectionConfig(serverID)
	if err != nil {
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
		return models.QueryResult{}, err
	}

	return result, nil
}

// FetchPage fetches a single page for a previously captured PageContext using
// the builtin engine. Returns ErrPagingUnsupported when mongosh is selected.
func (qe *QueryExecutor) FetchPage(serverID, dbName string, pc models.PageContext, page, pageSize int64) (models.QueryResult, error) {
	cfg, _ := qe.settings.GetSettings()
	if cfg.Query.QueryEngine != "builtin" {
		return models.QueryResult{}, ErrPagingUnsupported
	}
	client, err := qe.registry.GetClient(serverID)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("no active connection: %w", err)
	}
	engine := queryengine.NewGojaEngine(client, int64(cfg.Query.DefaultPageSize))
	return engine.FetchPage(qe.ctx, dbName, pc, page, pageSize)
}

// CountForPage returns the row count for a PageContext using the builtin
// engine. Returns ErrPagingUnsupported when mongosh is selected.
func (qe *QueryExecutor) CountForPage(serverID, dbName string, pc models.PageContext) (int64, bool, error) {
	cfg, _ := qe.settings.GetSettings()
	if cfg.Query.QueryEngine != "builtin" {
		return 0, false, ErrPagingUnsupported
	}
	client, err := qe.registry.GetClient(serverID)
	if err != nil {
		return 0, false, fmt.Errorf("no active connection: %w", err)
	}
	engine := queryengine.NewGojaEngine(client, 0)
	return engine.CountForPage(qe.ctx, dbName, pc)
}

// CancelQuery cancels the in-flight query identified by (serverID, queryID).
// Other queries against the same server are left running.
func (qe *QueryExecutor) CancelQuery(serverID, queryID string) {
	qe.mu.Lock()
	defer qe.mu.Unlock()

	key := queryKey{serverID, queryID}
	if cancel, ok := qe.cancels[key]; ok {
		if qe.log != nil {
			qe.log.Debug("Cancelling query", slog.String("serverID", serverID), slog.String("queryID", queryID))
		}
		cancel()
		delete(qe.cancels, key)
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
