#!/bin/sh
# Refresh the desktop database so the removed launcher disappears from menus.
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database -q /usr/share/applications
fi
exit 0
