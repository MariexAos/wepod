package ops

import (
	"context"
	"fmt"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/runner"
)

// Launch opens the instance via macOS `open -a`. Does not require sudo.
func (s *Service) Launch(ctx context.Context, inst domain.Instance) error {
	if s.dryRun {
		return nil
	}
	if _, err := s.runner.Run(ctx, runner.Command{Name: "open", Args: []string{"-a", inst.AppPath}}); err != nil {
		return fmt.Errorf("launch %s: %w", inst.Name, err)
	}
	return nil
}
