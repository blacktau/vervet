---
title: Querying
---

# Querying

## Opening a query tab

Clicking a database or a collection/view in the [data browser](/guide/browsing) opens a query tab for it. Clicking a collection or view pre-fills the editor with `db.getCollection('<name>').find({}).limit(<default limit>)`, ready to run. A query tab can also be opened directly against a workspace file via **Open on Server...**, or a new, empty tab can be started and pointed at any database with the database selector in the toolbar.

Each query tab is a small JavaScript editor (Monaco) plus a results pane, and several tabs can be open side by side.

## Two query engines

Vervet can run queries two ways, chosen in **Settings → Query Engine**:

- **Built-in (recommended)** — a JavaScript engine (goja) embedded in Vervet. No external dependencies.
- **mongosh** — shells out to a real `mongosh` binary on your `PATH`, for full shell compatibility. If mongosh isn't found, query tabs show a warning and the run button is disabled.

The built-in engine only understands a fixed set of collection-level methods (listed below). Anything else — for example `db.collection.aggregate().explain()` chaining beyond what's listed, or shell helpers the built-in engine doesn't implement — fails with *"unsupported operation … Switch to mongosh engine in settings for full shell compatibility"*.

## Query forms the engine accepts

Whichever engine is selected, a query is a snippet of JavaScript against the `db` object. The following `db.<database-level>` methods are supported: `runCommand`, `adminCommand`, `getName`, `getCollection`, `getCollectionNames`, `getCollectionInfos`, `createCollection`, `createView`, `dropDatabase`, `stats`, `version`, `getSiblingDB`, `getMongo`, `aggregate`, and the user/role management methods (`createUser`, `dropUser`, `getUser`, `getUsers`, `updateUser`, `changeUserPassword`, `grantRolesToUser`, `revokeRolesFromUser`, `dropAllUsers`, `createRole`, `dropRole`, `getRole`, `getRoles`, `updateRole`, `grantPrivilegesToRole`, `revokePrivilegesFromRole`, `grantRolesToRole`, `revokeRolesFromRole`, `dropAllRoles`).

On a collection (`db.collection.<method>`), the built-in engine dispatches these methods: `find`, `findOne`, `insertOne`, `insertMany`, `updateOne`, `updateMany`, `deleteOne`, `deleteMany`, `replaceOne`, `countDocuments`, `estimatedDocumentCount`, `aggregate`, `distinct`, `findOneAndDelete`, `findOneAndReplace`, `findOneAndUpdate`, `bulkWrite`, `drop`, `createIndex`, `createIndexes`, `dropIndex`, `dropIndexes`, and `listIndexes` — plus `explain()` on a `find`/`findOne` cursor. A leading `use <database>` line switches the tab's database and is stripped before the rest of the script runs.

Also supported as JavaScript utilities inside a query: `EJSON.stringify`, `EJSON.parse`, `EJSON.serialize` and `EJSON.deserialize`, for working with Extended JSON values directly.

## Autocompletion

Autocompletion is driven by where the cursor sits in the script, not just plain word matching:

- **`db.`** — suggests known collection names for the connected database.
- **`db.getCollection('`** — suggests collection names as a quoted string.
- **`db.<collection>.`** — suggests the collection methods above, plus a set of extra shell-style helpers Vervet offers as snippets: `stats`, `isCapped`, `dataSize`, `storageSize`, `totalIndexSize`, `totalSize`, `getIndexes`, `count`, `renameCollection`, `validate`, and `findAndModify`.
- **After a closing `)` followed by `.`** (a chained cursor call, e.g. `.find({}).`) — suggests cursor methods: `limit`, `skip`, `sort`, `toArray`, `count`, `forEach`, `pretty`, `explain`, `hint`, `batchSize`, `maxTimeMS`, `collation`, `comment`, `map`, `hasNext`, `next`.
- **Inside a filter/update object's field position** (`{ ` or after a comma) — suggests field names sampled from the collection's schema (see [Browsing your data](/guide/browsing#the-schema-browser)), fetched via a 100-document sample and cached per collection until the server disconnects.
- **Inside a `$`-prefixed operator position within a filter** — suggests query operators (comparison, logical, element, evaluation, array, geospatial and bitwise operators, e.g. `$eq`, `$in`, `$and`, `$exists`, `$regex`, `$elemMatch`, `$geoWithin`, `$bitsAllSet`, …).
- **Inside `updateOne`/`updateMany`/`findOneAndUpdate`'s update object** — suggests update operators (`$set`, `$inc`, `$unset`, `$push`, `$pull`, `$rename`, `$each`, `$position`, `$slice`, `$bit`, …).
- **Inside `db.collection.aggregate([ ])`, at a new stage position** — suggests aggregation stage names (`$match`, `$group`, `$project`, `$lookup`, `$unwind`, `$sort`, `$facet`, `$bucket`, `$graphLookup`, `$setWindowFields`, `$vectorSearch`, …).
- **Inside an aggregation stage's expression object** (e.g. inside `$group`'s accumulator values) — suggests aggregation expressions (accumulators like `$sum`/`$avg`/`$push`, arithmetic, string, array, date, conditional and type-conversion operators).
- **`EJSON.`** — suggests the EJSON helper methods above.
- **`use `** at the start of a line — suggests known database names.

## Syntax validation

As you type, the query is parsed with a JavaScript parser (Babel) and any syntax errors are shown as red squiggles in the editor, with the error message on hover — this happens purely client-side and doesn't require running the query. Validation is debounced (250ms after you stop typing) and understands a few shell-only conveniences (`show dbs`, `use <db>`, a bare `it`) that aren't valid JavaScript but are otherwise accepted, so they don't trigger a false error.

## Running a query and reading the results

Click **Run** (or press **F5** / **Ctrl+Enter**, **Cmd+Enter** on macOS) to execute. If text is selected in the editor, only the selection runs; otherwise the whole tab runs. While a query is running, the **Run** button becomes a **Cancel** button showing the elapsed time, and cancelling stops the in-flight query without affecting other queries running against the same server.

Results appear in the **Results** tab:

- Matching documents render in **Table View** by default, or **JSON View** for read-only syntax-highlighted Extended JSON — see [Browsing your data](/guide/browsing#viewing-results-table-vs-json) for what each view offers.
- If the query's result can't be shown as documents (for example a shell command that prints plain text), the raw output is shown instead.
- If the query ended in `.limit(n)`, and exactly `n` documents came back, a hint is shown: *"Limit `n` in effect — more documents may exist"*.
- A query returning no documents shows *"No documents returned"*.
- Failed queries show the error in the Results tab and switch focus to it automatically.

The **Messages** tab keeps a running log of what happened for every query run in the tab: an "Executing query..." line when it starts, a result line, and any errors — each tagged **Info**, **Warning** or **Error** and filterable by those three levels. Each message can be copied to the clipboard, and clicking a message jumps the editor to the query it came from. **Clear Messages** empties the log.

## Query timing

Every successful query reports how long it took, appended to its result message in the Messages tab (for example *"12 document(s) returned in 340ms"*, or in seconds once over a second). While a query is still running, the toolbar's Cancel button and the Results tab's loading state both show a live elapsed-time clock in `m:ss` (or `h:mm:ss` past an hour), updated four times a second. If the tab is switched away from while a query is running, a background notification reports when it finishes (or fails) along with the elapsed time.
