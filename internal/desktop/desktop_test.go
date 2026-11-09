package desktop

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckBrowserURL(t *testing.T) {
	allowed := []string{
		"https://github.com/blacktau/vervet",
		"http://localhost:8080/auth?state=abc&code_challenge=xyz",
		"HTTPS://login.example.com/authorize",
	}
	for _, u := range allowed {
		_, err := checkBrowserURL(u)
		assert.NoError(t, err, u)
	}

	refused := []string{
		"file:///etc/passwd",
		"smb://attacker.example/share",
		"vscode://extension/install",
		"javascript:alert(1)",
		"https://example.com/$(reboot)",
		"https:///no-host",
		"not a url",
	}
	for _, u := range refused {
		_, err := checkBrowserURL(u)
		assert.Error(t, err, u)
	}
}

func TestOpenURLRefusesBeforeTheAppIsAttached(t *testing.T) {
	assert.Error(t, New().OpenURL("file:///etc/passwd"))
	assert.NoError(t, New().OpenURL("https://github.com"))
}
