package api

import (
	"log/slog"
	"testing"

	"vervet/internal/buildinfo"
)

func TestBuildInfoProxyGetChannel(t *testing.T) {
	proxy := NewBuildInfoProxy(slog.Default())
	result := proxy.GetChannel()
	if !result.IsSuccess {
		t.Fatalf("GetChannel not success: %+v", result)
	}
	want := string(buildinfo.Channel())
	if result.Data != want {
		t.Errorf("GetChannel = %q, want %q", result.Data, want)
	}
}
