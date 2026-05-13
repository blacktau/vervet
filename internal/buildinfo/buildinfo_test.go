package buildinfo

import "testing"

func TestChannelDefaultsToGitHub(t *testing.T) {
	// Save and restore package state around the test
	prev := channel
	defer func() { channel = prev }()

	channel = ""
	if got := Channel(); got != ChannelGitHub {
		t.Errorf("Channel() with empty var = %q, want %q", got, ChannelGitHub)
	}
}

func TestChannelMSStore(t *testing.T) {
	prev := channel
	defer func() { channel = prev }()

	channel = "msstore"
	if got := Channel(); got != ChannelMSStore {
		t.Errorf("Channel() = %q, want %q", got, ChannelMSStore)
	}
	if !IsMSStore() {
		t.Error("IsMSStore() = false, want true")
	}
}

func TestChannelUnknownFallsBackToGitHub(t *testing.T) {
	prev := channel
	defer func() { channel = prev }()

	channel = "garbage"
	if got := Channel(); got != ChannelGitHub {
		t.Errorf("Channel() with garbage = %q, want %q", got, ChannelGitHub)
	}
}
