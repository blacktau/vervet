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
