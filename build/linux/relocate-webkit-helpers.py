#!/usr/bin/env python3
"""Point the AppImage's WebKit library at its bundled helper processes.

WebKitGTK looks for WebKitWebProcess and friends in a directory compiled into
the library (release builds ignore WEBKIT_EXEC_PATH). This moves that directory
to usr/lib/webkitgtk-6.0 and rewrites only the library's copy of its path to
"././webkitgtk-6.0/", NUL-padded to the same length. AppRun starts the app in
usr/lib, so the relative path resolves there, and WebKit's bubblewrap sandbox
binds it the same way. Every other path stays absolute, so the sandbox still
finds /usr/bin/bwrap and the host's libraries.

Same approach as Wails v3's own packager (internal/appimage/webkit.go). It
replaces linuxdeploy-plugin-gtk's rewrite of every /usr, which broke the
sandbox (#377).

Usage: relocate-webkit-helpers.py APPDIR
"""

import glob
import os
import sys

HELPER_DIR = "/usr/lib/x86_64-linux-gnu/webkitgtk-6.0"


def main(appdir: str) -> None:
    lib_dir = os.path.join(appdir, "usr", "lib")
    libs = glob.glob(os.path.join(lib_dir, "libwebkitgtk-6.0.so*"))
    libs = [lib for lib in libs if not os.path.islink(lib)]
    if not libs:
        sys.exit(f"no libwebkitgtk-6.0 in {lib_dir}")

    # The whole NUL-terminated string only: the injected-bundle directory
    # shares this prefix and must stay absolute. AppRun redirects it with
    # WEBKIT_INJECTED_BUNDLE_PATH.
    old = HELPER_DIR.encode() + b"\0"
    new = b"././" + os.path.basename(HELPER_DIR).encode() + b"/"
    new += b"\0" * (len(old) - len(new))

    patched = 0
    for lib in libs:
        with open(lib, "rb") as f:
            data = f.read()
        if b"././" in data:
            sys.exit(f"{lib} already has relative paths; is the plugin's rewrite still on?")
        count = data.count(old)
        if count == 0:
            continue
        with open(lib, "wb") as f:
            f.write(data.replace(old, new))
        patched += count
    if patched == 0:
        sys.exit(f"no WebKit library references {HELPER_DIR}")

    src = os.path.join(appdir, HELPER_DIR.lstrip("/"))
    dst = os.path.join(lib_dir, os.path.basename(HELPER_DIR))
    os.rename(src, dst)
    if not os.path.exists(os.path.join(dst, "WebKitWebProcess")):
        sys.exit(f"WebKitWebProcess missing from {dst}")
    print(f"relocated {HELPER_DIR} to usr/lib ({patched} reference(s))")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        sys.exit(__doc__)
    main(sys.argv[1])
