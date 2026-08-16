---
title: Browsing your data
---

# Browsing your data

## The server tree

Once connected, a server appears in the data browser as a tree: **server → databases → a Collections folder and a Views folder → individual collections and views**. Clicking a database or a collection/view opens a query tab for it — clicking a collection or view pre-fills the query with `db.getCollection('<name>').find({}).limit(<default limit>)`, ready to run.

Right-clicking a node offers actions appropriate to its level, including adding a database, adding a collection, renaming, dropping a database or collection, viewing indexes, viewing statistics, inspecting the schema, and disconnecting the server.

## Workspaces

Workspaces are separate from server connections — a workspace is a named collection of folders on disk (for example, a directory of saved mongosh scripts). You can create multiple workspaces, add or remove folders from the active one, and browse their contents as a file tree. Removing a folder or deleting a workspace only detaches it from Vervet; the files on disk are left alone. Files can be opened, renamed, created and deleted from the workspace tree, and a file can be opened directly against a chosen server via **Open on Server...**.

## Tabs

Work happens in tabs, similar to a browser. Opening a query against a collection, inspecting a schema, or opening a workspace file each opens its own tab, so you can have several queries or files open side by side and switch between them.

## The schema browser

The schema browser infers a collection's shape by sampling documents rather than relying on a fixed schema. Choose a sample size (100, 500, 1000 or 5000 documents) and Vervet reports, for each field path: its inferred type(s), how often it's present across the sample (as a percentage), and — for fields inside arrays — the average number of elements per parent document. Nested fields can be expanded to drill into sub-documents and arrays.

## Viewing results: table vs JSON

Query results can be viewed two ways, toggled with **Table View** and **JSON View**:

- **Table View** renders documents as an expandable tree of field/value/type rows. Rows can be expanded to reveal nested documents and arrays, and a document, a single value, or a single field can be copied to the clipboard from the row's context menu. Documents can also be opened in a dedicated **View Document** or **Edit Document** dialog.
- **JSON View** renders the same result set as read-only, syntax-highlighted EJSON in a Monaco editor pane, with folding.

## Exporting results

Query results can be exported via **Export results…**, in one of three formats: **CSV**, **JSON**, or **NDJSON**. For CSV, the field separator can be a comma, tab, semicolon, pipe or a custom character, a header row can be included or omitted, and a UTF-8 BOM can be added (useful when the file will be opened in Excel). A default filename is suggested based on the collection name and format, and Vervet reports the path the file was saved to once the export completes.
