package closeguard

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type harness struct {
	guard   *Guard
	emitted chan struct{}
	quits   int
	clock   time.Time
}

func newHarness() *harness {
	h := &harness{emitted: make(chan struct{}, 10), clock: time.Unix(1000, 0)}
	h.guard = New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	h.guard.now = func() time.Time { return h.clock }
	h.guard.Init(func() { h.emitted <- struct{}{} }, func() { h.quits++ })
	return h
}

func (h *harness) emitCount(t *testing.T, want int) {
	t.Helper()
	for range want {
		select {
		case <-h.emitted:
		case <-time.After(time.Second):
			t.Fatalf("expected %d close requests", want)
		}
	}
	select {
	case <-h.emitted:
		t.Fatalf("expected only %d close requests", want)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestBeforeClose_AllowsWithoutUnsavedChanges(t *testing.T) {
	h := newHarness()

	assert.False(t, h.guard.BeforeClose())
	h.emitCount(t, 0)
}

func TestBeforeClose_AllowsBeforeInit(t *testing.T) {
	guard := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	guard.SetUnsavedChanges(true)

	assert.False(t, guard.BeforeClose())
}

func TestBeforeClose_AllowsOnceChangesAreSaved(t *testing.T) {
	h := newHarness()
	h.guard.SetUnsavedChanges(true)
	h.guard.SetUnsavedChanges(false)

	assert.False(t, h.guard.BeforeClose())
}

func TestBeforeClose_AsksOnceWithUnsavedChanges(t *testing.T) {
	h := newHarness()
	h.guard.SetUnsavedChanges(true)

	assert.True(t, h.guard.BeforeClose())
	assert.True(t, h.guard.BeforeClose())
	h.emitCount(t, 1)
}

func TestBeforeClose_ClosesWhenFrontendNeverAnswers(t *testing.T) {
	h := newHarness()
	h.guard.SetUnsavedChanges(true)
	assert.True(t, h.guard.BeforeClose())

	h.clock = h.clock.Add(unansweredTimeout - time.Second)
	assert.True(t, h.guard.BeforeClose(), "still within the timeout")

	h.clock = h.clock.Add(time.Second)
	assert.False(t, h.guard.BeforeClose())
}

func TestBeforeClose_WaitsForTheUserHoweverLongTheyTake(t *testing.T) {
	h := newHarness()
	h.guard.SetUnsavedChanges(true)
	assert.True(t, h.guard.BeforeClose())
	h.guard.AcknowledgeCloseRequest()

	h.clock = h.clock.Add(time.Hour)

	assert.True(t, h.guard.BeforeClose())
	h.emitCount(t, 1)
}

func TestCancelClose_AsksAgainNextTime(t *testing.T) {
	h := newHarness()
	h.guard.SetUnsavedChanges(true)
	h.guard.BeforeClose()
	h.guard.AcknowledgeCloseRequest()

	h.guard.CancelClose()

	assert.True(t, h.guard.BeforeClose())
	h.emitCount(t, 2)
}

func TestConfirmClose_QuitsAndLetsTheCloseThrough(t *testing.T) {
	h := newHarness()
	h.guard.SetUnsavedChanges(true)
	h.guard.BeforeClose()
	h.guard.AcknowledgeCloseRequest()

	h.guard.ConfirmClose()

	assert.Equal(t, 1, h.quits)
	assert.False(t, h.guard.BeforeClose())
}
