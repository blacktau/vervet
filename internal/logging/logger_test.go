package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestLogAdapter_Levels(t *testing.T) {
	var buf bytes.Buffer
	// Create a handler that writes to our buffer in a simple text format
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)
	adapter := NewLogger(logger)

	tests := []struct {
		name     string
		logFunc  func(string)
		msg      string
		expected string
	}{
		{"Print", adapter.Print, "print msg", "level=INFO msg=\"print msg\""},
		{"Trace", adapter.Trace, "trace msg", "level=DEBUG msg=\"trace msg\""},
		{"Debug", adapter.Debug, "debug msg", "level=DEBUG msg=\"debug msg\""},
		{"Info", adapter.Info, "info msg", "level=INFO msg=\"info msg\""},
		{"Warning", adapter.Warning, "warning msg", "level=WARN msg=\"warning msg\""},
		{"Error", adapter.Error, "error msg", "level=ERROR msg=\"error msg\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc(tt.msg)
			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("%s: expected output to contain %q, got %q", tt.name, tt.expected, output)
			}
		})
	}
}

func TestLogAdapter_Fatal(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)
	adapter := NewLogger(logger)

	msg := "fatal error occurred"

	defer func() {
		if r := recover(); r != nil {
			if r != msg {
				t.Errorf("expected panic with %q, got %v", msg, r)
			}

			// Verify it was logged as ERROR before panicking
			output := buf.String()
			if !strings.Contains(output, "level=ERROR") || !strings.Contains(output, msg) {
				t.Errorf("expected fatal message to be logged as ERROR, got %q", output)
			}
		} else {
			t.Error("expected Fatal to panic, but it did not")
		}
	}()

	adapter.Fatal(msg)
}
