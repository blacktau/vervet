package updates

import (
	"context"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type BrowserOpener struct {
	ctx context.Context
}

func NewBrowserOpener(ctx context.Context) *BrowserOpener {
	return &BrowserOpener{ctx: ctx}
}

func (b *BrowserOpener) SetContext(ctx context.Context) {
	b.ctx = ctx
}

func (b *BrowserOpener) OpenURL(url string) {
	wailsRuntime.BrowserOpenURL(b.ctx, url)
}
