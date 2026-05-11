package runner_test

import (
	"context"
	"testing"

	"github.com/mariexaos/wepod/internal/runner"
)

func TestFake_RunRecordsCalls(t *testing.T) {
	f := runner.NewFake()
	f.SetResponse("echo hi", runner.FakeResponse{Stdout: "hi\n"})

	res, err := f.Run(context.Background(), runner.Command{Name: "echo", Args: []string{"hi"}})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if got, want := string(res.Stdout), "hi\n"; got != want {
		t.Errorf("Stdout = %q, want %q", got, want)
	}

	if got, want := f.CallCount(), 1; got != want {
		t.Errorf("CallCount = %d, want %d", got, want)
	}
	if got, want := f.CallStrings()[0], "echo hi"; got != want {
		t.Errorf("CallStrings()[0] = %q, want %q", got, want)
	}
}

func TestFake_RunReturnsErrorOnNonZeroExit(t *testing.T) {
	f := runner.NewFake()
	f.SetResponse("false", runner.FakeResponse{ExitCode: 1})

	_, err := f.Run(context.Background(), runner.Command{Name: "false"})
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
}

func TestFake_StreamEmitsLinesAndExit(t *testing.T) {
	f := runner.NewFake()
	f.SetResponse("cp -R a b", runner.FakeResponse{
		StreamLines: []string{"copying", "almost"},
	})

	ch, err := f.Stream(context.Background(), runner.Command{Name: "cp", Args: []string{"-R", "a", "b"}})
	if err != nil {
		t.Fatalf("Stream() unexpected error: %v", err)
	}

	var lines []string
	var exits int
	for ev := range ch {
		switch ev.Kind {
		case runner.EventStdout:
			lines = append(lines, ev.Line)
		case runner.EventExit:
			exits++
		}
	}
	if got, want := len(lines), 2; got != want {
		t.Errorf("got %d stdout lines, want %d", got, want)
	}
	if exits != 1 {
		t.Errorf("got %d exit events, want 1", exits)
	}
}

func TestFake_RespectsCanceledContext(t *testing.T) {
	f := runner.NewFake()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.Run(ctx, runner.Command{Name: "true"}); err == nil {
		t.Error("Run() with canceled ctx should error")
	}
	if _, err := f.Stream(ctx, runner.Command{Name: "true"}); err == nil {
		t.Error("Stream() with canceled ctx should error")
	}
}
