---
title: Privacy Policy
---

# Vervet Privacy Policy

_Last updated: 2026-05-13_

Vervet is a desktop MongoDB explorer. It runs entirely on your computer.

## What Vervet collects

Nothing. Vervet has no servers, no analytics, no telemetry, no user accounts, no crash reporting, and no remote logging.

## Network activity

Vervet makes network connections only to:

1. **MongoDB servers you configure.** Vervet connects to the servers you add in the app. It does not connect to any MongoDB server you have not configured.
2. **The GitHub Releases API** (`api.github.com`), once per update-check interval, to read the public list of Vervet releases. No identifying information is sent. This check is disabled in the Microsoft Store build of Vervet (updates are handled by the Store).
3. **OIDC identity providers** you configure, when you choose OIDC authentication for a MongoDB server.

Vervet does not contact any server operated by the Vervet maintainers.

## Local storage

Vervet stores the following on your computer:

- Server metadata (name, ID, group, colour) under your platform's standard config location (e.g. `~/.config/vervet/` on Linux/macOS or the equivalent AppData location on Windows).
- MongoDB connection strings in your operating system's credential store (Keychain on macOS, Credential Manager on Windows, Secret Service on Linux).
- App settings and workspace state in the same config directory.
- Diagnostic logs in the user data directory.

None of this leaves your machine.

## Third-party services

Vervet does not embed or call any third-party analytics, advertising, crash reporting, or telemetry services.

## Contact

Questions about this policy: <vervet@blacktau.com>.

## Changes

This policy may be updated. The "Last updated" date at the top reflects the most recent change.
