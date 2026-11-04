package api

import "log/slog"

// CloseGuardProvider tracks a request to close the application.
type CloseGuardProvider interface {
	SetUnsavedChanges(unsaved bool)
	AcknowledgeCloseRequest()
	CancelClose()
	ConfirmClose()
}

// CloseGuardProxy lets the frontend report unsaved queries and answer requests to
// close the application.
type CloseGuardProxy struct {
	log     *slog.Logger
	service CloseGuardProvider
}

func NewCloseGuardProxy(log *slog.Logger, service CloseGuardProvider) *CloseGuardProxy {
	return &CloseGuardProxy{log: log, service: service}
}

func (p *CloseGuardProxy) SetUnsavedChanges(unsaved bool) EmptyResult {
	p.service.SetUnsavedChanges(unsaved)
	return Success()
}

func (p *CloseGuardProxy) AcknowledgeCloseRequest() EmptyResult {
	p.service.AcknowledgeCloseRequest()
	return Success()
}

func (p *CloseGuardProxy) CancelClose() EmptyResult {
	p.service.CancelClose()
	return Success()
}

func (p *CloseGuardProxy) ConfirmClose() EmptyResult {
	p.service.ConfirmClose()
	return Success()
}
