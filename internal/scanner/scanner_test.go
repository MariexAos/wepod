package scanner_test

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/scanner"
)

// mapFS wraps fstest.MapFS so it satisfies scanner.FS using absolute paths.
// All keys in entries are absolute and rewritten to relative for fstest.
type mapFS struct {
	files map[string]bool // path -> isDir (we only care about presence here)
}

func newMapFS(paths map[string]bool) *mapFS {
	return &mapFS{files: paths}
}

func (m *mapFS) Stat(path string) (fs.FileInfo, error) {
	isDir, ok := m.files[path]
	if !ok {
		return nil, &fs.PathError{Op: "stat", Path: path, Err: os.ErrNotExist}
	}
	return fakeInfo{name: lastSegment(path), dir: isDir}, nil
}

func (m *mapFS) ReadDir(dir string) ([]fs.DirEntry, error) {
	if isDir, ok := m.files[dir]; !ok || !isDir {
		return nil, &fs.PathError{Op: "readdir", Path: dir, Err: os.ErrNotExist}
	}
	var out []fs.DirEntry
	prefix := dir
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for p, isDir := range m.files {
		if !strings.HasPrefix(p, prefix) {
			continue
		}
		rest := strings.TrimPrefix(p, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		out = append(out, fakeDirEntry{name: rest, dir: isDir})
	}
	return out, nil
}

func lastSegment(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

type fakeInfo struct {
	name string
	dir  bool
}

func (f fakeInfo) Name() string       { return f.name }
func (f fakeInfo) Size() int64        { return 0 }
func (f fakeInfo) Mode() fs.FileMode  { return 0 }
func (f fakeInfo) ModTime() time.Time { return time.Time{} }
func (f fakeInfo) IsDir() bool        { return f.dir }
func (f fakeInfo) Sys() interface{}   { return nil }

type fakeDirEntry struct {
	name string
	dir  bool
}

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return f.dir }
func (f fakeDirEntry) Type() fs.FileMode          { return 0 }
func (f fakeDirEntry) Info() (fs.FileInfo, error) { return fakeInfo(f), nil }

// compile-time check
var _ scanner.FS = (*mapFS)(nil)
var _ = fstest.MapFS{} // keep import used in future tests

func TestScan_OriginalMissing(t *testing.T) {
	fsys := newMapFS(map[string]bool{
		"/Applications": true,
	})
	s := scanner.New(fsys, domain.DefaultConfig("/Users/t"))
	_, err := s.Scan()
	if !errors.Is(err, scanner.ErrOriginalMissing) {
		t.Fatalf("err = %v, want ErrOriginalMissing", err)
	}
}

func TestScan_OriginalOnly(t *testing.T) {
	fsys := newMapFS(map[string]bool{
		"/Applications":            true,
		"/Applications/WeChat.app": true,
	})
	s := scanner.New(fsys, domain.DefaultConfig("/Users/t"))
	got, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() unexpected error: %v", err)
	}
	if len(got) != 1 || !got[0].IsOriginal() {
		t.Fatalf("want only original, got %+v", got)
	}
}

func TestScan_OriginalAndCopies(t *testing.T) {
	fsys := newMapFS(map[string]bool{
		"/Applications":             true,
		"/Applications/WeChat.app":  true,
		"/Applications/WeChat2.app": true,
		"/Applications/WeChat5.app": true,
		"/Applications/WeChat3.app": true,
		"/Applications/Other.app":   true, // ignored
		"/Applications/notes.txt":   false,
	})
	s := scanner.New(fsys, domain.DefaultConfig("/Users/t"))
	got, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() unexpected error: %v", err)
	}
	wantIDs := []domain.InstanceID{0, 2, 3, 5}
	if len(got) != len(wantIDs) {
		t.Fatalf("got %d instances, want %d (%+v)", len(got), len(wantIDs), got)
	}
	for i, id := range wantIDs {
		if got[i].ID != id {
			t.Errorf("instance[%d].ID = %d, want %d", i, got[i].ID, id)
		}
	}
}

func TestScan_RejectsOutOfRangeCopies(t *testing.T) {
	fsys := newMapFS(map[string]bool{
		"/Applications":               true,
		"/Applications/WeChat.app":    true,
		"/Applications/WeChat1.app":   true, // reserved, ignored
		"/Applications/WeChat100.app": true, // too large
		"/Applications/WeChat42.app":  true,
	})
	s := scanner.New(fsys, domain.DefaultConfig("/Users/t"))
	got, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan() unexpected error: %v", err)
	}
	if len(got) != 2 || got[1].ID != 42 {
		t.Fatalf("got %+v, want [original, 42]", got)
	}
}

func TestNextAvailableID(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	mk := func(ids ...domain.InstanceID) []domain.Instance {
		out := []domain.Instance{cfg.Original()}
		for _, id := range ids {
			inst, err := cfg.Copy(id)
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, inst)
		}
		return out
	}
	cases := []struct {
		name string
		in   []domain.Instance
		want domain.InstanceID
	}{
		{"empty", mk(), 2},
		{"dense_from_2", mk(2, 3, 4), 5},
		{"with_gap", mk(2, 4, 5), 3},
		{"5_gap_at_top", mk(2, 3, 4, 6), 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := scanner.NextAvailableID(tc.in)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}
