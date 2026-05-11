package runner

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Exec is the real-world Runner backed by os/exec.
type Exec struct{}

// NewExec returns a Runner that shells out to real processes.
func NewExec() *Exec { return &Exec{} }

// Run executes cmd to completion and returns the captured output.
func (Exec) Run(ctx context.Context, cmd Command) (Result, error) {
	c := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	if cmd.Env != nil {
		c.Env = cmd.Env
	}
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	res := Result{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: c.ProcessState.ExitCode(),
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return res, fmt.Errorf("%s: exit %d: %s", cmd.String(), exitErr.ExitCode(), bytes.TrimSpace(stderr.Bytes()))
		}
		return res, fmt.Errorf("%s: %w", cmd.String(), err)
	}
	return res, nil
}

// Stream executes cmd, returning a channel of line events.
// The caller MUST drain the channel until closed to avoid goroutine leaks.
func (Exec) Stream(ctx context.Context, cmd Command) (<-chan Event, error) {
	c := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	if cmd.Env != nil {
		c.Env = cmd.Env
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := c.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}
	if err := c.Start(); err != nil {
		return nil, fmt.Errorf("%s: start: %w", cmd.String(), err)
	}

	ch := make(chan Event, 64)
	var wg sync.WaitGroup
	wg.Add(2)
	go pump(&wg, stdout, EventStdout, ch)
	go pump(&wg, stderr, EventStderr, ch)

	go func() {
		wg.Wait()
		err := c.Wait()
		ev := Event{Kind: EventExit, ExitCode: c.ProcessState.ExitCode()}
		if err != nil {
			ev.Err = fmt.Errorf("%s: %w", cmd.String(), err)
		}
		ch <- ev
		close(ch)
	}()

	return ch, nil
}

func pump(wg *sync.WaitGroup, r io.Reader, kind EventKind, ch chan<- Event) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		ch <- Event{Kind: kind, Line: scanner.Text()}
	}
}
