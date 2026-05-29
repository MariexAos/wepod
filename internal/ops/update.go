package ops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/runner"
)

// updateTotalSteps is the number of progress steps an Update emits per copy.
const updateTotalSteps = 9

// iconStash returns the temp path used to preserve a copy's custom icon across
// the bundle replacement. Overridable in tests for deterministic assertions.
var iconStash = func(inst domain.Instance) string {
	return filepath.Join(os.TempDir(), "wepod-update-"+inst.BundleID+".icns")
}

// Update refreshes a single copy from the current original install.
//
// The copy's bundle is replaced wholesale with the present
// /Applications/WeChat.app, then re-stamped with the copy's bundle ID and
// display name and re-signed — the same recipe Create uses, so a copy made
// against an older WeChat ends up byte-identical to one freshly created against
// the upgraded original. Any custom icon is preserved across the replacement on
// a best-effort basis.
//
// Update refuses to act on the original install. The copy should be quit before
// updating — replacing the bundle of a running app is unsupported.
func (s *Service) Update(ctx context.Context, id domain.InstanceID) error {
	if id.IsOriginal() {
		return fmt.Errorf("refusing to update the original install")
	}
	dst, err := s.cfg.Copy(id)
	if err != nil {
		return err
	}
	src := s.cfg.OriginalApp
	iconPath := filepath.Join(dst.AppPath, "Contents", "Resources", "AppIcon.icns")
	stash := iconStash(dst)

	if s.dryRun {
		s.sink.Send(UpdateProgressEvent{ID: id, Step: updateTotalSteps, Total: updateTotalSteps, Label: "(dry-run) skipped"})
		return nil
	}

	step := 0
	emit := func(label string) {
		step++
		s.sink.Send(UpdateProgressEvent{ID: id, Step: step, Total: updateTotalSteps, Label: label})
	}
	run := func(label string, cmd runner.Command) error {
		emit(label)
		if _, err := s.sudo.Run(ctx, cmd); err != nil {
			return fmt.Errorf("update %s (%s): %w", dst.Name, label, err)
		}
		return nil
	}

	// 1. Preserve the copy's current icon (best-effort): a missing or unreadable
	// icon just means we skip the later restore.
	emit("save current icon")
	_, stashErr := s.sudo.Run(ctx, runner.Command{Name: "cp", Args: []string{iconPath, stash}})
	iconSaved := stashErr == nil

	// 2-5. Replace the bundle and re-stamp identity.
	if err := run("remove old bundle", runner.Command{Name: "rm", Args: []string{"-rf", dst.AppPath}}); err != nil {
		return err
	}
	if err := run("copy app bundle", runner.Command{Name: "cp", Args: []string{"-R", src, dst.AppPath}}); err != nil {
		return err
	}
	if err := run("set bundle id", runner.Command{Name: "/usr/libexec/PlistBuddy", Args: []string{
		"-c", "Set :CFBundleIdentifier " + dst.BundleID,
		dst.AppPath + "/Contents/Info.plist",
	}}); err != nil {
		return err
	}
	if err := run("set display name", runner.Command{Name: "/usr/libexec/PlistBuddy", Args: []string{
		"-c", "Set :CFBundleName " + dst.Name,
		"-c", "Set :CFBundleDisplayName " + dst.Name,
		dst.AppPath + "/Contents/Info.plist",
	}}); err != nil {
		return err
	}

	// 6. Restore the preserved icon before re-signing so the signature covers it
	// (best-effort: a restore failure must not abort an otherwise-good update).
	emit("restore icon")
	if iconSaved {
		_, _ = s.sudo.Run(ctx, runner.Command{Name: "cp", Args: []string{stash, iconPath}})
		_, _ = s.runner.Run(ctx, runner.Command{Name: "rm", Args: []string{"-f", stash}})
	}

	// 7-9. Clear xattrs, re-sign, fix ownership — mirrors Create's tail.
	if err := run("clear extended attrs", runner.Command{Name: "xattr", Args: []string{"-cr", dst.AppPath}}); err != nil {
		return err
	}
	if err := run("re-sign bundle", runner.Command{Name: "codesign", Args: []string{"--force", "--deep", "--sign", "-", dst.AppPath}}); err != nil {
		return err
	}
	if err := run("fix permissions", runner.Command{Name: "chown", Args: []string{"-R", currentUser(), dst.AppPath}}); err != nil {
		return err
	}
	return nil
}

// UpdateMany refreshes each copy in turn from the current original, emitting
// per-step progress for each. It does not stop on the first error; the returned
// error wraps the first failure encountered.
func (s *Service) UpdateMany(ctx context.Context, ids []domain.InstanceID) error {
	var firstErr error
	for _, id := range ids {
		if err := s.Update(ctx, id); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
