package logging_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"vervet/internal/logging"
)

func TestMultiHandler_FansOut(t *testing.T) {
	var a, b bytes.Buffer
	ha := slog.NewTextHandler(&a, &slog.HandlerOptions{Level: slog.LevelDebug})
	hb := slog.NewTextHandler(&b, &slog.HandlerOptions{Level: slog.LevelDebug})
	log := slog.New(logging.NewMultiHandler(ha, hb))

	log.Info("hello", slog.String("k", "v"))

	if !strings.Contains(a.String(), "hello") || !strings.Contains(a.String(), "k=v") {
		t.Errorf("handler A missing log: %q", a.String())
	}
	if !strings.Contains(b.String(), "hello") || !strings.Contains(b.String(), "k=v") {
		t.Errorf("handler B missing log: %q", b.String())
	}
}

func TestMultiHandler_Enabled(t *testing.T) {
	var buf bytes.Buffer
	warnOnly := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	mh := logging.NewMultiHandler(warnOnly)

	if mh.Enabled(context.Background(), slog.LevelDebug) {
		t.Errorf("expected debug not enabled when all children are warn-only")
	}
	if !mh.Enabled(context.Background(), slog.LevelError) {
		t.Errorf("expected error enabled")
	}
}

func TestMultiHandler_WithAttrs(t *testing.T) {
	var a bytes.Buffer
	ha := slog.NewTextHandler(&a, &slog.HandlerOptions{Level: slog.LevelDebug})
	log := slog.New(logging.NewMultiHandler(ha)).With(slog.String("service", "test"))
	log.Info("msg")
	if !strings.Contains(a.String(), "service=test") {
		t.Errorf("expected service attr, got %q", a.String())
	}
}

func TestMultiHandler_NoHandlersNeverEnabled(t *testing.T) {
	mh := logging.NewMultiHandler()
	if mh.Enabled(context.Background(), slog.LevelError) {
		t.Errorf("empty MultiHandler must never be enabled")
	}
}
