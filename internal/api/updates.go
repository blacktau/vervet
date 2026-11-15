package api

import (
	"context"
	"log/slog"

	"vervet/internal/updates"
)

type UpdatesService interface {
	CheckNow(ctx context.Context) (updates.UpdateInfo, error)
	CheckIfDue(ctx context.Context) error
	DismissVersion(version string) error
	Install(ctx context.Context) error
}

// URLOpener opens web links. It refuses anything but http(s), so a link in a
// document can't launch file://, smb:// or a custom URL handler.
type URLOpener interface {
	OpenURL(url string) error
}

type UpdatesProxy struct {
	log     *slog.Logger
	svc     UpdatesService
	opener  URLOpener
	ctx     context.Context
	restart func() error
}

// NewUpdatesProxy takes the app's context, which lasts until the app has shut
// down. Update checks run with it. restart installs a downloaded update once
// unsaved queries have been dealt with.
func NewUpdatesProxy(ctx context.Context, log *slog.Logger, svc UpdatesService, opener URLOpener, restart func() error) *UpdatesProxy {
	return &UpdatesProxy{log: log, svc: svc, opener: opener, ctx: ctx, restart: restart}
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

// InstallUpdate downloads and verifies the newest release. Progress arrives
// as wails:updater:* events. Call RestartToUpdate when it succeeds.
func (p *UpdatesProxy) InstallUpdate() EmptyResult {
	if err := p.svc.Install(p.ctx); err != nil {
		logFail(p.log, "InstallUpdate", err)
		return Fail(err)
	}
	return Success()
}

// RestartToUpdate asks about unsaved queries, as closing does, then restarts
// into the downloaded update.
func (p *UpdatesProxy) RestartToUpdate() EmptyResult {
	if err := p.restart(); err != nil {
		logFail(p.log, "RestartToUpdate", err)
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
