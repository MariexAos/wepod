package ops

import (
	"context"
	"fmt"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/runner"
)

// Stop terminates an instance's process by name.
//
// We avoid `killall -9` to allow the app's own shutdown path to run.
// A successful killall returns 0; an absent process returns 1 — both are fine here.
func (s *Service) Stop(ctx context.Context, inst domain.Instance) error {
	if s.dryRun {
		return nil
	}
	_, err := s.runner.Run(ctx, runner.Command{Name: "killall", Args: []string{inst.Name}})
	if err != nil && !isNoSuchProcess(err) {
		return fmt.Errorf("stop %s: %w", inst.Name, err)
	}
	return nil
}

// StopMany stops a batch sequentially, swallowing "no such process".
func (s *Service) StopMany(ctx context.Context, insts []domain.Instance) error {
	var firstErr error
	for _, inst := range insts {
		if err := s.Stop(ctx, inst); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// StopAll is a convenience for the "panic stop" UI action.
func (s *Service) StopAll(ctx context.Context, insts []domain.Instance) error {
	return s.StopMany(ctx, insts)
}

// dummy to silence unused-import linter if domain isn't referenced elsewhere here.
var _ = domain.OriginalID

func isNoSuchProcess(err error) bool {
	return contains(err.Error(), "No matching processes")
}
