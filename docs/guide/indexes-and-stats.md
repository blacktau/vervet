---
title: Indexes and statistics
---

# Indexes and statistics

## Viewing a collection's indexes

Right-clicking a collection in the [data browser](/guide/browsing) and choosing **View Indexes** opens an indexes tab listing every index on that collection, with columns for **Name**, **Keys** (field: direction pairs), **Size**, **Usage**, **Unique**, **Sparse**, and **TTL (seconds)**. Size comes from the collection's `collStats` index sizes; usage is the operation count from a `$indexStats` aggregation against the collection, so both reflect the server's own accounting rather than anything Vervet computes.

## Creating an index

**Add Index** opens a dialog where you build the index key:

- One or more **key fields**, each with a **direction** of **Ascending (1)** or **Descending (-1)**. Extra key fields can be added with **Add Key** to build compound indexes, and each row (past the first) can be removed. Field name entry offers autocomplete suggestions sampled from the collection's schema.
- An optional **Index Name** — left blank, MongoDB generates one automatically.
- **Unique** and **Sparse** checkboxes.
- An optional **TTL (seconds)** — sets `expireAfterSeconds` on the index, for documents that should expire automatically.

## Editing and dropping

Selecting a row enables **Edit Index** and **Drop Index** — except for the collection's built-in `_id_` index, which can be neither edited nor dropped. Editing reopens the same dialog pre-filled with the index's current keys, name, options; confirming it drops the existing index and creates the replacement with the new definition (a warning in the dialog says so). If the new index keeps the same name, Vervet drops the old index before creating the new one, and tries to restore the original definition if the create fails; if the name changes, the new index is created first and the old one is dropped only once that succeeds. **Drop Index** asks for confirmation before removing the selected index.

## Database statistics

Right-clicking a database and choosing **Statistics** runs the server's `dbStats` command and shows it two ways: summary cards for **Collections**, **Objects**, **Avg Object Size**, **Data Size**, **Storage Size** and **Index Size** (each size formatted as a human-readable value alongside the exact byte count), followed by the full `dbStats` document underneath, with size-like fields formatted the same way. **Refresh** re-runs the command.

## Collection statistics

Right-clicking a collection and choosing **Statistics** runs `collStats` for that collection and shows summary cards for **Documents**, **Avg Doc Size**, **Data Size**, **Storage Size**, **Total Size** and **Total Index Size**, followed by the full `collStats` document. This includes **index sizes**: the document's `indexSizes` field (bytes per index, keyed by index name) is rendered with each value reformatted as a human-readable size alongside the exact byte count, the same way as the top-level size fields.
