package ops

import (
	"context"
	"path"
	"strconv"
	"strings"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/runner"
)

// RefreshRuntime returns a map from InstanceID to its current Runtime state.
//
// We run a single `pgrep -lf WeChat` and identify each process by the .app
// bundle directly above /Contents/MacOS/. The executable basename is NOT
// reliable: `cp -R WeChat.app WeChat2.app` leaves the binary named "WeChat"
// inside every copy, so the bundle directory is the only thing that varies.
func (s *Service) RefreshRuntime(ctx context.Context, insts []domain.Instance) (map[domain.InstanceID]domain.Runtime, error) {
	out := make(map[domain.InstanceID]domain.Runtime, len(insts))
	for _, inst := range insts {
		out[inst.ID] = domain.Runtime{}
	}

	res, err := s.runner.Run(ctx, runner.Command{Name: "pgrep", Args: []string{"-lf", "WeChat"}})
	if err != nil && len(res.Stdout) == 0 {
		// pgrep returns exit 1 when no matches; treat as "nothing running".
		return out, nil
	}

	byName := map[string]domain.InstanceID{}
	for _, inst := range insts {
		byName[inst.Name] = inst.ID
	}

	for _, line := range strings.Split(string(res.Stdout), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		id, ok := bundleIDFromExe(fields[1], byName)
		if !ok {
			continue
		}
		out[id] = domain.Runtime{PID: pid, Running: true}
	}
	return out, nil
}

// bundleIDFromExe finds the InstanceID whose .app bundle directly owns the
// executable path. Helpers nest as <Parent>.app/Contents/Frameworks/Helper.app/
// Contents/MacOS/Helper — the .app immediately above Contents/MacOS in that
// case is Helper.app, which won't match any of our names and is filtered out.
func bundleIDFromExe(exe string, byName map[string]domain.InstanceID) (domain.InstanceID, bool) {
	idx := strings.Index(exe, "/Contents/MacOS/")
	if idx <= 0 {
		return 0, false
	}
	bundlePath := exe[:idx]
	bundleName := path.Base(bundlePath)
	appBase, ok := strings.CutSuffix(bundleName, ".app")
	if !ok {
		return 0, false
	}
	id, ok := byName[appBase]
	return id, ok
}
