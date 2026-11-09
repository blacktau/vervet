// Package desktop is the only place Vervet's services reach the Wails runtime.
//
// The Wails app and its window are created after the services are built, so a
// Runtime starts detached and is attached once both exist. Until then every
// method is a safe no-op, which also keeps services usable in plain Go tests.
package desktop

import (
	"fmt"
	"net/url"
	"strings"
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
	// lastBounds is the window's position and size when it was last not
	// minimised. Windows reports a minimised window at (-32000, -32000).
	lastBounds *bounds
}

type bounds struct{ x, y, width, height int }

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
	if window == nil {
		return
	}
	// Listeners run on their own goroutine, so they may query the window.
	window.OnWindowEvent(events.Common.WindowDidResize, func(*application.WindowEvent) { r.recordBounds() })
	window.OnWindowEvent(events.Common.WindowDidMove, func(*application.WindowEvent) { r.recordBounds() })
}

func (r *Runtime) recordBounds() {
	_, window := r.get()
	if window == nil || window.IsMinimised() {
		return
	}
	b, ok := liveBounds(window)
	if !ok {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastBounds = &b
}

func liveBounds(window *application.WebviewWindow) (bounds, bool) {
	width, height := window.Size()
	if width <= 0 || height <= 0 {
		return bounds{}, false
	}
	x, y := window.Position()
	return bounds{x, y, width, height}, true
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

// OpenURL opens an http or https URL in the user's default browser. Other
// schemes are refused: some URLs come from MongoDB servers (the OIDC login
// page), and the OS opener would launch file://, smb:// or any handler app.
func (r *Runtime) OpenURL(rawURL string) error {
	safe, err := checkBrowserURL(rawURL)
	if err != nil {
		return err
	}
	app, _ := r.get()
	if app == nil {
		return nil
	}
	return app.Browser.OpenURL(safe)
}

func checkBrowserURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("refusing to open URL: %w", err)
	}
	if scheme := strings.ToLower(parsed.Scheme); scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("refusing to open a %q URL in the browser", parsed.Scheme)
	}
	safe, err := application.ValidateAndSanitizeURL(rawURL)
	if err != nil {
		return "", fmt.Errorf("refusing to open URL: %w", err)
	}
	return safe, nil
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
// there is no window or it is already gone. A minimised window reports where
// it was before it was minimised.
func (r *Runtime) WindowBounds() (x, y, width, height int, ok bool) {
	_, window := r.get()
	if window == nil {
		return 0, 0, 0, 0, false
	}
	if window.IsMinimised() {
		r.mu.RLock()
		last := r.lastBounds
		r.mu.RUnlock()
		if last == nil {
			return 0, 0, 0, 0, false
		}
		return last.x, last.y, last.width, last.height, true
	}
	b, ok := liveBounds(window)
	if !ok {
		return 0, 0, 0, 0, false
	}
	return b.x, b.y, b.width, b.height, true
}
