---
title: Scripts
---

# Scripts

## What the script runner is

A query tab isn't limited to a single expression — the whole contents of the editor is sent to the selected [query engine](/guide/querying#two-query-engines) as one script, and both engines (built-in and mongosh) run it as ordinary top-level JavaScript rather than wrapping it in a function. That matters for scripts that reassign `db`, since MongoDB shell scripts commonly start with something like `const db = db.getSiblingDB('other')` — evaluated at true top level, that line reads the existing global `db` and rebinds it; evaluated inside a function body, plain JavaScript's temporal dead zone would throw *"Cannot access 'db' before initialization"*. Both engines rewrite top-level `const`/`let` to `var` (in a way that preserves line and column numbers, so error locations still point at your source) specifically to match this shell behaviour.

## Multi-statement scripts

A tab can contain any number of statements — declarations, loops, `if`/`try`, helper functions, several queries in sequence. Only the value of the **last top-level statement** is captured as the tab's result (for example the last `find(...).toArray()` or the object a script builds up); everything before it just runs for effect. Anything printed along the way with `print()` or `console.log`/`console.error` is collected too: if the last statement doesn't produce a capturable value, that printed output becomes the raw result shown in the Results tab, and with mongosh, any output the script sent to stderr (warnings, `console.error`) is appended after the structured result rather than discarded.

## `load()`, `__dirname` and script-relative file access

Once a query tab has been saved to a file, scripts run in it get the same file-location globals mongosh provides:

- **`__filename`** — the absolute path of the saved script.
- **`__dirname`** — the directory it lives in.
- **`process.cwd()`** — also resolves to that directory.
- **`load(path)`** — runs another script. A relative `path` is resolved against `__dirname`. The loaded file gets the same top-level `const`/`let` rewrite as the main script, so anything it declares becomes visible to the caller afterwards — `load()` returns `true`, matching mongosh.
- File access via `require('fs')` (`readFileSync`, `writeFileSync`, `existsSync`, `readdirSync`, and so on) and `require('path')` also resolve relative paths against the script's own directory, rather than wherever Vervet happened to be launched from.

An **unsaved** tab has no path: `__filename` is empty and `__dirname`/`process.cwd()` fall back to Vervet's own working directory, so `load()` and relative file access won't reach your project files until the tab is saved somewhere.

This applies to both engines. With mongosh, the temp script mongosh actually runs is written next to your saved file and mongosh's working directory is set to match, so mongosh's own `__dirname`/`load()` behaviour is correct rather than pointing at a temp directory.

## Known mongosh compatibility limits

The built-in engine is not a full shell — it only implements the fixed list of `db`/collection methods described in [Querying](/guide/querying#query-forms-the-engine-accepts). A script that calls anything outside that list (shell-only helpers, more exotic cursor chaining, etc.) fails with *"unsupported operation … Switch to mongosh engine in settings for full shell compatibility"*. Switching **Settings → Query Engine** to **mongosh** runs the script through a real `mongosh` binary instead, which understands the full shell API — at the cost of requiring mongosh to be installed and on `PATH`.

## Example

A saved script that inserts a document, then reads back everything matching a filter — a typical two-statement script where only the final `find().toArray()` becomes the tab's result:

```javascript
db.getCollection('people').insertOne({ name: 'Ada Lovelace', role: 'pioneer' })
db.getCollection('people').find({ role: 'pioneer' }).toArray()
```

This runs unchanged on either engine. Once the tab is saved (for example as `seed.js`), you could split the insert into a sibling file and pull it in with `load()`:

```javascript
// helper.js, saved next to seed.js
db.getCollection('people').insertOne({ name: 'Ada Lovelace', role: 'pioneer' })
```

```javascript
// seed.js
load(__dirname + '/helper.js')
db.getCollection('people').find({ role: 'pioneer' }).toArray()
```
