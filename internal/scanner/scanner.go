// Package scanner discovers WeChat installations under a given /Applications-like directory.
package scanner

import (
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strconv"

	"github.com/mariexaos/wepod/internal/domain"
)

// FS is the minimal filesystem surface the Scanner depends on.
type FS interface {
	Stat(path string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
}

// OS adapts the standard library's os package to FS.
type OS struct{}

// Stat calls os.Stat.
func (OS) Stat(p string) (fs.FileInfo, error) { return os.Stat(p) }

// ReadDir calls os.ReadDir.
func (OS) ReadDir(p string) ([]fs.DirEntry, error) { return os.ReadDir(p) }

// Scanner finds WeChat instances on disk.
type Scanner struct {
	fs  FS
	cfg domain.Config
}

// New returns a Scanner backed by the given filesystem and config.
func New(filesys FS, cfg domain.Config) *Scanner {
	return &Scanner{fs: filesys, cfg: cfg}
}

// ErrOriginalMissing is returned by Scan when the original WeChat install is absent.
var ErrOriginalMissing = fmt.Errorf("original WeChat install not found")

var copyNamePattern = regexp.MustCompile(`^WeChat(\d+)\.app$`)

// Scan lists the original install (if present) followed by all copies in ID order.
// Returns ErrOriginalMissing if /Applications/WeChat.app does not exist.
func (s *Scanner) Scan() ([]domain.Instance, error) {
	if _, err := s.fs.Stat(s.cfg.OriginalApp); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrOriginalMissing
		}
		return nil, fmt.Errorf("stat original app: %w", err)
	}

	entries, err := s.fs.ReadDir(s.cfg.AppsDir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", s.cfg.AppsDir, err)
	}

	var ids []domain.InstanceID
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m := copyNamePattern.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		id := domain.InstanceID(n)
		if !id.IsValidCopy() {
			continue
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	out := make([]domain.Instance, 0, len(ids)+1)
	out = append(out, s.cfg.Original())
	for _, id := range ids {
		inst, err := s.cfg.Copy(id)
		if err != nil {
			return nil, fmt.Errorf("build copy %d: %w", id, err)
		}
		out = append(out, inst)
	}
	return out, nil
}

// NextAvailableID returns the smallest copy ID not present in instances.
// It returns 0 with an error if MaxCopyID has been reached.
func NextAvailableID(instances []domain.Instance) (domain.InstanceID, error) {
	occupied := map[domain.InstanceID]bool{}
	for _, inst := range instances {
		occupied[inst.ID] = true
	}
	for id := domain.MinCopyID; id <= domain.MaxCopyID; id++ {
		if !occupied[id] {
			return id, nil
		}
	}
	return 0, fmt.Errorf("no copy ids available (range %d..%d exhausted)", domain.MinCopyID, domain.MaxCopyID)
}
