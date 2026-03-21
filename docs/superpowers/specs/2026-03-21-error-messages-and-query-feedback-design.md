# Error Messages and Query Feedback Design

## Problem

The query tab has three feedback gaps:

1. **Errors are not translatable** — Go backend returns hardcoded English error strings (e.g., `"Error connecting to mongo instance: %v"`). These arrive at the frontend as raw strings with no i18n support.
2. **Messages pane is underused** — only shows `[ERROR]` entries. No success feedback like result counts or execution time.
3. **No zero-results indicator** — when a query succeeds but returns nothing, the results pane is blank with no indication of what happened.

## Approach

Centralised error classification in Go with structured error codes returned to the frontend, which maps codes to i18n keys. The messages pane becomes an informational log for all query activity.

## Design

### 1. Error Codes and Classification

#### New package: `internal/errcodes/`

Defines error code constants and a `ClassifyError` function.

**Error code constants:**

| Code | Source | When |
|---|---|---|
| `connection_failed` | mongo driver | Can't reach server |
| `auth_failed` | mongo driver | Bad credentials / auth mechanism |
| `query_timeout` | mongo driver / mongosh | Operation exceeded deadline |
| `query_syntax_error` | mongosh / goja | Invalid JS or query syntax |
| `network_error` | mongo driver | Connection dropped mid-operation |
| `not_authorized` | mongo driver | User lacks permission for operation |
| `namespace_not_found` | mongo driver | DB or collection doesn't exist |
| `shell_not_found` | shell package | mongosh binary not on PATH |
| `no_database_selected` | frontend only | Pre-flight check (never in a `Result`) |
| `query_cancelled` | frontend / executor | User cancelled (frontend-originated when user clicks cancel; backend-originated via `context.Canceled`) |
| `operation_not_supported` | query engine | Unrecognised command for current engine |
| `unknown_error` | fallback | Anything unclassified |

**Note on frontend-only codes:** `no_database_selected` is a pre-flight check in the frontend store and never passes through `ClassifyError` or appears in a `Result`. The frontend maps it directly to its i18n key. `query_cancelled` can be either frontend-originated (user clicks cancel, store sets error directly) or backend-originated (`context.Canceled` classified by `ClassifyError`).

**`ClassifyError` function:**

```go
type ClassifiedError struct {
    Code   string
    Detail string
}

func ClassifyError(err error) ClassifiedError
```

Returns a typed struct (not two bare strings) for clarity. Uses:
- `errors.Is()` / `errors.As()` against `mongo.CommandError` (check code numbers for auth, namespace, syntax), `mongo.NetworkError`, `context.DeadlineExceeded`, `context.Canceled`
- Sentinel errors from the shell package (`ErrShellNotFound`, `ErrQueryTimeout`)
- Falls back to `unknown_error` with raw `err.Error()` as detail

Sentinel errors for non-mongo cases (shell not found, query cancelled, operation not supported) are also defined in the relevant packages and checked by `ClassifyError`.

### 2. Result Struct Changes

`internal/api/result.go` — replace `Error string` with `ErrorCode` + `ErrorDetail` on both structs:

```go
type Result[T any] struct {
    IsSuccess   bool   `json:"isSuccess"`
    Data        T      `json:"data"`
    ErrorCode   string `json:"errorCode,omitempty"`
    ErrorDetail string `json:"errorDetail,omitempty"`
}

type EmptyResult struct {
    IsSuccess   bool   `json:"isSuccess"`
    ErrorCode   string `json:"errorCode,omitempty"`
    ErrorDetail string `json:"errorDetail,omitempty"`
}
```

No backward compatibility shim — `Error string` is removed. All frontend consumers are migrated in this change.

**Helper changes:** Update existing `Success()` and `Error()` helpers, and add new generic helpers:

```go
// Existing helpers updated
func Success() EmptyResult { ... }
func Fail(err error) EmptyResult { ... }  // replaces Error(), uses ClassifyError

// New generic helpers
func SuccessResult[T any](data T) Result[T] { ... }
func FailResult[T any](err error) Result[T] { ... }  // uses ClassifyError
```

### 3. QueryResult Struct Changes

`internal/models/query_result.go` — add fields for richer feedback:

```go
type QueryResult struct {
    Documents     []any  `json:"documents"`
    RawOutput     string `json:"rawOutput"`
    OperationType string `json:"operationType,omitempty"`
    AffectedCount int    `json:"affectedCount,omitempty"`
}
```

- `OperationType`: operation that was executed (see below)
- `AffectedCount`: number of documents affected (for mutations)

`DocumentCount` is intentionally omitted — the frontend computes it from `len(documents)`.

**Supported operation types:** `find`, `findOne`, `aggregate`, `insertOne`, `insertMany`, `updateOne`, `updateMany`, `deleteOne`, `deleteMany`, `replaceOne`, `countDocuments`, `estimatedDocumentCount`, `distinct`, `findOneAndDelete`, `findOneAndReplace`, `findOneAndUpdate`, `bulkWrite`, `drop`, `createIndex`, `createIndexes`, `dropIndex`, `dropIndexes`, `listIndexes`

**Where populated:**
- **Dispatch engine** (`dispatch.go`): each of the ~20 dispatch functions sets `OperationType` and `AffectedCount` on the returned `QueryResult`. This is the primary source.
- **Goja engine**: delegates to dispatch, so inherits the same fields.
- **Mongosh engine**: `OperationType` will be empty — the shell returns raw text/JSON with no structured operation metadata. The frontend falls back to `query.messages.genericResult` when `OperationType` is empty.

Any operation type without a specific i18n key also falls back to `query.messages.genericResult`.

The results pane always shows whatever BSON documents or raw output were returned — including mutation result documents (e.g., `{ acknowledged: true, insertedId: ... }`).

### 4. Messages Pane Enhancements

**Log line format:** `HH:MM:SS [TAG] <translated message>`

| Event | Tag | Example |
|---|---|---|
| Query starts | `[INFO]` | `12:30:45 [INFO] Executing query...` |
| Query succeeds (find/aggregate) | `[INFO]` | `12:30:45 [INFO] 42 document(s) returned in 230ms` |
| Query succeeds (mutation) | `[INFO]` | `12:30:45 [INFO] 3 document(s) deleted in 120ms` |
| Query succeeds, 0 results | `[INFO]` | `12:30:45 [INFO] 0 documents returned in 230ms` |
| Query error | `[ERROR]` | `12:30:45 [ERROR] Authentication failed` |
| Error detail (raw) | `[ERROR]` | `12:30:45 [ERROR] SCRAM auth error: bad auth...` |
| Query cancelled | `[WARNING]` | `12:30:45 [WARNING] Query cancelled` |

Tags (`[ERROR]`, `[WARNING]`, `[INFO]`) remain untranslated — they're technical log tokens. The message text after the tag is translated via i18n.

**Execution timing:** `queryStore` records `Date.now()` before and after the proxy call. No Go-side timing needed.

**Accumulation:** Messages accumulate across runs within the same query tab.

**Clear button:** Small icon button in the messages tab header to clear all messages. Clearing sets `queryState.messages` to empty string — no "Messages cleared" line is logged.

**Duration format:** Durations under 1000ms display as `XXXms`. Durations of 1s or more display as `X.Xs` (one decimal place). Example: `230ms`, `1.2s`, `45.0s`.

### 5. Zero Results Empty State

When a query succeeds but `Documents` is empty AND `RawOutput` is empty:
- **Results pane:** shows a centered, muted "No documents returned" message (same style as existing `query.emptyState`)
- **Messages pane:** shows `[INFO] 0 documents returned in <time>`

Does NOT show when:
- There's an error (error display takes priority)
- Documents or raw output exist
- Query is loading
- Initial state (existing `query.emptyState` handles that)

### 6. i18n Keys

**Error codes:**

```
errors.connection_failed → "Failed to connect to the server"
errors.auth_failed → "Authentication failed"
errors.query_timeout → "Query timed out"
errors.query_syntax_error → "Query syntax error"
errors.network_error → "Network error"
errors.not_authorized → "Not authorised to perform this operation"
errors.namespace_not_found → "Database or collection not found"
errors.shell_not_found → "mongosh is not installed or not found on PATH"
errors.no_database_selected → "No database selected"
errors.query_cancelled → "Query cancelled"
errors.operation_not_supported → "Operation not supported by the current query engine"
errors.unknown_error → "An unexpected error occurred"
```

**Messages pane:**

```
query.messages.executing → "Executing query..."
query.messages.findResult → "{count} document(s) returned in {time}"
query.messages.aggregateResult → "{count} document(s) returned in {time}"
query.messages.insertOneResult → "1 document inserted in {time}"
query.messages.insertManyResult → "{count} document(s) inserted in {time}"
query.messages.updateOneResult → "{count} document(s) updated in {time}"
query.messages.updateManyResult → "{count} document(s) updated in {time}"
query.messages.deleteOneResult → "{count} document(s) deleted in {time}"
query.messages.deleteManyResult → "{count} document(s) deleted in {time}"
query.messages.createIndexResult → "Index created in {time}"
query.messages.dropIndexResult → "Index dropped in {time}"
query.messages.commandResult → "Command completed in {time}"
query.messages.genericResult → "Query completed in {time}"
```

**Empty state:**

```
query.noResults → "No documents returned"
```

**Removed keys** (replaced by error codes):
- `query.noDatabaseSelected` → `errors.no_database_selected`
- `query.mongoshNotFound` → `errors.shell_not_found`
- `query.queryCancelled` → `errors.query_cancelled`

### 7. Highlight.js

No changes needed. The existing `vervet-log` language registration already matches `[ERROR]`, `[WARNING]` patterns. `[INFO]` lines get no special styling (default text colour), which is correct — they're informational.

## Files Changed

### Go backend

| File/Package | Change |
|---|---|
| New `internal/errcodes/` | Error code constants, `ClassifiedError` struct, `ClassifyError()` |
| `internal/api/result.go` | Add `ErrorCode` + `ErrorDetail` to `Result[T]` and `EmptyResult`; update `Success()`/`Error()` helpers; add `FailResult[T]()` and `Fail()` |
| `internal/models/query_result.go` | Add `OperationType`, `AffectedCount` |
| `internal/api/connections.go` | Use `ClassifyError()` in all error returns |
| `internal/api/shell.go` | Use `ClassifyError()` in `ExecuteQuery` |
| `internal/queryengine/dispatch.go` | Populate `OperationType` + `AffectedCount` in all ~20 dispatch functions |
| `internal/queryexecutor/executor.go` | Populate new `QueryResult` fields where applicable |
| `internal/shell/shell.go` | Ensure sentinel errors are classifiable |

### Go tests

| File | Change |
|---|---|
| `internal/api/result_test.go` | Update assertions for new struct fields |
| Other test files using `Result`/`EmptyResult` | Update `.Error` assertions to also check `.ErrorCode` |

### Frontend

All frontend consumers of `result.error` are migrated to use `result.errorCode` / `result.errorDetail`. The pattern for non-query stores: map `errorCode` to i18n for the notifier toast, include `errorDetail` in the toast body for debugging context.

| File | Change |
|---|---|
| `frontend/src/i18n/en-GB/index.ts` | Add `errors.*`, `query.messages.*`, `query.noResults`; remove replaced keys |
| `frontend/src/features/queries/queryStore.ts` | Map error codes to i18n; append `[INFO]` messages; timing; duration formatting |
| `frontend/src/features/queries/QueryTab.vue` | Zero-results empty state; clear messages button |
| `frontend/src/features/server-pane/serverStore.ts` | Migrate `result.error` → `errorCode`/`errorDetail` |
| `frontend/src/features/server-pane/ServerDialog.vue` | Migrate `result.error` → `errorCode`/`errorDetail` |
| `frontend/src/features/data-browser/browserStore.ts` | Migrate `result.error` → `errorCode`/`errorDetail` |
| `frontend/src/features/data-browser/AddDatabaseDialog.vue` | Migrate `result.error` → `errorCode`/`errorDetail` |
| `frontend/src/features/data-browser/AddCollectionDialog.vue` | Migrate `result.error` → `errorCode`/`errorDetail` |
| `frontend/src/features/data-browser/RenameCollectionDialog.vue` | Migrate `result.error` → `errorCode`/`errorDetail` |
| `frontend/src/features/data-browser/DataBrowserTree.vue` | Migrate `result.error` → `errorCode`/`errorDetail` |
| `frontend/src/features/settings/settingsStore.ts` | Migrate `result.error` → `errorCode`/`errorDetail` |
| `frontend/src/features/indexes/indexStore.ts` | Migrate `result.error` → `errorCode`/`errorDetail` |
| `frontend/src/features/results-document-tree/DocumentEditDialog.vue` | Migrate `result.error` → `errorCode`/`errorDetail` |
| `frontend/src/features/results-document-tree/DocumentTreeTable.vue` | Migrate `result.error` → `errorCode`/`errorDetail` |
| `frontend/src/features/statistics/CollectionStatisticsTab.vue` | Migrate `result.error` → `errorCode`/`errorDetail` |
| `frontend/src/features/statistics/DatabaseStatisticsTab.vue` | Migrate `result.error` → `errorCode`/`errorDetail` |
| `frontend/src/features/statistics/ServerStatisticsTab.vue` | Migrate `result.error` → `errorCode`/`errorDetail` |
| `frontend/src/features/server-pane/tests/connectionString.test.ts` | Update test assertions |
