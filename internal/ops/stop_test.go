package ops_test

import (
	"context"
	"testing"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/runner"
)

func TestStop_KillsByName(t *testing.T) {
	r := runner.NewFake()
	svc, _ := newService(t, r)
	cfg := domain.DefaultConfig("/Users/t")
	inst, err := cfg.Copy(2)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Stop(context.Background(), inst); err != nil {
		t.Fatalf("Stop() unexpected: %v", err)
	}
	got := r.CallStrings()
	if len(got) != 1 || got[0] != "killall WeChat2" {
		t.Errorf("calls = %v, want [killall WeChat2]", got)
	}
}

func TestStop_NoMatchingProcessIsSoft(t *testing.T) {
	r := runner.NewFake()
	r.SetResponse("killall WeChat2", runner.FakeResponse{
		ExitCode: 1,
		Stderr:   "No matching processes belonging to you were found",
	})
	svc, _ := newService(t, r)
	cfg := domain.DefaultConfig("/Users/t")
	inst, _ := cfg.Copy(2)
	if err := svc.Stop(context.Background(), inst); err != nil {
		t.Errorf("Stop() should swallow 'no matching processes', got %v", err)
	}
}
