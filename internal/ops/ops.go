// Package ops implements the WeChat copy lifecycle operations.
//
// Each public method consumes a context and a domain.Instance (or ID), shells
// out via the injected Runner / Session, and reports progress via the ProgressSink.
//
// ops has no knowledge of bubbletea; the TUI layer adapts ProgressSink to tea.Program.Send.
package ops

import (
	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/runner"
	"github.com/mariexaos/wepod/internal/sudo"
)

// ProgressSink receives intermediate updates from long-running operations.
// Implementations must be safe to call from any goroutine.
type ProgressSink interface {
	Send(any)
}

// nopSink discards progress events. Useful in tests and dry-runs.
type nopSink struct{}

// Send is a no-op.
func (nopSink) Send(any) {}

// NopSink returns a ProgressSink that discards events.
func NopSink() ProgressSink { return nopSink{} }

// Service bundles the dependencies needed to perform operations.
//
// All methods are pure with respect to Service state: no mutexes, no caching.
// Concurrent invocations on the same instance are the caller's responsibility.
type Service struct {
	cfg    domain.Config
	runner runner.Runner
	sudo   sudo.Session
	sink   ProgressSink
	dryRun bool
}

// Option configures a Service.
type Option func(*Service)

// WithDryRun causes destructive shell-outs to be replaced with logs only.
func WithDryRun(b bool) Option {
	return func(s *Service) { s.dryRun = b }
}

// WithSink installs a custom progress sink.
func WithSink(sink ProgressSink) Option {
	return func(s *Service) {
		if sink != nil {
			s.sink = sink
		}
	}
}

// NewService constructs a Service.
func NewService(cfg domain.Config, r runner.Runner, sess sudo.Session, opts ...Option) *Service {
	s := &Service{
		cfg:    cfg,
		runner: r,
		sudo:   sess,
		sink:   NopSink(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Config exposes the bundled config (read-only).
func (s *Service) Config() domain.Config { return s.cfg }

// DryRun reports whether destructive operations are suppressed.
func (s *Service) DryRun() bool { return s.dryRun }

// SetSink installs a progress sink post-construction.
//
// This exists because the production sink references the *tea.Program, which
// cannot be created before the Model the Service is wired into. Pass nil to
// leave the current sink untouched.
func (s *Service) SetSink(sink ProgressSink) {
	if sink != nil {
		s.sink = sink
	}
}
