package api

import (
	"context"
	"errors"
	"testing"
	"vervet/internal/updates"
)

type fakeUpdatesService struct {
	info      updates.UpdateInfo
	err       error
	dismissed string
	openURL   string
	ctx       context.Context
}

func (f *fakeUpdatesService) CheckNow(ctx context.Context) (updates.UpdateInfo, error) {
	f.ctx = ctx
	return f.info, f.err
}
func (f *fakeUpdatesService) CheckIfDue(ctx context.Context) error {
	f.ctx = ctx
	return f.err
}
func (f *fakeUpdatesService) DismissVersion(v string) error {
	f.dismissed = v
	return nil
}

type fakeURLOpener struct{ url string }

func (f *fakeURLOpener) OpenURL(url string) error { f.url = url; return nil }

func TestUpdatesProxy_CheckNow_Success(t *testing.T) {
	svc := &fakeUpdatesService{info: updates.UpdateInfo{Available: true, Version: "2026.05.1"}}
	p := NewUpdatesProxy(context.Background(), testLogger(), svc, &fakeURLOpener{})
	r := p.CheckNow()
	if !r.IsSuccess || r.Data.Version != "2026.05.1" {
		t.Fatalf("unexpected: %+v", r)
	}
}

func TestUpdatesProxy_CheckNow_Failure(t *testing.T) {
	svc := &fakeUpdatesService{err: errors.New("boom")}
	p := NewUpdatesProxy(context.Background(), testLogger(), svc, &fakeURLOpener{})
	r := p.CheckNow()
	if r.IsSuccess {
		t.Fatalf("expected failure")
	}
}

// Checks must run with the app's context, so they stop when the app shuts down.
func TestUpdatesProxy_ChecksUseAppContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := &fakeUpdatesService{}
	p := NewUpdatesProxy(ctx, testLogger(), svc, &fakeURLOpener{})

	p.CheckNow()
	if svc.ctx == nil || svc.ctx.Err() == nil {
		t.Fatalf("CheckNow did not get the app context")
	}

	svc.ctx = nil
	p.CheckIfDue()
	if svc.ctx == nil || svc.ctx.Err() == nil {
		t.Fatalf("CheckIfDue did not get the app context")
	}
}

func TestUpdatesProxy_Dismiss(t *testing.T) {
	svc := &fakeUpdatesService{}
	p := NewUpdatesProxy(context.Background(), testLogger(), svc, &fakeURLOpener{})
	r := p.DismissUpdate("2026.05.1")
	if !r.IsSuccess || svc.dismissed != "2026.05.1" {
		t.Fatalf("unexpected: dismissed=%q result=%+v", svc.dismissed, r)
	}
}

func TestUpdatesProxy_OpenReleasePage(t *testing.T) {
	opener := &fakeURLOpener{}
	p := NewUpdatesProxy(context.Background(), testLogger(), &fakeUpdatesService{}, opener)
	r := p.OpenReleasePage("https://example.com")
	if !r.IsSuccess || opener.url != "https://example.com" {
		t.Fatalf("url not forwarded: %q", opener.url)
	}
}
