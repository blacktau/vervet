//go:build linux && !purego

package main

/*
#cgo pkg-config: glib-2.0
#include <glib.h>
#include <stdlib.h>
*/
import "C"

import "unsafe"

// setApplicationName names the app for GTK. Wails doesn't title a frameless
// window on Linux, so GTK titles it with the application name, which otherwise
// falls back to the program name, com.blacktau.vervet.
func setApplicationName(name string) {
	cName := C.CString(name)
	C.g_set_application_name(cName)
	C.free(unsafe.Pointer(cName))
}
