// Package sudo wraps a Runner with sudo timestamp management.
//
// Workflow:
//  1. The TUI calls Ensure() before a privileged operation. If the timestamp is fresh, it returns nil.
//  2. If expired, Ensure() returns ErrNeedsPrompt. The caller should suspend the TUI,
//     run an interactive `sudo -v` (e.g. via tea.ExecProcess), then call Refreshed().
//  3. Run / Stream prepend `sudo -n` (non-interactive) and surface ErrNeedsPrompt
//     when the timestamp has gone stale mid-session.
package sudo

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/mariexaos/wepod/internal/runner"
)

// ErrNeedsPrompt indicates that an interactive sudo authentication is required.
var ErrNeedsPrompt = errors.New("sudo: interactive authentication required")

// Session manages a cached sudo credential.
type Session interface {
	// Ensure verifies that the sudo timestamp is valid; returns ErrNeedsPrompt if not.
	Ensure(ctx context.Context) error

	// Refreshed is called after the host has run an interactive `sudo -v`.
	Refreshed()

	// Run invokes cmd under sudo -n.
	Run(ctx context.Context, cmd runner.Command) (runner.Result, error)

	// Stream invokes cmd under sudo -n with streaming output.
	Stream(ctx context.Context, cmd runner.Command) (<-chan runner.Event, error)
}

// Default is the production Session.
type Default struct {
	r runner.Runner

	mu          sync.Mutex
	lastRefresh time.Time
	// validity is the assumed lifetime of a sudo timestamp.
	// macOS defaults to 5 minutes; we conservatively assume 4.
	validity time.Duration
}

// New returns a Session backed by r.
func New(r runner.Runner) *Default {
	return &Default{r: r, validity: 4 * time.Minute}
}

// SetValidity overrides the assumed timestamp lifetime (useful in tests).
func (s *Default) SetValidity(d time.Duration) {
	s.mu.Lock()
	s.validity = d
	s.mu.Unlock()
}

// Ensure checks the cached timestamp; if expired, probes with `sudo -n -v` once.
// On failure it returns ErrNeedsPrompt without prompting.
func (s *Default) Ensure(ctx context.Context) error {
	s.mu.Lock()
	fresh := !s.lastRefresh.IsZero() && time.Since(s.lastRefresh) < s.validity
	s.mu.Unlock()
	if fresh {
		return nil
	}
	_, err := s.r.Run(ctx, runner.Command{Name: "sudo", Args: []string{"-n", "-v"}})
	if err != nil {
		return ErrNeedsPrompt
	}
	s.Refreshed()
	return nil
}

// Refreshed marks the timestamp as just-validated by an external interactive prompt.
func (s *Default) Refreshed() {
	s.mu.Lock()
	s.lastRefresh = time.Now()
	s.mu.Unlock()
}

// Run executes `sudo -n <cmd>` and translates auth failures into ErrNeedsPrompt.
func (s *Default) Run(ctx context.Context, cmd runner.Command) (runner.Result, error) {
	wrapped := wrap(cmd)
	res, err := s.r.Run(ctx, wrapped)
	if err != nil && looksLikeAuthFailure(res.Stderr, err) {
		return res, ErrNeedsPrompt
	}
	return res, err
}

// Stream executes `sudo -n <cmd>` and surfaces auth failures via ErrNeedsPrompt
// on the final EventExit event.
func (s *Default) Stream(ctx context.Context, cmd runner.Command) (<-chan runner.Event, error) {
	wrapped := wrap(cmd)
	src, err := s.r.Stream(ctx, wrapped)
	if err != nil {
		return nil, err
	}
	out := make(chan runner.Event, cap(src))
	go func() {
		defer close(out)
		var stderrBuf strings.Builder
		for ev := range src {
			if ev.Kind == runner.EventStderr {
				stderrBuf.WriteString(ev.Line)
				stderrBuf.WriteString("\n")
			}
			if ev.Kind == runner.EventExit && ev.Err != nil {
				if looksLikeAuthFailure([]byte(stderrBuf.String()), ev.Err) {
					ev.Err = ErrNeedsPrompt
				}
			}
			out <- ev
		}
	}()
	return out, nil
}

func wrap(cmd runner.Command) runner.Command {
	args := append([]string{"-n", cmd.Name}, cmd.Args...)
	return runner.Command{Name: "sudo", Args: args, Env: cmd.Env}
}

func looksLikeAuthFailure(stderr []byte, err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(string(stderr) + " " + err.Error())
	return strings.Contains(msg, "a password is required") ||
		strings.Contains(msg, "sudo: a terminal is required") ||
		strings.Contains(msg, "sudo: a password is required")
}
