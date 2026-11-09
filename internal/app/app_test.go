package app

import (
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"vervet/internal/closeguard"
	"vervet/internal/models"
	"vervet/internal/settings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type fakeWindow struct {
	mu        sync.Mutex
	events    []string
	revealed  bool
	closed    bool
	onClosing func() bool
	bounds    [4]int
	gone      bool
}

func (w *fakeWindow) Attach(*application.App, *application.WebviewWindow) {}

func (w *fakeWindow) Emit(name string, _ ...any) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.events = append(w.events, name)
}

func (w *fakeWindow) RevealWindow() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.revealed = true
}

func (w *fakeWindow) CloseWindow() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
}

func (w *fakeWindow) OnWindowClosing(fn func() bool) { w.onClosing = fn }

func (w *fakeWindow) WindowBounds() (x, y, width, height int, ok bool) {
	if w.gone {
		return 0, 0, 0, 0, false
	}
	return w.bounds[0], w.bounds[1], w.bounds[2], w.bounds[3], true
}

func (w *fakeWindow) emitted() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.events...)
}

// fakeSettings records saved window states; other methods are unused here.
type fakeSettings struct {
	settings.Service
	saved []models.WindowState
}

func (s *fakeSettings) SaveWindowState(state models.WindowState) error {
	s.saved = append(s.saved, state)
	return nil
}

func newGuardedApp(t *testing.T) (*App, *fakeWindow, *fakeSettings) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	win := &fakeWindow{bounds: [4]int{10, 20, 1200, 800}}
	store := &fakeSettings{}
	a := &App{
		log:             log,
		closeGuard:      closeguard.New(log),
		runtime:         win,
		settingsService: store,
	}
	a.guardWindow()
	require.NotNil(t, win.onClosing, "the closing hook should be registered")
	return a, win, store
}

func TestCloseWithNothingUnsavedSavesWindowStateAndCloses(t *testing.T) {
	_, win, store := newGuardedApp(t)

	keepOpen := win.onClosing()

	assert.False(t, keepOpen)
	assert.Equal(t, []models.WindowState{{X: 10, Y: 20, Width: 1200, Height: 800}}, store.saved)
	assert.Empty(t, win.emitted())
}

func TestCloseWithUnsavedQueriesAsksTheFrontend(t *testing.T) {
	a, win, store := newGuardedApp(t)
	a.closeGuard.SetUnsavedChanges(true)

	keepOpen := win.onClosing()

	assert.True(t, keepOpen)
	assert.Empty(t, store.saved, "window state is saved only when the window really closes")
	assert.Eventually(t, func() bool {
		return len(win.emitted()) == 1 && win.emitted()[0] == closeguard.EventCloseRequested
	}, time.Second, 10*time.Millisecond)
	win.mu.Lock()
	defer win.mu.Unlock()
	assert.True(t, win.revealed, "a minimised window is restored so the prompt can be seen")
}

func TestConfirmingCloseClosesTheWindow(t *testing.T) {
	a, win, _ := newGuardedApp(t)
	a.closeGuard.SetUnsavedChanges(true)
	win.onClosing()

	a.closeGuard.ConfirmClose()

	win.mu.Lock()
	assert.True(t, win.closed)
	win.mu.Unlock()
	assert.False(t, win.onClosing(), "the close that follows confirmation goes ahead")
}

func TestQuitAfterTheWindowIsGoneDoesNotSaveWindowState(t *testing.T) {
	a, win, store := newGuardedApp(t)
	win.gone = true

	assert.True(t, a.ShouldQuit())
	assert.Empty(t, store.saved)
}
