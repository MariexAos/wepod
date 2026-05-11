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

// recordingSink captures progress events for assertions.
type recordingSink struct{ events []any }

func (r *recordingSink) Send(v any) { r.events = append(r.events, v) }

func newService(t *testing.T, r *runner.Fake) (*ops.Service, *recordingSink) {
	t.Helper()
	sess := sudo.New(r)
	sess.Refreshed()
	sink := &recordingSink{}
	svc := ops.NewService(domain.DefaultConfig("/Users/t"), r, sess, ops.WithSink(sink))
	return svc, sink
}

func TestCreate_RunsAllSixStepsInOrder(t *testing.T) {
	r := runner.NewFake() // default response = success
	svc, sink := newService(t, r)

	if err := svc.Create(context.Background(), 2); err != nil {
		t.Fatalf("Create() unexpected: %v", err)
	}

	got := r.CallStrings()
	wantPrefixes := []string{
		"sudo -n cp -R /Applications/WeChat.app /Applications/WeChat2.app",
		"sudo -n /usr/libexec/PlistBuddy -c Set :CFBundleIdentifier com.tencent.xinWeChat2",
		"sudo -n /usr/libexec/PlistBuddy -c Set :CFBundleName WeChat2",
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

	if len(sink.events) != 6 {
		t.Fatalf("got %d progress events, want 6", len(sink.events))
	}
	first := sink.events[0].(ops.CreateProgressEvent)
	want := ops.CreateProgressEvent{ID: 2, Step: 1, Total: 6, Label: "copy app bundle"}
	if diff := cmp.Diff(want, first); diff != "" {
		t.Errorf("first event mismatch (-want +got):\n%s", diff)
	}
}

func TestCreate_RejectsInvalidID(t *testing.T) {
	r := runner.NewFake()
	svc, _ := newService(t, r)
	if err := svc.Create(context.Background(), 1); err == nil {
		t.Fatal("Create(1) error = nil, want non-nil")
	}
	if r.CallCount() != 0 {
		t.Errorf("invalid id should not shell out, got: %v", r.CallStrings())
	}
}

func TestCreate_StopsAtFirstFailure(t *testing.T) {
	r := runner.NewFake()
	// Make the 4th step (xattr) fail.
	r.SetResponse("sudo -n xattr -cr /Applications/WeChat2.app", runner.FakeResponse{ExitCode: 1, Stderr: "boom"})
	svc, _ := newService(t, r)

	err := svc.Create(context.Background(), 2)
	if err == nil {
		t.Fatal("Create() error = nil, want non-nil")
	}
	// Should have called steps 1-4 only.
	if got, want := r.CallCount(), 4; got != want {
		t.Errorf("calls = %d, want %d (%v)", got, want, r.CallStrings())
	}
}

func TestCreate_DryRunDoesNothing(t *testing.T) {
	r := runner.NewFake()
	sess := sudo.New(r)
	sess.Refreshed()
	sink := &recordingSink{}
	svc := ops.NewService(domain.DefaultConfig("/Users/t"), r, sess, ops.WithSink(sink), ops.WithDryRun(true))

	if err := svc.Create(context.Background(), 3); err != nil {
		t.Fatalf("Create() unexpected: %v", err)
	}
	if r.CallCount() != 0 {
		t.Errorf("dry-run shelled out: %v", r.CallStrings())
	}
	if len(sink.events) != 1 {
		t.Errorf("got %d events, want 1 (dry-run marker)", len(sink.events))
	}
}
