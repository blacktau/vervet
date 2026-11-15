package api

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeBrowser struct {
	opened []string
	err    error
}

func (b *fakeBrowser) OpenURL(url string) error {
	if b.err != nil {
		return b.err
	}
	b.opened = append(b.opened, url)
	return nil
}

func TestSystemProxyOpenURL(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("opens the link", func(t *testing.T) {
		browser := &fakeBrowser{}
		result := NewSystemProxy(log, nil, browser).OpenURL("https://github.com/blacktau/vervet")
		assert.True(t, result.IsSuccess)
		assert.Equal(t, []string{"https://github.com/blacktau/vervet"}, browser.opened)
	})

	t.Run("reports a refused link", func(t *testing.T) {
		browser := &fakeBrowser{err: errors.New(`refusing to open a "smb" URL in the browser`)}
		result := NewSystemProxy(log, nil, browser).OpenURL("smb://attacker.example/share")
		assert.False(t, result.IsSuccess)
	})
}
