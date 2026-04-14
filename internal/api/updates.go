package api

import (
	"context"

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
	svc    UpdatesService
	opener URLOpener
	ctx    context.Context
}

func NewUpdatesProxy(svc UpdatesService, opener URLOpener) *UpdatesProxy {
	return &UpdatesProxy{svc: svc, opener: opener, ctx: context.Background()}
}

func (p *UpdatesProxy) Init(ctx context.Context) {
	p.ctx = ctx
}

func (p *UpdatesProxy) CheckNow() Result[updates.UpdateInfo] {
	info, err := p.svc.CheckNow(p.ctx)
	if err != nil {
		return FailResult[updates.UpdateInfo](err)
	}
	return SuccessResult(info)
}

func (p *UpdatesProxy) DismissUpdate(version string) EmptyResult {
	if err := p.svc.DismissVersion(version); err != nil {
		return Fail(err)
	}
	return Success()
}

func (p *UpdatesProxy) OpenReleasePage(url string) EmptyResult {
	p.opener.OpenURL(url)
	return Success()
}
