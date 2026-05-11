package ops_test

import (
	"context"
	"testing"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/runner"
)

func TestLaunch_OpensAppPath(t *testing.T) {
	r := runner.NewFake()
	svc, _ := newService(t, r)
	cfg := domain.DefaultConfig("/Users/t")
	if err := svc.Launch(context.Background(), cfg.Original()); err != nil {
		t.Fatalf("Launch() unexpected: %v", err)
	}
	got := r.CallStrings()
	if len(got) != 1 || got[0] != "open -a /Applications/WeChat.app" {
		t.Errorf("calls = %v, want [open -a /Applications/WeChat.app]", got)
	}
}
