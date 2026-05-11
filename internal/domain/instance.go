// Package domain holds the core value types used across the application.
// It has no dependencies on other internal packages.
package domain

import (
	"fmt"
	"path/filepath"
	"strconv"
)

// InstanceID identifies a WeChat instance.
// ID 0 is the original /Applications/WeChat.app; positive IDs (2..) are copies.
type InstanceID int

// OriginalID is the well-known identifier for the unmodified WeChat install.
const OriginalID InstanceID = 0

// MinCopyID is the smallest valid copy identifier. WeChat1 is reserved.
const MinCopyID InstanceID = 2

// MaxCopyID is the largest copy identifier we allow.
const MaxCopyID InstanceID = 99

// IsOriginal reports whether the ID refers to the original install.
func (id InstanceID) IsOriginal() bool { return id == OriginalID }

// IsValidCopy reports whether id falls in the allowed copy range.
func (id InstanceID) IsValidCopy() bool { return id >= MinCopyID && id <= MaxCopyID }

// String returns "0" for the original, or the numeric form for copies.
func (id InstanceID) String() string { return strconv.Itoa(int(id)) }

// Instance is the immutable filesystem metadata for a WeChat install.
type Instance struct {
	ID       InstanceID
	Name     string // "WeChat" or "WeChat2"
	AppPath  string // "/Applications/WeChat2.app"
	BundleID string // "com.tencent.xinWeChat2"
	DataPath string // "$HOME/Library/Containers/com.tencent.xinWeChat2"
}

// IsOriginal reports whether this is the original install.
func (i Instance) IsOriginal() bool { return i.ID.IsOriginal() }

// Runtime is the volatile, process-level state of an instance.
// It is intentionally separate from Instance so it can be refreshed on its own cadence.
type Runtime struct {
	PID     int
	Running bool
}

// View is the composite shown to the UI layer.
// DataSize == -1 means "not computed yet".
type View struct {
	Instance Instance
	Runtime  Runtime
	DataSize int64
}

// DataSizeUnknown is the sentinel for an uncomputed data size.
const DataSizeUnknown int64 = -1

// Config describes external paths the application needs.
// It is supplied at startup; nothing else mutates it.
type Config struct {
	AppsDir       string // default "/Applications"
	HomeDir       string // user home
	OriginalApp   string // default "/Applications/WeChat.app"
	BaseBundleID  string // default "com.tencent.xinWeChat"
	ContainersDir string // default "$HOME/Library/Containers"
}

// DefaultConfig returns the production defaults given a home directory.
func DefaultConfig(homeDir string) Config {
	return Config{
		AppsDir:       "/Applications",
		HomeDir:       homeDir,
		OriginalApp:   "/Applications/WeChat.app",
		BaseBundleID:  "com.tencent.xinWeChat",
		ContainersDir: filepath.Join(homeDir, "Library", "Containers"),
	}
}

// Original returns the Instance for the unmodified install.
func (c Config) Original() Instance {
	return Instance{
		ID:       OriginalID,
		Name:     "WeChat",
		AppPath:  c.OriginalApp,
		BundleID: c.BaseBundleID,
		DataPath: filepath.Join(c.ContainersDir, c.BaseBundleID),
	}
}

// Copy returns the Instance descriptor for a given copy ID.
// It does not verify that the underlying files exist.
func (c Config) Copy(id InstanceID) (Instance, error) {
	if !id.IsValidCopy() {
		return Instance{}, fmt.Errorf("invalid copy id %d (allowed range %d..%d)", id, MinCopyID, MaxCopyID)
	}
	name := fmt.Sprintf("WeChat%d", id)
	bundle := fmt.Sprintf("%s%d", c.BaseBundleID, id)
	return Instance{
		ID:       id,
		Name:     name,
		AppPath:  filepath.Join(c.AppsDir, name+".app"),
		BundleID: bundle,
		DataPath: filepath.Join(c.ContainersDir, bundle),
	}, nil
}
