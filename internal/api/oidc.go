package api

import (
	"log/slog"
)

type OIDCProxy struct {
	log      *slog.Logger
	provider OIDCProvider
}

type OIDCProvider interface {
	CancelLogin(serverID string)
	ResetSession(serverID string) error
}

func NewOIDCProxy(log *slog.Logger, provider OIDCProvider) *OIDCProxy {
	return &OIDCProxy{
		log:      log,
		provider: provider,
	}
}

func (p *OIDCProxy) CancelLogin(serverID string) EmptyResult {
	p.provider.CancelLogin(serverID)
	return Success()
}

func (p *OIDCProxy) ResetSession(serverID string) EmptyResult {
	if err := p.provider.ResetSession(serverID); err != nil {
		logFail(p.log, "ResetSession", err)
		return Fail(err)
	}
	return Success()
}
