# wepod

An elegant terminal UI for managing WeChat multi-instance setups on macOS.

[![CI](https://github.com/MariexAos/wepod/actions/workflows/ci.yml/badge.svg)](https://github.com/MariexAos/wepod/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/MariexAos/wepod?include_prereleases&sort=semver)](https://github.com/MariexAos/wepod/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/mariexaos/wepod.svg)](https://pkg.go.dev/github.com/mariexaos/wepod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Running multiple WeChat accounts on a single Mac means duplicating
`/Applications/WeChat.app` with different bundle IDs — tedious by hand and easy
to mess up (forgotten re-sign, wrong `chown`, leftover xattrs, …). `wepod`
collapses the whole workflow into a single keyboard-driven dashboard.

## Features

- **Single-screen dashboard** — every install (original + copies) on one view, with live process state, bundle ID, and selection markers
- **One-keystroke ops** — create / delete / launch / stop / update / icon-swap, all single keys
- **Manual update** — rebuild any copy from the current `/Applications/WeChat.app` after WeChat upgrades, keeping its bundle ID, name, and custom icon
- **Multi-select** — operate on any subset of copies, not just one at a time
- **Live state** — process detection refreshes every 1.5 s via `pgrep`
- **Soft delete** — copies move to `~/.Trash/wepod-undo/` before they're gone; restore with `mv` if you slip
- **Dry-run mode** — `--dry-run` reports what would happen without touching anything
- **Single static binary** — ~3 MB, no runtime dependencies

## Install

### One-liner (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/MariexAos/wepod/main/scripts/install.sh | bash
```

The script:

1. Detects your CPU (Apple Silicon vs Intel)
2. Downloads the matching release artifact and **verifies its sha256**
3. Strips the `com.apple.quarantine` xattr and ad-hoc re-signs the binary, so macOS Gatekeeper does **not** show the "developer cannot be verified" dialog
4. Installs to `/usr/local/bin/wepod` (sudo prompt if needed)

Install a specific version: append `bash -s -- v0.3.1`. Install without writing
to a system directory: use `PREFIX=$HOME/.local bash …`.

### Manual download

If you'd rather inspect each step yourself, grab the asset from [Releases](https://github.com/MariexAos/wepod/releases) and run:

```bash
tar xzf wepod-darwin-arm64.tar.gz   # or amd64
xattr -d com.apple.quarantine wepod   # bypass Gatekeeper warning
codesign --force --sign - wepod       # ad-hoc sign
sudo install -m 0755 wepod /usr/local/bin/wepod
```

### From source

```bash
go install github.com/mariexaos/wepod/cmd/wepod@latest
```

Or clone and build:

```bash
git clone https://github.com/MariexAos/wepod
cd wepod
make install
```

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/MariexAos/wepod/main/scripts/uninstall.sh | bash
```

## Usage

```bash
wepod
```

Browsing, launching, and stopping WeChat do not require sudo. You'll only be
prompted for your sudo password when an operation needs to modify an app bundle
under `/Applications` (create, update, delete, or icon swap). `--dry-run` never
prompts because it does not modify the filesystem.

### Key bindings

| Key | Action |
|---|---|
| `↑` / `↓` (`j` / `k`) | Move cursor |
| `Space` | Toggle selection on the current row |
| `a` | Select all / clear selection |
| `Enter` | Launch the current row or every selected row |
| `n` | New copy (form prefilled with the next free ID) |
| `d` | Delete selected (modal confirm; press `d` again to also wipe data dir) |
| `s` | Stop selected (with nothing selected: stop everything) |
| `u` | Update selected — rebuild from the current `WeChat.app`, preserving bundle ID / name / icon (quit the copy first) |
| `i` | Apply a `.icns` icon to selected copies |
| `r` | Rescan `/Applications` |
| `?` | Help overlay |
| `q` / `Ctrl+C` | Quit |

### Flags

| Flag | Description |
|---|---|
| `--dry-run` | Simulate destructive operations |
| `--debug` | Write logs to `$XDG_STATE_HOME/wepod/debug.log` |
| `--apps-dir DIR` | Override `/Applications` (handy for tests) |
| `--icon-dir DIR` | Override the icon directory (default: `./icon` next to the binary) |
| `--version` | Print version and exit |

## How it works

To materialize a copy, `wepod` runs:

1. `cp -R /Applications/WeChat.app /Applications/WeChat<N>.app`
2. `PlistBuddy` — set `CFBundleIdentifier` → `com.tencent.xinWeChat<N>`
3. `PlistBuddy` — set `CFBundleName` / `CFBundleDisplayName` → `WeChat<N>`
4. `xattr -cr` — strip quarantine and signing attributes
5. `codesign --force --deep --sign -` — ad-hoc re-sign
6. `chown -R` — fix ownership

Each copy lives in its own macOS sandbox container under
`~/Library/Containers/com.tencent.xinWeChat<N>/`, so the accounts are fully
isolated — login state, message history, attachments, everything.

Deletion moves the `.app` bundle to `~/.Trash/wepod-undo/<bundle>`. If you
hit `d` + `y` by mistake, just `mv` it back.

## Development

```bash
make test        # unit + integration tests
make test-race   # same with -race
make vet         # go vet
make lint        # golangci-lint (requires the binary)
make build       # bin/wepod
```

Layout:

```
cmd/wepod/        # entry point, dependency wiring
internal/
  domain/         # pure value types (InstanceID, Config, ...)
  runner/         # subprocess abstraction (Real + Fake)
  scanner/        # /Applications discovery
  sudo/           # sudo session management
  ops/            # business operations (create, delete, ...)
  tui/            # Bubble Tea front-end
```

All operations are unit-tested with a fake runner — no real `cp -R` runs in CI.

## License

[MIT](LICENSE)
