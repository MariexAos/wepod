package runner_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mariexaos/wepod/internal/runner"
)

func TestExec_RunSuccess(t *testing.T) {
	r := runner.NewExec()
	res, err := r.Run(context.Background(), runner.Command{Name: "echo", Args: []string{"hello"}})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if got := strings.TrimSpace(string(res.Stdout)); got != "hello" {
		t.Errorf("Stdout = %q, want %q", got, "hello")
	}
}

func TestExec_RunFailure(t *testing.T) {
	r := runner.NewExec()
	_, err := r.Run(context.Background(), runner.Command{Name: "false"})
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
}

func TestExec_StreamCapturesLines(t *testing.T) {
	r := runner.NewExec()
	ch, err := r.Stream(context.Background(), runner.Command{
		Name: "sh",
		Args: []string{"-c", "echo a; echo b; echo c >&2"},
	})
	if err != nil {
		t.Fatalf("Stream() unexpected error: %v", err)
	}
	var stdout, stderr []string
	var exits int
	for ev := range ch {
		switch ev.Kind {
		case runner.EventStdout:
			stdout = append(stdout, ev.Line)
		case runner.EventStderr:
			stderr = append(stderr, ev.Line)
		case runner.EventExit:
			exits++
		}
	}
	if got, want := len(stdout), 2; got != want {
		t.Errorf("stdout lines = %d, want %d (%v)", got, want, stdout)
	}
	if got, want := len(stderr), 1; got != want {
		t.Errorf("stderr lines = %d, want %d (%v)", got, want, stderr)
	}
	if exits != 1 {
		t.Errorf("exit events = %d, want 1", exits)
	}
}

func TestExec_StreamReportsExitError(t *testing.T) {
	r := runner.NewExec()
	ch, err := r.Stream(context.Background(), runner.Command{Name: "false"})
	if err != nil {
		t.Fatalf("Stream() unexpected error: %v", err)
	}
	var sawErr bool
	for ev := range ch {
		if ev.Kind == runner.EventExit && ev.Err != nil {
			sawErr = true
		}
	}
	if !sawErr {
		t.Error("expected EventExit with Err for `false`")
	}
}
