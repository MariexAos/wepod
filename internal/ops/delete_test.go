package ops_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/ops"
	"github.com/mariexaos/wepod/internal/runner"
)

// ids is a tiny helper to keep test tables readable.
func ids(values ...int) []domain.InstanceID {
	out := make([]domain.InstanceID, len(values))
	for i, v := range values {
		out[i] = domain.InstanceID(v)
	}
	return out
}

func TestDelete_RefusesOriginal(t *testing.T) {
	r := runner.NewFake()
	svc, _ := newService(t, r)
	if err := svc.Delete(context.Background(), 0, false); err == nil {
		t.Fatal("Delete(0) error = nil, want non-nil")
	}
}

func TestDelete_MovesAppToTrashWithoutData(t *testing.T) {
	r := runner.NewFake()
	svc, _ := newService(t, r)

	if err := svc.Delete(context.Background(), 2, false); err != nil {
		t.Fatalf("Delete() unexpected: %v", err)
	}

	calls := r.CallStrings()
	// Expect: mkdir -p <trash>, sudo -n mv WeChat2.app <trash>/WeChat2.app
	if len(calls) != 2 {
		t.Fatalf("got %d calls, want 2: %v", len(calls), calls)
	}
	if !strings.HasPrefix(calls[0], "mkdir -p /Users/t/.Trash/wepod-undo") {
		t.Errorf("call[0] = %q, want mkdir prefix", calls[0])
	}
	if !strings.Contains(calls[1], "sudo -n mv /Applications/WeChat2.app") {
		t.Errorf("call[1] = %q, want sudo mv", calls[1])
	}
}

func TestDelete_WithDataAlsoMovesContainer(t *testing.T) {
	r := runner.NewFake()
	svc, _ := newService(t, r)

	if err := svc.Delete(context.Background(), 3, true); err != nil {
		t.Fatalf("Delete() unexpected: %v", err)
	}

	calls := r.CallStrings()
	if len(calls) != 3 {
		t.Fatalf("got %d calls, want 3: %v", len(calls), calls)
	}
	if !strings.Contains(calls[2], "mv /Users/t/Library/Containers/com.tencent.xinWeChat3") {
		t.Errorf("call[2] = %q, want data mv", calls[2])
	}
}

func TestDelete_MissingDataIsNotFatal(t *testing.T) {
	r := runner.NewFake()
	r.SetResponse(
		"mv /Users/t/Library/Containers/com.tencent.xinWeChat3 /Users/t/.Trash/wepod-undo/com.tencent.xinWeChat3",
		runner.FakeResponse{ExitCode: 1, Stderr: "mv: No such file or directory"},
	)
	svc, _ := newService(t, r)

	if err := svc.Delete(context.Background(), 3, true); err != nil {
		t.Fatalf("Delete() error = %v, want nil (missing data should be soft)", err)
	}
}

func TestDeleteMany_EmitsProgressForEachID(t *testing.T) {
	r := runner.NewFake()
	svc, sink := newService(t, r)

	if err := svc.DeleteMany(context.Background(), ids(2, 3, 5), false); err != nil {
		t.Fatalf("DeleteMany() unexpected: %v", err)
	}
	// 3 per-step + 1 final summary = 4 events
	if len(sink.events) != 4 {
		t.Fatalf("got %d events, want 4 (%v)", len(sink.events), sink.events)
	}
	last := sink.events[3].(ops.DeleteProgressEvent)
	if last.Done != 3 || last.Total != 3 {
		t.Errorf("final event = %+v, want Done=3 Total=3", last)
	}
}
