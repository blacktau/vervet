// Package desktop is the only place Vervet's services reach the Wails runtime.
//
// The Wails app and its window are created after the services are built, so a
// Runtime starts detached and is attached once both exist. Until then every
// method is a safe no-op, which also keeps services usable in plain Go tests.
package desktop

import (
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// FileFilter restricts a file dialog to matching files, e.g. "*.js;*.mongodb".
type FileFilter struct {
	DisplayName string
	Pattern     string
}

// Runtime wraps the Wails app and Vervet's main window.
type Runtime struct {
	mu     sync.RWMutex
	app    *application.App
	window *application.WebviewWindow
}

// New returns a Runtime that is not yet attached to a Wails app.
func New() *Runtime {
	return &Runtime{}
}

// Attach connects the Runtime to the running app and its main window.
func (r *Runtime) Attach(app *application.App, window *application.WebviewWindow) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.app = app
	r.window = window
}

func (r *Runtime) get() (*application.App, *application.WebviewWindow) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.app, r.window
}

// Emit sends an event to the frontend. With one data argument the frontend
// receives it as the event's data; with several it receives them as an array.
func (r *Runtime) Emit(name string, data ...any) {
	app, _ := r.get()
	if app == nil {
		return
	}
	app.Event.Emit(name, data...)
}

// OpenURL opens url in the user's default browser.
func (r *Runtime) OpenURL(url string) error {
	app, _ := r.get()
	if app == nil {
		return nil
	}
	return app.Browser.OpenURL(url)
}

// SelectFile asks the user to pick an existing file. A cancelled dialog
// returns an empty path and no error.
func (r *Runtime) SelectFile(title string, filters []FileFilter) (string, error) {
	app, window := r.get()
	if app == nil {
		return "", nil
	}
	dialog := app.Dialog.OpenFile().
		SetTitle(title).
		CanChooseFiles(true).
		ShowHiddenFiles(true)
	if window != nil {
		dialog.AttachToWindow(window)
	}
	for _, f := range filters {
		dialog.AddFilter(f.DisplayName, f.Pattern)
	}
	return dialog.PromptForSingleSelection()
}

// SelectDirectory asks the user to pick a directory. A cancelled dialog
// returns an empty path and no error.
func (r *Runtime) SelectDirectory() (string, error) {
	app, window := r.get()
	if app == nil {
		return "", nil
	}
	dialog := app.Dialog.OpenFile().
		CanChooseFiles(false).
		CanChooseDirectories(true).
		CanCreateDirectories(true)
	if window != nil {
		dialog.AttachToWindow(window)
	}
	return dialog.PromptForSingleSelection()
}

// SaveFile asks the user where to save a file. A cancelled dialog returns an
// empty path and no error.
func (r *Runtime) SaveFile(title, defaultFilename string, filters []FileFilter) (string, error) {
	app, window := r.get()
	if app == nil {
		return "", nil
	}
	options := &application.SaveFileDialogOptions{
		Title:                title,
		Filename:             defaultFilename,
		CanCreateDirectories: true,
	}
	for _, f := range filters {
		options.Filters = append(options.Filters, application.FileFilter{DisplayName: f.DisplayName, Pattern: f.Pattern})
	}
	// A nil *WebviewWindow in the Window interface would not compare equal to nil.
	if window != nil {
		options.Window = window
	}
	return app.Dialog.SaveFileWithOptions(options).PromptForSingleSelection()
}

// ScreenSize returns the size of the screen showing the main window, or
// ok=false when it isn't known yet.
func (r *Runtime) ScreenSize() (width, height int, ok bool) {
	_, window := r.get()
	if window == nil {
		return 0, 0, false
	}
	screen, err := window.GetScreen()
	if err != nil || screen == nil {
		return 0, 0, false
	}
	return screen.Size.Width, screen.Size.Height, true
}

// RevealWindow brings the main window back if it is minimised.
func (r *Runtime) RevealWindow() {
	_, window := r.get()
	if window == nil {
		return
	}
	if window.IsMinimised() {
		window.UnMinimise()
	}
}

// CloseWindow closes the main window, which quits the app.
func (r *Runtime) CloseWindow() {
	_, window := r.get()
	if window == nil {
		return
	}
	window.Close()
}

// OnWindowClosing calls fn when the user tries to close the main window. If fn
// returns true, the window stays open.
func (r *Runtime) OnWindowClosing(fn func() (keepOpen bool)) {
	_, window := r.get()
	if window == nil {
		return
	}
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if fn() {
			e.Cancel()
		}
	})
}

// WindowBounds returns the main window's position and size, or ok=false when
// there is no window or it is already gone.
func (r *Runtime) WindowBounds() (x, y, width, height int, ok bool) {
	_, window := r.get()
	if window == nil {
		return 0, 0, 0, 0, false
	}
	width, height = window.Size()
	if width <= 0 || height <= 0 {
		return 0, 0, 0, 0, false
	}
	x, y = window.Position()
	return x, y, width, height, true
}
