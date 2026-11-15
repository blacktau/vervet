package updates

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Manifest is the update manifest read by the Wails updater's endpoint
// provider (schema version 1). CI writes one per release with
// cmd/release-manifest and publishes it as a release asset.
type Manifest struct {
	SchemaVersion int                `json:"schemaVersion"`
	Version       string             `json:"version"`
	Name          string             `json:"name"`
	PublishedAt   string             `json:"publishedAt"`
	Artifacts     []ManifestArtifact `json:"artifacts"`
}

type ManifestArtifact struct {
	URL           string `json:"url"`
	Filename      string `json:"filename"`
	Size          int64  `json:"size"`
	Platform      string `json:"platform"`
	Arch          string `json:"arch"`
	DigestAlgo    string `json:"digestAlgo"`
	Digest        string `json:"digest"`
	SignatureAlgo string `json:"signatureAlgo"`
	Signature     string `json:"signature"`
}

// Release assets the app can install itself. Installers, deb and rpm are
// left out: a Windows installer build updates from the portable zip (the
// installer only places Vervet.exe), and package-manager builds don't
// self-update.
var updatableAssets = []struct {
	pattern  *regexp.Regexp
	platform string
}{
	{regexp.MustCompile(`^Vervet-linux-(amd64|arm64)\.AppImage$`), "linux"},
	{regexp.MustCompile(`^Vervet-windows-(amd64|arm64)-portable\.zip$`), "windows"},
	{regexp.MustCompile(`^Vervet-macos-(amd64|arm64)\.app\.zip$`), "darwin"},
}

// ArtifactTarget reports the platform and arch of a release asset, or ok=false
// when the app can't install that asset itself.
func ArtifactTarget(filename string) (platform, arch string, ok bool) {
	for _, a := range updatableAssets {
		if m := a.pattern.FindStringSubmatch(filename); m != nil {
			return a.platform, m[1], true
		}
	}
	return "", "", false
}

// GenerateSigningKey returns a new key pair, both base64-encoded. The private
// key is the 32-byte seed.
func GenerateSigningKey() (private, public string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(priv.Seed()), base64.StdEncoding.EncodeToString(pub), nil
}

// ParseSigningKey decodes a base64 private key written by GenerateSigningKey.
func ParseSigningKey(s string) (ed25519.PrivateKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return nil, fmt.Errorf("signing key is not base64: %w", err)
	}
	if len(raw) != ed25519.SeedSize {
		return nil, fmt.Errorf("signing key is %d bytes, want %d", len(raw), ed25519.SeedSize)
	}
	return ed25519.NewKeyFromSeed(raw), nil
}

// SignArtifact hashes r with SHA-512 and signs the hash with Ed25519ph, the
// scheme the Wails updater verifies for "ed25519ph".
func SignArtifact(key ed25519.PrivateKey, r io.Reader) (digest, signature []byte, size int64, err error) {
	h := sha512.New()
	size, err = io.Copy(h, r)
	if err != nil {
		return nil, nil, 0, err
	}
	digest = h.Sum(nil)
	signature, err = key.Sign(nil, digest, &ed25519.Options{Hash: crypto.SHA512})
	if err != nil {
		return nil, nil, 0, err
	}
	return digest, signature, size, nil
}

// BuildManifest signs every installable file in paths and returns the
// manifest. baseURL is the directory the assets are downloaded from, e.g.
// https://github.com/blacktau/vervet/releases/download/v2026.11.2. Files the
// app can't install are skipped; finding none is an error.
func BuildManifest(version, baseURL string, key ed25519.PrivateKey, paths []string, now time.Time) (Manifest, error) {
	if !IsReleaseVersion(version) {
		return Manifest{}, fmt.Errorf("%q is not a release version", version)
	}
	m := Manifest{
		SchemaVersion: 1,
		Version:       SemverVersion(version),
		Name:          strings.TrimPrefix(version, "v"),
		PublishedAt:   now.UTC().Format(time.RFC3339),
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	for _, p := range paths {
		name := filepath.Base(p)
		platform, arch, ok := ArtifactTarget(name)
		if !ok {
			continue
		}
		art, err := signFile(key, p)
		if err != nil {
			return Manifest{}, fmt.Errorf("%s: %w", name, err)
		}
		art.URL = baseURL + "/" + name
		art.Filename = name
		art.Platform = platform
		art.Arch = arch
		m.Artifacts = append(m.Artifacts, art)
	}
	if len(m.Artifacts) == 0 {
		return Manifest{}, errors.New("no installable artifacts found")
	}
	return m, nil
}

func signFile(key ed25519.PrivateKey, path string) (ManifestArtifact, error) {
	f, err := os.Open(path)
	if err != nil {
		return ManifestArtifact{}, err
	}
	defer f.Close()
	digest, sig, size, err := SignArtifact(key, f)
	if err != nil {
		return ManifestArtifact{}, err
	}
	return ManifestArtifact{
		Size:          size,
		DigestAlgo:    "sha512",
		Digest:        base64.StdEncoding.EncodeToString(digest),
		SignatureAlgo: "ed25519ph",
		Signature:     base64.StdEncoding.EncodeToString(sig),
	}, nil
}
