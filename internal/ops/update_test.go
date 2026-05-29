package ops_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/ops"
	"github.com/mariexaos/wepod/internal/runner"
	"github.com/mariexaos/wepod/internal/sudo"
)

func TestUpdate_ReplacesBundleAndPreservesIcon(t *testing.T) {
	r := runner.NewFake() // default response = success → icon stash succeeds
	svc, sink := newService(t, r)

	if err := svc.Update(context.Background(), 2); err != nil {
		t.Fatalf("Update() unexpected: %v", err)
	}

	got := r.CallStrings()
	wantPrefixes := []string{
		"sudo -n cp /Applications/WeChat2.app/Contents/Resources/AppIcon.icns ",
		"sudo -n rm -rf /Applications/WeChat2.app",
		"sudo -n cp -R /Applications/WeChat.app /Applications/WeChat2.app",
		"sudo -n /usr/libexec/PlistBuddy -c Set :CFBundleIdentifier com.tencent.xinWeChat2",
		"sudo -n /usr/libexec/PlistBuddy -c Set :CFBundleName WeChat2",
		"sudo -n cp ", // restore icon into the fresh bundle
		"rm -f ",      // remove the stash (non-sudo)
		"sudo -n xattr -cr /Applications/WeChat2.app",
		"sudo -n codesign --force --deep --sign - /Applications/WeChat2.app",
		"sudo -n chown -R",
	}
	if len(got) != len(wantPrefixes) {
		t.Fatalf("got %d calls, want %d: %v", len(got), len(wantPrefixes), got)
	}
	for i, want := range wantPrefixes {
		if !strings.HasPrefix(got[i], want) {
			t.Errorf("call[%d] = %q, want prefix %q", i, got[i], want)
		}
	}

	// Nine progress steps, the first labelled "save current icon".
	if len(sink.events) != 9 {
		t.Fatalf("got %d progress events, want 9", len(sink.events))
	}
	first := sink.events[0].(ops.UpdateProgressEvent)
	want := ops.UpdateProgressEvent{ID: 2, Step: 1, Total: 9, Label: "save current icon"}
	if diff := cmp.Diff(want, first); diff != "" {
		t.Errorf("first event mismatch (-want +got):\n%s", diff)
	}
}

func TestUpdate_SkipsIconRestoreWhenNoneSaved(t *testing.T) {
	r := runner.NewFake()
	const stash = "/tmp/wepod-test-stash.icns"
	restore := ops.SetIconStash(func(domain.Instance) string { return stash })
	defer restore()
	// No existing custom icon: the stash copy fails, so restore is skipped.
	r.SetResponse(
		"sudo -n cp /Applications/WeChat2.app/Contents/Resources/AppIcon.icns "+stash,
		runner.FakeResponse{ExitCode: 1, Stderr: "No such file or directory"},
	)
	svc, _ := newService(t, r)

	if err := svc.Update(context.Background(), 2); err != nil {
		t.Fatalf("Update() unexpected: %v", err)
	}

	for _, c := range r.CallStrings() {
		if strings.HasPrefix(c, "rm -f ") {
			t.Errorf("stash cleanup should not run when no icon was saved: %q", c)
		}
	}
}

func TestUpdate_RejectsOriginal(t *testing.T) {
	r := runner.NewFake()
	svc, _ := newService(t, r)
	if err := svc.Update(context.Background(), domain.OriginalID); err == nil {
		t.Fatal("Update(original) error = nil, want non-nil")
	}
	if r.CallCount() != 0 {
		t.Errorf("rejecting original should not shell out, got: %v", r.CallStrings())
	}
}

func TestUpdate_StopsAtFirstFailure(t *testing.T) {
	r := runner.NewFake()
	// Fail the bundle copy (step 3 overall, the 2nd hard step).
	r.SetResponse("sudo -n cp -R /Applications/WeChat.app /Applications/WeChat2.app", runner.FakeResponse{ExitCode: 1, Stderr: "boom"})
	svc, _ := newService(t, r)

	if err := svc.Update(context.Background(), 2); err == nil {
		t.Fatal("Update() error = nil, want non-nil")
	}
	// stash icon, rm -rf, cp -R(fail) → 3 calls, then abort.
	if got, want := r.CallCount(), 3; got != want {
		t.Errorf("calls = %d, want %d (%v)", got, want, r.CallStrings())
	}
}

func TestUpdate_DryRunDoesNothing(t *testing.T) {
	r := runner.NewFake()
	sess := sudo.New(r)
	sess.Refreshed()
	sink := &recordingSink{}
	svc := ops.NewService(domain.DefaultConfig("/Users/t"), r, sess, ops.WithSink(sink), ops.WithDryRun(true))

	if err := svc.Update(context.Background(), 3); err != nil {
		t.Fatalf("Update() unexpected: %v", err)
	}
	if r.CallCount() != 0 {
		t.Errorf("dry-run shelled out: %v", r.CallStrings())
	}
	if len(sink.events) != 1 {
		t.Errorf("got %d events, want 1 (dry-run marker)", len(sink.events))
	}
}

func TestUpdateMany_ProcessesEachID(t *testing.T) {
	r := runner.NewFake()
	svc, _ := newService(t, r)

	if err := svc.UpdateMany(context.Background(), []domain.InstanceID{2, 3}); err != nil {
		t.Fatalf("UpdateMany() unexpected: %v", err)
	}
	calls := r.CallStrings()
	if !containsCall(calls, "sudo -n cp -R /Applications/WeChat.app /Applications/WeChat2.app") ||
		!containsCall(calls, "sudo -n cp -R /Applications/WeChat.app /Applications/WeChat3.app") {
		t.Errorf("expected both copies updated, got: %v", calls)
	}
}

func containsCall(calls []string, want string) bool {
	for _, c := range calls {
		if c == want {
			return true
		}
	}
	return false
}
