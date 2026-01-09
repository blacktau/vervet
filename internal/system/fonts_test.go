package system

import (
	"fmt"
	"log/slog"
	"testing"
)

func newSUT(log *slog.Logger) FontService {
	if log == nil {
		log = slog.Default()
	}
	return NewFontService(log)
}

func TestFontService_GetInstalledFonts(t *testing.T) {
	service := newSUT(nil)
	fonts := service.GetInstalledFonts()

	// Since we are running on a real system during tests,
	// we expect at least some fonts or an empty slice (not nil).
	if fonts == nil {
		t.Fatal("expected fonts slice to be initialized, got nil")
	}

	fmt.Printf("Fonts: %v\n", fonts)
}