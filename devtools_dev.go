//go:build !production

package main

// devBuild is true for builds made without the production tag, such as
// `wails3 dev`. Only these get the web inspector without -debug-ui.
const devBuild = true
