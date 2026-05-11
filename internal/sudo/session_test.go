package sudo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mariexaos/wepod/internal/runner"
	"github.com/mariexaos/wepod/internal/sudo"
)

func TestEnsure_FreshTimestampNoProbe(t *testing.T) {
	r := runner.NewFake()
	s := sudo.New(r)
	s.Refreshed()

	if err := s.Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure() = %v, want nil", err)
	}
	if r.CallCount() != 0 {
		t.Errorf("expected no sudo probe, got calls: %v", r.CallStrings())
	}
}

func TestEnsure_ExpiredProbesAndSucceeds(t *testing.T) {
	r := runner.NewFake()
	r.SetResponse("sudo -n -v", runner.FakeResponse{}) // success
	s := sudo.New(r)
	s.SetValidity(1 * time.Nanosecond)

	if err := s.Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure() = %v, want nil", err)
	}
	if got := r.CallStrings(); len(got) != 1 || got[0] != "sudo -n -v" {
		t.Errorf("calls = %v, want [\"sudo -n -v\"]", got)
	}
}

func TestEnsure_ExpiredAndProbeFailsReturnsNeedsPrompt(t *testing.T) {
	r := runner.NewFake()
	r.SetResponse("sudo -n -v", runner.FakeResponse{ExitCode: 1, Stderr: "sudo: a password is required"})
	s := sudo.New(r)

	err := s.Ensure(context.Background())
	if !errors.Is(err, sudo.ErrNeedsPrompt) {
		t.Fatalf("err = %v, want ErrNeedsPrompt", err)
	}
}

func TestRun_PrependsSudoFlags(t *testing.T) {
	r := runner.NewFake()
	r.SetResponse("sudo -n cp -R src dst", runner.FakeResponse{})
	s := sudo.New(r)
	s.Refreshed()

	_, err := s.Run(context.Background(), runner.Command{Name: "cp", Args: []string{"-R", "src", "dst"}})
	if err != nil {
		t.Fatalf("Run() unexpected: %v", err)
	}
	got := r.CallStrings()
	if len(got) != 1 || got[0] != "sudo -n cp -R src dst" {
		t.Errorf("calls = %v, want sudo-prefixed", got)
	}
}

func TestRun_AuthFailureTranslated(t *testing.T) {
	r := runner.NewFake()
	r.SetResponse("sudo -n cp x y", runner.FakeResponse{
		ExitCode: 1,
		Stderr:   "sudo: a password is required\n",
	})
	s := sudo.New(r)
	s.Refreshed()

	_, err := s.Run(context.Background(), runner.Command{Name: "cp", Args: []string{"x", "y"}})
	if !errors.Is(err, sudo.ErrNeedsPrompt) {
		t.Fatalf("err = %v, want ErrNeedsPrompt", err)
	}
}

func TestStream_PassesEventsThrough(t *testing.T) {
	r := runner.NewFake()
	r.SetResponse("sudo -n cp -R a b", runner.FakeResponse{
		StreamLines: []string{"line1", "line2"},
	})
	s := sudo.New(r)
	s.Refreshed()

	ch, err := s.Stream(context.Background(), runner.Command{Name: "cp", Args: []string{"-R", "a", "b"}})
	if err != nil {
		t.Fatalf("Stream() unexpected: %v", err)
	}
	var stdout int
	var exits int
	for ev := range ch {
		switch ev.Kind {
		case runner.EventStdout:
			stdout++
		case runner.EventExit:
			exits++
		}
	}
	if stdout != 2 {
		t.Errorf("stdout events = %d, want 2", stdout)
	}
	if exits != 1 {
		t.Errorf("exit events = %d, want 1", exits)
	}
}

func TestStream_AuthFailureTranslated(t *testing.T) {
	r := runner.NewFake()
	r.SetResponse("sudo -n cp x y", runner.FakeResponse{
		ExitCode:     1,
		StreamStderr: []string{"sudo: a password is required"},
	})
	s := sudo.New(r)
	s.Refreshed()

	ch, err := s.Stream(context.Background(), runner.Command{Name: "cp", Args: []string{"x", "y"}})
	if err != nil {
		t.Fatalf("Stream() unexpected: %v", err)
	}
	var exit runner.Event
	for ev := range ch {
		if ev.Kind == runner.EventExit {
			exit = ev
		}
	}
	if !errors.Is(exit.Err, sudo.ErrNeedsPrompt) {
		t.Fatalf("exit.Err = %v, want ErrNeedsPrompt", exit.Err)
	}
}
