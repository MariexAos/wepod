// Package runner abstracts subprocess execution so that ops can be unit-tested
// without spawning real commands.
package runner

import "context"

// Command describes a process invocation.
type Command struct {
	Name string
	Args []string
	// Env, when non-nil, replaces the inherited environment.
	Env []string
}

// String returns a shell-like representation for logs and fake lookups.
func (c Command) String() string {
	s := c.Name
	for _, a := range c.Args {
		s += " " + a
	}
	return s
}

// Result holds the outcome of a one-shot Run.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// EventKind tags a streaming event.
type EventKind int

const (
	// EventStdout carries one line from stdout.
	EventStdout EventKind = iota
	// EventStderr carries one line from stderr.
	EventStderr
	// EventExit is the final event; Err is non-nil on failure.
	EventExit
)

// Event is one item in a Stream channel.
type Event struct {
	Kind     EventKind
	Line     string
	ExitCode int
	Err      error
}

// Runner executes external commands.
//
// Implementations must:
//   - honor ctx cancellation (kill the process)
//   - never return a nil channel from Stream
//   - always emit exactly one EventExit on Stream and close the channel
type Runner interface {
	Run(ctx context.Context, cmd Command) (Result, error)
	Stream(ctx context.Context, cmd Command) (<-chan Event, error)
}
