package api

import (
	"context"
	"log/slog"

	"vervet/internal/updates"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type UpdatesService interface {
	CheckNow(ctx context.Context) (updates.UpdateInfo, error)
	CheckIfDue(ctx context.Context) error
	DismissVersion(version string) error
}

// URLOpener opens web links. It refuses anything but http(s), so a link in a
// document can't launch file://, smb:// or a custom URL handler.
type URLOpener interface {
	OpenURL(url string) error
}

type UpdatesProxy struct {
	log    *slog.Logger
	svc    UpdatesService
	opener URLOpener
	ctx    context.Context
}

func NewUpdatesProxy(log *slog.Logger, svc UpdatesService, opener URLOpener) *UpdatesProxy {
	return &UpdatesProxy{log: log, svc: svc, opener: opener, ctx: context.Background()}
}

// ServiceStartup is called by Wails when the app starts; the context lasts
// until shutdown. It is not exposed to the frontend.
func (p *UpdatesProxy) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	p.ctx = ctx
	return nil
}

func (p *UpdatesProxy) CheckNow() Result[updates.UpdateInfo] {
	info, err := p.svc.CheckNow(p.ctx)
	if err != nil {
		logFail(p.log, "CheckNow", err)
		return FailResult[updates.UpdateInfo](err)
	}
	return SuccessResult(info)
}

func (p *UpdatesProxy) CheckIfDue() EmptyResult {
	if err := p.svc.CheckIfDue(p.ctx); err != nil {
		logFail(p.log, "CheckIfDue", err)
		return Fail(err)
	}
	return Success()
}

func (p *UpdatesProxy) DismissUpdate(version string) EmptyResult {
	if err := p.svc.DismissVersion(version); err != nil {
		logFail(p.log, "DismissUpdate", err)
		return Fail(err)
	}
	return Success()
}

func (p *UpdatesProxy) OpenReleasePage(url string) EmptyResult {
	if err := p.opener.OpenURL(url); err != nil {
		logFail(p.log, "OpenReleasePage", err)
		return Fail(err)
	}
	return Success()
}
