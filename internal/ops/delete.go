package ops

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/runner"
)

// Delete removes a copy's .app bundle and (optionally) its data directory.
//
// The bundle is first moved to ~/.Trash/wepod-undo/<bundle-name>-<unix>,
// which gives the UI an undo window. withData==true additionally moves the
// container directory to the same trash location.
//
// Deleting the original install is rejected.
func (s *Service) Delete(ctx context.Context, id domain.InstanceID, withData bool) error {
	if id.IsOriginal() {
		return fmt.Errorf("refusing to delete the original install")
	}
	inst, err := s.cfg.Copy(id)
	if err != nil {
		return err
	}

	trashDir := s.trashDir()

	if s.dryRun {
		return nil
	}

	if _, err := s.runner.Run(ctx, runner.Command{Name: "mkdir", Args: []string{"-p", trashDir}}); err != nil {
		return fmt.Errorf("delete %s: prepare trash: %w", inst.Name, err)
	}

	if _, err := s.sudo.Run(ctx, runner.Command{
		Name: "mv",
		Args: []string{inst.AppPath, filepath.Join(trashDir, filepath.Base(inst.AppPath))},
	}); err != nil {
		return fmt.Errorf("delete %s: trash app: %w", inst.Name, err)
	}

	if withData {
		if _, err := s.runner.Run(ctx, runner.Command{
			Name: "mv",
			Args: []string{inst.DataPath, filepath.Join(trashDir, filepath.Base(inst.DataPath))},
		}); err != nil {
			// Data dir absence is not fatal; surface as a soft warning by ignoring.
			// Other errors should surface.
			if !isMissingPath(err) {
				return fmt.Errorf("delete %s: trash data: %w", inst.Name, err)
			}
		}
	}
	return nil
}

// DeleteMany applies Delete to each ID in order, emitting DeleteProgressEvent.
// It does not stop on the first error; the returned error wraps the first failure.
func (s *Service) DeleteMany(ctx context.Context, ids []domain.InstanceID, withData bool) error {
	var firstErr error
	for i, id := range ids {
		s.sink.Send(DeleteProgressEvent{ID: id, Done: i, Total: len(ids), WithData: withData})
		if err := s.Delete(ctx, id, withData); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	s.sink.Send(DeleteProgressEvent{Done: len(ids), Total: len(ids), WithData: withData})
	return firstErr
}

// trashDir returns the per-session undo directory under the user's Trash.
func (s *Service) trashDir() string {
	return filepath.Join(s.cfg.HomeDir, ".Trash", "wepod-undo")
}

func isMissingPath(err error) bool {
	// We deliberately match on string; the runner wraps the exec error so errors.Is
	// against fs.ErrNotExist would not work. macOS `mv` prints these consistently.
	return contains(err.Error(), "No such file or directory")
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
