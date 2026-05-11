package runner

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// FakeResponse describes what a Fake should return for a particular command.
type FakeResponse struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
	// StreamLines are emitted as EventStdout / EventStderr (alternating only if both filled).
	StreamLines  []string
	StreamStderr []string
}

// Fake is a deterministic in-memory Runner for tests.
//
// Responses are looked up by the command's String() form (e.g. "cp -R a b").
// If no response matches, Default is used (which by default succeeds silently).
//
// All recorded calls are kept in Calls, in invocation order.
type Fake struct {
	mu        sync.Mutex
	Responses map[string]FakeResponse
	Default   FakeResponse
	Calls     []Command
}

// NewFake returns a Fake with an empty response map and a successful default.
func NewFake() *Fake {
	return &Fake{Responses: map[string]FakeResponse{}}
}

// SetResponse registers a response for the exact command string.
func (f *Fake) SetResponse(key string, r FakeResponse) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Responses[key] = r
}

// CallCount returns the number of times Run or Stream was invoked.
func (f *Fake) CallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.Calls)
}

// CallStrings returns a snapshot of recorded commands as their string form.
func (f *Fake) CallStrings() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.Calls))
	for i, c := range f.Calls {
		out[i] = c.String()
	}
	return out
}

// Run records the call and returns the configured response.
func (f *Fake) Run(ctx context.Context, cmd Command) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	f.mu.Lock()
	f.Calls = append(f.Calls, cmd)
	resp, ok := f.Responses[cmd.String()]
	if !ok {
		resp = f.Default
	}
	f.mu.Unlock()
	res := Result{
		Stdout:   []byte(resp.Stdout),
		Stderr:   []byte(resp.Stderr),
		ExitCode: resp.ExitCode,
	}
	if resp.Err != nil {
		return res, fmt.Errorf("%s: %w", cmd.String(), resp.Err)
	}
	if resp.ExitCode != 0 {
		stderr := trimRight(resp.Stderr)
		if stderr != "" {
			return res, fmt.Errorf("%s: exit %d: %s", cmd.String(), resp.ExitCode, stderr)
		}
		return res, fmt.Errorf("%s: exit %d", cmd.String(), resp.ExitCode)
	}
	return res, nil
}

func trimRight(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// Stream records the call and emits the configured stream events.
func (f *Fake) Stream(ctx context.Context, cmd Command) (<-chan Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.Calls = append(f.Calls, cmd)
	resp, ok := f.Responses[cmd.String()]
	if !ok {
		resp = f.Default
	}
	f.mu.Unlock()

	ch := make(chan Event, len(resp.StreamLines)+len(resp.StreamStderr)+1)
	go func() {
		defer close(ch)
		for _, line := range resp.StreamLines {
			select {
			case <-ctx.Done():
				ch <- Event{Kind: EventExit, Err: ctx.Err()}
				return
			case ch <- Event{Kind: EventStdout, Line: line}:
			}
		}
		for _, line := range resp.StreamStderr {
			select {
			case <-ctx.Done():
				ch <- Event{Kind: EventExit, Err: ctx.Err()}
				return
			case ch <- Event{Kind: EventStderr, Line: line}:
			}
		}
		ev := Event{Kind: EventExit, ExitCode: resp.ExitCode}
		if resp.Err != nil {
			ev.Err = resp.Err
		} else if resp.ExitCode != 0 {
			ev.Err = errors.New(cmd.String() + " failed")
		}
		ch <- ev
	}()
	return ch, nil
}
