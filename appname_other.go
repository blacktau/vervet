//go:build !linux || purego

package main

// setApplicationName only matters on Linux, where GTK titles the window with
// it. purego builds skip it, as they link no GTK headers.
func setApplicationName(string) {}
