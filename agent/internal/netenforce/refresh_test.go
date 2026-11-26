package netenforce

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeEnforcer struct {
	applied int
	fail    bool
}

func (f *fakeEnforcer) Apply(_ context.Context, _ Spec) error {
	if f.fail {
		return errors.New("apply failed")
	}
	f.applied++
	return nil
}

func (f *fakeEnforcer) Clear(context.Context) error { return nil }

func (f *fakeEnforcer) OnDrop(func(DropInfo)) {}

func TestRefreshOnceAppliesOnlyOnChange(t *testing.T) {
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	fakeHostResolver(t, &now)
	lookupIP = lookupReturns("203.0.113.1")

	spec := Spec{Rules: []Rule{{ID: "h", Action: "deny", DestType: "host", DestValue: "cdn.test"}}}
	enf := &fakeEnforcer{}
	ctx := context.Background()

	sig, err := refreshOnce(ctx, enf, spec, "", "")
	if err != nil || enf.applied != 1 {
		t.Fatalf("first call: err=%v applied=%d", err, enf.applied)
	}

	sig, err = refreshOnce(ctx, enf, spec, "", sig)
	if err != nil || enf.applied != 1 {
		t.Fatalf("unchanged call should not apply: err=%v applied=%d", err, enf.applied)
	}

	now = now.Add(time.Minute)
	lookupIP = lookupReturns("203.0.113.2")
	_, err = refreshOnce(ctx, enf, spec, "", sig)
	if err != nil || enf.applied != 2 {
		t.Fatalf("rotated IP should apply: err=%v applied=%d", err, enf.applied)
	}
}

func TestRefreshOnceKeepsOldSigOnApplyError(t *testing.T) {
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	fakeHostResolver(t, &now)
	lookupIP = lookupReturns("203.0.113.1")

	spec := Spec{Rules: []Rule{{ID: "h", Action: "deny", DestType: "host", DestValue: "cdn.test"}}}
	enf := &fakeEnforcer{fail: true}

	sig, err := refreshOnce(context.Background(), enf, spec, "", "old")
	if err == nil {
		t.Fatal("expected apply error")
	}
	if sig != "old" {
		t.Fatalf("on error signature must stay old, got %q", sig)
	}
}
