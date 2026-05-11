package ops_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/runner"
)

func TestApplyIcon_CopiesAndTouches(t *testing.T) {
	r := runner.NewFake()
	svc, _ := newService(t, r)
	cfg := domain.DefaultConfig("/Users/t")
	inst, _ := cfg.Copy(2)

	if err := svc.ApplyIcon(context.Background(), inst, "/some/icon.icns"); err != nil {
		t.Fatalf("ApplyIcon() unexpected: %v", err)
	}
	got := r.CallStrings()
	if len(got) != 2 {
		t.Fatalf("got %d calls, want 2: %v", len(got), got)
	}
	if !strings.Contains(got[0], "cp /some/icon.icns /Applications/WeChat2.app/Contents/Resources/AppIcon.icns") {
		t.Errorf("call[0] = %q", got[0])
	}
	if !strings.Contains(got[1], "touch /Applications/WeChat2.app") {
		t.Errorf("call[1] = %q", got[1])
	}
}

func TestApplyIcon_RefusesOriginal(t *testing.T) {
	r := runner.NewFake()
	svc, _ := newService(t, r)
	cfg := domain.DefaultConfig("/Users/t")
	if err := svc.ApplyIcon(context.Background(), cfg.Original(), "/x.icns"); err == nil {
		t.Fatal("ApplyIcon(original) error = nil, want non-nil")
	}
}

func TestApplyIconMany_EmitsProgressAndFinalizes(t *testing.T) {
	r := runner.NewFake()
	svc, sink := newService(t, r)
	cfg := domain.DefaultConfig("/Users/t")
	a, _ := cfg.Copy(2)
	b, _ := cfg.Copy(3)

	if err := svc.ApplyIconMany(context.Background(), []domain.Instance{a, b}, "/x.icns"); err != nil {
		t.Fatalf("ApplyIconMany() unexpected: %v", err)
	}
	if len(sink.events) != 3 {
		t.Fatalf("got %d events, want 3 (%v)", len(sink.events), sink.events)
	}
	// Final cache-flush calls present
	got := r.CallStrings()
	foundRmCache := false
	foundKillDock := false
	for _, c := range got {
		if strings.Contains(c, "rm -rf /Library/Caches/com.apple.iconservices.store") {
			foundRmCache = true
		}
		if c == "killall Dock" {
			foundKillDock = true
		}
	}
	if !foundRmCache {
		t.Error("expected iconservices cache flush call")
	}
	if !foundKillDock {
		t.Error("expected killall Dock call")
	}
}
