package oidc

import (
	"testing"
)

func TestGeneratePKCE(t *testing.T) {
	verifier, challenge, err := generatePKCE()
	if err != nil {
		t.Fatalf("generatePKCE: %v", err)
	}
	if len(verifier) < 43 {
		t.Errorf("verifier too short: %d", len(verifier))
	}
	if len(challenge) == 0 {
		t.Error("challenge is empty")
	}
	if verifier == challenge {
		t.Error("verifier and challenge should differ")
	}
}

func TestGenerateState(t *testing.T) {
	state, err := generateState()
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}
	if len(state) < 16 {
		t.Errorf("state too short: %d", len(state))
	}
}
