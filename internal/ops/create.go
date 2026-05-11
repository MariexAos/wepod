package ops

import (
	"context"
	"fmt"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/runner"
)

// createSteps lists the six sequenced commands used to materialize a copy.
// Splitting them by name keeps testing tractable and lets the UI label steps.
const createTotalSteps = 6

// Create materializes a single WeChat copy under the given ID.
// It is idempotent at the start: if the destination already exists, it returns an error.
func (s *Service) Create(ctx context.Context, id domain.InstanceID) error {
	dst, err := s.cfg.Copy(id)
	if err != nil {
		return err
	}
	src := s.cfg.OriginalApp

	if s.dryRun {
		s.sink.Send(CreateProgressEvent{ID: id, Step: createTotalSteps, Total: createTotalSteps, Label: "(dry-run) skipped"})
		return nil
	}

	steps := []struct {
		label string
		cmd   runner.Command
	}{
		{"copy app bundle", runner.Command{Name: "cp", Args: []string{"-R", src, dst.AppPath}}},
		{"set bundle id", runner.Command{Name: "/usr/libexec/PlistBuddy", Args: []string{
			"-c", "Set :CFBundleIdentifier " + dst.BundleID,
			dst.AppPath + "/Contents/Info.plist",
		}}},
		{"set display name", runner.Command{Name: "/usr/libexec/PlistBuddy", Args: []string{
			"-c", "Set :CFBundleName " + dst.Name,
			"-c", "Set :CFBundleDisplayName " + dst.Name,
			dst.AppPath + "/Contents/Info.plist",
		}}},
		{"clear extended attrs", runner.Command{Name: "xattr", Args: []string{"-cr", dst.AppPath}}},
		{"re-sign bundle", runner.Command{Name: "codesign", Args: []string{"--force", "--deep", "--sign", "-", dst.AppPath}}},
		{"fix permissions", runner.Command{Name: "chown", Args: []string{"-R", currentUser(), dst.AppPath}}},
	}

	for i, st := range steps {
		s.sink.Send(CreateProgressEvent{ID: id, Step: i + 1, Total: createTotalSteps, Label: st.label})
		if _, err := s.sudo.Run(ctx, st.cmd); err != nil {
			return fmt.Errorf("create %s (%s): %w", dst.Name, st.label, err)
		}
	}
	return nil
}

// currentUser returns $USER as a fallback for chown; it is overridden in tests.
var currentUser = func() string {
	// Resolved via env, not user.Current(): cheaper and avoids cgo on macOS.
	return getEnv("USER", "root")
}
