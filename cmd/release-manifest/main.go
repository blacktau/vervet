// Command release-manifest creates the signing key for in-app updates and
// writes the signed update manifest for a release.
//
//	go run ./cmd/release-manifest keygen -out signing-key.txt
//	UPDATE_SIGNING_KEY=... go run ./cmd/release-manifest sign \
//	    -version 2026.11.2 \
//	    -base-url https://github.com/blacktau/vervet/releases/download/v2026.11.2 \
//	    -out manifest.json artifacts/*
//
// keygen writes the private key to -out (mode 0600) and prints the public key,
// which goes in internal/updates/publickey.go. The private key goes in the
// UPDATE_SIGNING_KEY secret of the release environment.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"vervet/internal/updates"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "release-manifest:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: release-manifest keygen|sign [flags]")
	}
	switch args[0] {
	case "keygen":
		return keygen(args[1:])
	case "sign":
		return sign(args[1:])
	}
	return fmt.Errorf("unknown command %q", args[0])
}

func keygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	out := fs.String("out", "", "file to write the private key to (must not exist)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return errors.New("-out is required")
	}
	priv, pub, err := updates.GenerateSigningKey()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(f, priv); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Println(pub)
	return nil
}

func sign(args []string) error {
	fs := flag.NewFlagSet("sign", flag.ContinueOnError)
	version := fs.String("version", "", "release version, e.g. 2026.11.2")
	baseURL := fs.String("base-url", "", "URL of the directory the assets download from")
	out := fs.String("out", "manifest.json", "manifest file to write")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *version == "" || *baseURL == "" {
		return errors.New("-version and -base-url are required")
	}
	keyText := os.Getenv("UPDATE_SIGNING_KEY")
	if keyText == "" {
		return errors.New("UPDATE_SIGNING_KEY is not set")
	}
	key, err := updates.ParseSigningKey(keyText)
	if err != nil {
		return err
	}
	m, err := updates.BuildManifest(*version, *baseURL, key, fs.Args(), time.Now())
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(*out, append(data, '\n'), 0o644); err != nil {
		return err
	}
	for _, a := range m.Artifacts {
		fmt.Printf("signed %s (%s/%s)\n", a.Filename, a.Platform, a.Arch)
	}
	return nil
}
