package api

import (
	"context"
	"log/slog"

	"vervet/internal/updates"
)

type UpdatesService interface {
	CheckNow(ctx context.Context) (updates.UpdateInfo, error)
	DismissVersion(version string) error
}

type URLOpener interface {
	OpenURL(url string)
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

func (p *UpdatesProxy) Init(ctx context.Context) {
	p.ctx = ctx
}

func (p *UpdatesProxy) CheckNow() Result[updates.UpdateInfo] {
	info, err := p.svc.CheckNow(p.ctx)
	if err != nil {
		logFail(p.log, "CheckNow", err)
		return FailResult[updates.UpdateInfo](err)
	}
	return SuccessResult(info)
}

func (p *UpdatesProxy) DismissUpdate(version string) EmptyResult {
	if err := p.svc.DismissVersion(version); err != nil {
		logFail(p.log, "DismissUpdate", err)
		return Fail(err)
	}
	return Success()
}

func (p *UpdatesProxy) OpenReleasePage(url string) EmptyResult {
	p.opener.OpenURL(url)
	return Success()
}
