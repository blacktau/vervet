// Package closeguard decides whether the application may close, so that closing
// the window can't silently drop unsaved queries. Only the frontend knows about
// those, so it reports them here and answers when asked.
package closeguard

import (
	"log/slog"
	"sync"
	"time"
)

// EventCloseRequested is emitted when closing needs the frontend to ask the user
// about unsaved queries.
const EventCloseRequested = "app-close-requested"

// unansweredTimeout is how long the frontend has to acknowledge a request before
// another attempt to close goes ahead. It stops a hung webview trapping the user
// in an application they can't close.
const unansweredTimeout = 5 * time.Second

// Guard tracks the state of a request to close the application.
type Guard struct {
	log *slog.Logger
	now func() time.Time

	mu          sync.Mutex
	emit        func()
	quit        func()
	unsaved     bool
	approved    bool
	prompting   bool
	requestedAt time.Time
}

// New returns a Guard that allows closing until the frontend reports unsaved
// queries.
func New(log *slog.Logger) *Guard {
	return &Guard{log: log.With(slog.String("source", "CloseGuard")), now: time.Now}
}

// Init sets how to ask the frontend about unsaved queries, and how to quit.
func (g *Guard) Init(emit func(), quit func()) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.emit = emit
	g.quit = quit
}

// BeforeClose reports whether closing should be prevented for now, asking the
// frontend to prompt the user if it hasn't been asked already.
func (g *Guard) BeforeClose() (prevent bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.approved || !g.unsaved || g.emit == nil {
		return false
	}
	if g.prompting {
		return true
	}
	if !g.requestedAt.IsZero() {
		if g.now().Sub(g.requestedAt) < unansweredTimeout {
			return true
		}
		g.log.Warn("Frontend did not answer a request to close; closing anyway")
		return false
	}

	g.requestedAt = g.now()
	// Not called inline: this runs on the main thread on Windows, where emitting
	// waits for that same thread.
	go g.emit()
	return true
}

// SetUnsavedChanges records whether any query has unsaved changes.
func (g *Guard) SetUnsavedChanges(unsaved bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.unsaved = unsaved
}

// AcknowledgeCloseRequest records that the frontend is asking the user.
func (g *Guard) AcknowledgeCloseRequest() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.prompting = true
	g.requestedAt = time.Time{}
}

// CancelClose records that the user chose to keep the application open.
func (g *Guard) CancelClose() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.prompting = false
	g.requestedAt = time.Time{}
}

// ConfirmClose records that the user dealt with their unsaved queries, and quits.
func (g *Guard) ConfirmClose() {
	g.mu.Lock()
	g.approved = true
	quit := g.quit
	g.mu.Unlock()

	if quit != nil {
		quit()
	}
}
