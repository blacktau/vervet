//go:build production

package main

// devBuild is false for release builds (`wails3 task build`, the release
// workflow), whatever their version string, so they ship without the web
// inspector unless started with -debug-ui.
const devBuild = false
