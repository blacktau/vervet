---
layout: home
hero:
  name: Vervet
  text: A desktop MongoDB explorer
  tagline: Browse, query and manage your MongoDB servers from a native desktop app. No telemetry, no accounts, no cloud.
  image:
    src: /logo.svg
    alt: Vervet
  actions:
    - theme: brand
      text: Download
      link: /#download
    - theme: alt
      text: Documentation
      link: /guide/install
    - theme: alt
      text: GitHub
      link: https://github.com/blacktau/vervet
features:
  - title: Connection management
    details: Connect to as many servers as you like. Connection strings are held in your OS keyring, never written to a config file.
  - title: OIDC authentication
    details: Sign in to servers that use OpenID Connect, alongside standard connection-string authentication.
  - title: Workspaces
    details: Group folders of saved mongosh scripts on disk into named workspaces, and browse their contents as a file tree.
  - title: Data browser
    details: Navigate servers, databases and collections in a tree view.
  - title: Query editor
    details: A Monaco-based editor with MongoDB syntax highlighting, autocompletion and live syntax validation.
  - title: Script runner
    details: Run multi-statement mongosh-compatible scripts, including load() and script-relative file access.
  - title: Schema browser
    details: See the inferred shape of a collection — fields, types and how often each one appears.
  - title: Results viewer
    details: View query results as an expandable Table View or as read-only, syntax-highlighted EJSON in the JSON View.
  - title: Index management
    details: Create, edit and drop indexes without leaving the app.
  - title: Statistics
    details: Database and collection statistics, including storage and index sizes.
  - title: Export results
    details: Send query results out to a file when you need them somewhere else.
  - title: Multi-tab interface
    details: Keep several queries and views open at once.
  - title: Cross-platform
    details: Native builds for Linux, macOS and Windows.
---

## Download

<DownloadPicker />

## Screenshots

| Server tree | Query editor (dark) |
|---|---|
| ![Server tree](/screenshots/server-tree.png) | ![Query editor, dark theme](/screenshots/query-dark.png) |

| Query editor (light) | Statistics |
|---|---|
| ![Query editor, light theme](/screenshots/query-light.png) | ![Statistics](/screenshots/statistics.png) |

## Status

Vervet is under active development and not yet feature-complete. It is usable
day to day, but expect rough edges, and check the
[release notes](https://github.com/blacktau/vervet/releases) before upgrading.

Found a problem? [Open an issue](https://github.com/blacktau/vervet/issues).
