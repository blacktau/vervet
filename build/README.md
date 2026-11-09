# Build Directory

Build configuration and platform files for Wails v3. Build output goes to `bin/`
at the repository root.

- `config.yml` - product name, identifier, version and `wails3 dev` settings.
  After changing `info`, run `wails3 task common:update:build-assets` to
  regenerate the platform files below. It overwrites them, so re-apply any
  changes made by hand (see `linux/nfpm/nfpm.yaml`).
- `Taskfile.yml` - tasks shared by every platform: frontend build, bindings, icons.
- `appicon.png` - the source icon for every platform.

## darwin

- `Taskfile.yml` - builds the binary and the `.app` bundle.
- `Info.plist`, `Info.dev.plist` - bundle metadata for release and `wails3 dev` builds.

## windows

- `Taskfile.yml` - builds the binary and the NSIS installer.
- `nsis/` - the installer script. `wails_tools.nsh` is generated from `config.yml`.
- `info.json`, `wails.exe.manifest`, `icon.ico` - embedded in the executable.

## linux

- `Taskfile.yml` - builds the binary, `.deb` and `.rpm`.
- `nfpm/nfpm.yaml` - `.deb` and `.rpm` packaging. Edited by hand for install paths
  and GTK 4 dependencies, so check it after regenerating build assets.
- `vervet.spec` - builds the source RPM that Copr rebuilds from.
- `AppRun`, `xdg-open`, `com.blacktau.vervet.desktop`, `Vervet-512.png` - used by the release
  workflow's AppImage step, which uses linuxdeploy rather than `wails3 generate appimage`.
