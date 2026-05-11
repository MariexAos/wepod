package ops

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/runner"
)

// ApplyIcon copies iconPath to the app's Contents/Resources/AppIcon.icns.
// It then bumps the bundle's mtime and clears caches so Dock picks up the change.
func (s *Service) ApplyIcon(ctx context.Context, inst domain.Instance, iconPath string) error {
	if inst.IsOriginal() {
		return fmt.Errorf("refusing to overwrite the original icon")
	}
	if s.dryRun {
		return nil
	}
	target := filepath.Join(inst.AppPath, "Contents", "Resources", "AppIcon.icns")
	if _, err := s.sudo.Run(ctx, runner.Command{Name: "cp", Args: []string{iconPath, target}}); err != nil {
		return fmt.Errorf("apply icon to %s: %w", inst.Name, err)
	}
	if _, err := s.sudo.Run(ctx, runner.Command{Name: "touch", Args: []string{inst.AppPath}}); err != nil {
		return fmt.Errorf("touch %s: %w", inst.Name, err)
	}
	return nil
}

// ApplyIconMany applies one icon to multiple copies, emitting progress.
// After the batch it refreshes the icon caches and Dock.
func (s *Service) ApplyIconMany(ctx context.Context, insts []domain.Instance, iconPath string) error {
	iconName := filepath.Base(iconPath)
	var firstErr error
	for i, inst := range insts {
		s.sink.Send(IconApplyEvent{ID: inst.ID, Done: i, Total: len(insts), IconName: iconName})
		if err := s.ApplyIcon(ctx, inst, iconPath); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	s.sink.Send(IconApplyEvent{Done: len(insts), Total: len(insts), IconName: iconName})

	if s.dryRun {
		return firstErr
	}
	// Best-effort cache flush; we don't fail the whole op on these.
	_, _ = s.sudo.Run(ctx, runner.Command{Name: "rm", Args: []string{"-rf", "/Library/Caches/com.apple.iconservices.store"}})
	_, _ = s.runner.Run(ctx, runner.Command{Name: "killall", Args: []string{"Dock"}})
	return firstErr
}
