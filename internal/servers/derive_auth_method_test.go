package servers

import "testing"

func TestDeriveAuthMethod_GSSAPI(t *testing.T) {
	cfg := &exportConnectionConfig{URI: "mongodb://user@host/?authMechanism=GSSAPI"}
	got := deriveAuthMethod(cfg)
	if got != "gssapi" {
		t.Fatalf("expected gssapi, got %q", got)
	}
}

func TestDeriveAuthMethod_PLAIN(t *testing.T) {
	cfg := &exportConnectionConfig{URI: "mongodb://user@host/?authMechanism=PLAIN"}
	got := deriveAuthMethod(cfg)
	if got != "plain" {
		t.Fatalf("expected plain, got %q", got)
	}
}

func TestDeriveAuthMethod_X509(t *testing.T) {
	cfg := &exportConnectionConfig{URI: "mongodb://host/?authMechanism=MONGODB-X509"}
	got := deriveAuthMethod(cfg)
	if got != "x509" {
		t.Fatalf("expected x509, got %q", got)
	}
}

func TestDeriveAuthMethod_AWS(t *testing.T) {
	cfg := &exportConnectionConfig{URI: "mongodb://AKIA:secret@host/?authMechanism=MONGODB-AWS"}
	got := deriveAuthMethod(cfg)
	if got != "aws" {
		t.Fatalf("expected aws, got %q", got)
	}
}

func TestDeriveAuthMethod_OIDC(t *testing.T) {
	cfg := &exportConnectionConfig{URI: "mongodb://host/?authMechanism=MONGODB-OIDC"}
	got := deriveAuthMethod(cfg)
	if got != "oidc" {
		t.Fatalf("expected oidc, got %q", got)
	}
}

func TestDeriveAuthMethod_Password(t *testing.T) {
	cfg := &exportConnectionConfig{URI: "mongodb://user:pass@host/"}
	got := deriveAuthMethod(cfg)
	if got != "password" {
		t.Fatalf("expected password, got %q", got)
	}
}

func TestDeriveAuthMethod_None(t *testing.T) {
	cfg := &exportConnectionConfig{URI: "mongodb://host/"}
	got := deriveAuthMethod(cfg)
	if got != "none" {
		t.Fatalf("expected none, got %q", got)
	}
}
