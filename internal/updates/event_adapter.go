package updates

import (
	"context"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type WailsEmitter struct {
	ctx context.Context
}

func NewWailsEmitter(ctx context.Context) *WailsEmitter {
	return &WailsEmitter{ctx: ctx}
}

func (e *WailsEmitter) SetContext(ctx context.Context) {
	e.ctx = ctx
}

func (e *WailsEmitter) EmitEvent(name string, data any) {
	wailsRuntime.EventsEmit(e.ctx, name, data)
}
