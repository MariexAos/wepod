package ops_test

import (
	"context"
	"testing"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/runner"
)

// Realistic pgrep -lf output: the executable file under Contents/MacOS/ keeps
// the original name "WeChat" inside every copy bundle (cp -R doesn't rename
// the binary). The .app directory is what differs.
const pgrepOutput = `1234 /Applications/WeChat.app/Contents/MacOS/WeChat
5678 /Applications/WeChat2.app/Contents/MacOS/WeChat
9001 /Applications/WeChat2.app/Contents/Frameworks/WeChatAppEx.app/Contents/MacOS/WeChatAppEx
9002 /Applications/WeChat.app/Contents/Frameworks/WeChatAppEx.app/Contents/MacOS/WeChatAppEx
`

func TestRefreshRuntime_MapsPidsByBundleDir(t *testing.T) {
	r := runner.NewFake()
	r.SetResponse("pgrep -lf WeChat", runner.FakeResponse{Stdout: pgrepOutput})
	svc, _ := newService(t, r)
	cfg := domain.DefaultConfig("/Users/t")
	original := cfg.Original()
	two, _ := cfg.Copy(2)
	three, _ := cfg.Copy(3)

	got, err := svc.RefreshRuntime(context.Background(), []domain.Instance{original, two, three})
	if err != nil {
		t.Fatalf("RefreshRuntime() unexpected: %v", err)
	}
	if !got[0].Running || got[0].PID != 1234 {
		t.Errorf("original = %+v, want PID=1234 Running=true", got[0])
	}
	if !got[2].Running || got[2].PID != 5678 {
		t.Errorf("instance 2 = %+v, want PID=5678 Running=true", got[2])
	}
	if got[3].Running {
		t.Errorf("instance 3 = %+v, want Running=false", got[3])
	}
}

func TestRefreshRuntime_IgnoresHelperBundles(t *testing.T) {
	r := runner.NewFake()
	// Only helper processes present — none of the instances should be flagged.
	r.SetResponse("pgrep -lf WeChat", runner.FakeResponse{Stdout: `9001 /Applications/WeChat2.app/Contents/Frameworks/WeChatAppEx.app/Contents/MacOS/WeChatAppEx
`})
	svc, _ := newService(t, r)
	cfg := domain.DefaultConfig("/Users/t")
	two, _ := cfg.Copy(2)

	got, _ := svc.RefreshRuntime(context.Background(), []domain.Instance{cfg.Original(), two})
	if got[2].Running {
		t.Errorf("instance 2 = %+v, want Running=false (only helper alive)", got[2])
	}
}

func TestRefreshRuntime_NoProcessesYieldsAllFalse(t *testing.T) {
	r := runner.NewFake()
	r.SetResponse("pgrep -lf WeChat", runner.FakeResponse{ExitCode: 1})
	svc, _ := newService(t, r)
	cfg := domain.DefaultConfig("/Users/t")
	inst, _ := cfg.Copy(2)

	got, err := svc.RefreshRuntime(context.Background(), []domain.Instance{cfg.Original(), inst})
	if err != nil {
		t.Fatalf("RefreshRuntime() unexpected: %v", err)
	}
	for id, rt := range got {
		if rt.Running {
			t.Errorf("instance %d should be stopped, got %+v", id, rt)
		}
	}
}
