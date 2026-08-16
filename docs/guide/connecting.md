---
title: Connecting to a server
---

# Connecting to a server

## Adding a server

In the server pane, use **Add Server** (or right-click a group and choose **Add Server**) to open the server dialog. The dialog has two tabs:

- **General** — the connection string, a display name, the group to file the server under, and a colour.
- **Authentication** — the authentication method and its settings.

Only **Name** and **Connection String** are required. **Test Connection** attempts a real connection using the current form values (this is unavailable for OIDC — see below).

## The connection string

The connection string is a standard MongoDB URI:

```
mongodb://[username:password@]host1[:port1][,...hostN[:portN]][/[defaultauthdb][?options]]
```

Vervet parses the URI as you type it and, when the URI implies a specific authentication mechanism, switches the **Authentication** tab to match automatically. If you haven't edited the name field yourself, the server's display name is also pre-filled from the URI's host.

The Authentication tab supports several mechanisms beyond a plain URI:

- **None** — no credentials.
- **Username & password (SCRAM)** — username, password, auth source and mechanism (or auto-detected).
- **X.509 certificate** — a client certificate + key PEM file, optional passphrase, and an optional username override.
- **OpenID Connect (OIDC)** — see below.
- **AWS IAM** — access key ID, secret access key, and an optional session token.
- **Kerberos (GSSAPI)** — principal, service name, host-name canonicalisation and an optional service realm.
- **LDAP (PLAIN)** — username, password and auth source.

## Where the connection string is stored

The connection string (and any credentials that go with it) is written to the OS keyring via [`go-keyring`](https://github.com/zalando/go-keyring), under the service name `Vervet`. It is **never** written to `~/.config/vervet/connections.yaml` — that file only holds server metadata: ID, name, parent group ID, and colour.

If the OS secret service is unavailable or unresponsive (for example, no keyring daemon running on Linux), keyring operations fail after a timeout and Vervet reports the error rather than hanging indefinitely.

## Signing in with OIDC

Choosing **OpenID Connect** as the authentication method opens a browser-based sign-in flow when you connect:

1. Vervet starts a local callback listener on `http://localhost:27097/redirect` and opens your default browser at the identity provider's authorisation URL (using PKCE).
2. You sign in with the identity provider in the browser.
3. The provider redirects back to the local listener, which exchanges the authorisation code for tokens and completes the connection.

Sign-in behaviour can be adjusted in the Authentication tab:

- **Open browser** — the default behaviour described above.
- **Open browser, force account picker** — adds a `prompt` parameter so the provider re-prompts for an account rather than silently reusing an existing session.
- **Show URL to copy** — instead of opening a browser automatically, Vervet shows the authorisation URL to copy and open manually (useful for signing in as a different account in a private window or separate browser profile).

The sign-in attempt times out after five minutes if it isn't completed. **Test Connection** does not support OIDC — save the server first, then connect to trigger sign-in.

A refresh token obtained through OIDC is stored in the same keyring entry as the rest of the connection config. Right-clicking a server with cached OIDC tokens offers **Reset OIDC Session**, which discards them so the next connection attempt prompts for account selection again.

## Groups and colours

Servers can be organised into groups, created either from the server dialog's **Group** field (via **Add New Group**) or independently. Groups can be nested, renamed, and moved by changing their parent. A server (or group) can also be given one of a fixed set of colour swatches, or no colour, from the **Colour** picker in the server dialog — this colour is shown against the server in the tree.

## Connecting and disconnecting

Right-click a server (or select it) and choose **Connect** to open it. A successful connection opens a browser tab for that server and emits a `connection-connected` event. Choosing **Disconnect** closes the connection and emits `connection-disconnected`; editing a server that's currently connected requires closing that connection first.
