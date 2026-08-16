---
title: Install
---

# Install

<DownloadPicker />

## Linux

Linux builds are amd64 only — there is no Linux arm64 build.

Three packages are published for each release:

- **AppImage** — download `Vervet-linux-amd64.AppImage`, make it executable, then run it:

  ```bash
  chmod +x Vervet-linux-amd64.AppImage
  ./Vervet-linux-amd64.AppImage
  ```

- **.deb** — for Debian/Ubuntu-based distributions:

  ```bash
  sudo dpkg -i Vervet-linux-amd64.deb
  ```

- **.rpm** — for Fedora/RHEL-based distributions:

  ```bash
  sudo rpm -i Vervet-linux-amd64.rpm
  ```

  A source RPM (`Vervet-linux-amd64.src.rpm`) is also published alongside the binary RPM, and builds are mirrored to a Copr repository.

## macOS

macOS builds are published as `.dmg` images for both Intel (amd64) and Apple silicon (arm64) — download the one matching your Mac.

The DMG is not code-signed or notarised. On first launch, macOS Gatekeeper will refuse to open it as a normal double-click. To open it:

1. Drag Vervet from the mounted DMG into `/Applications`.
2. In Finder, **right-click (or Control-click) Vervet.app** and choose **Open**.
3. In the warning dialog, click **Open** again to confirm. This only needs to be done once — subsequent launches work normally.

If macOS instead reports that the app "is damaged and can't be opened", the quarantine attribute needs removing directly:

```bash
xattr -cr /Applications/Vervet.app
```

## Windows

Windows builds are published for both amd64 and arm64, as two artifact types:

- **Installer** — `Vervet-windows-<arch>-installer.exe`, an NSIS-built installer.
- **Portable** — `Vervet-windows-<arch>-portable.zip`, a zip containing `Vervet.exe` that can be run without installing.

Windows SmartScreen may warn that the app is from an unrecognised publisher, since the build is not code-signed. Choose **More info** → **Run anyway** to proceed.

## Updates

Vervet checks GitHub for a newer release and, when one is found and not already dismissed, shows an in-app notification (`update-available`). The check:

- Compares your build's version against the latest GitHub release for `blacktau/vervet`.
- Only runs for versioned release builds — development builds (version `dev`) never check.
- Runs on a schedule controlled by **Settings → Updates → Check for updates**, with four options: **Never**, **On startup**, **Daily**, or **Weekly**. The default is daily.
- Can be triggered on demand from the same Settings page with **Check now**, which also shows the current version and when the app was last checked.
- Lets you dismiss a specific version; a dismissed version won't notify again until a newer one is released.

Vervet does not download or install updates itself — the notification links to the GitHub release for you to download manually.
