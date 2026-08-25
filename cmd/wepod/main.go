// Command wepod is a TUI for managing WeChat multi-instance setups on macOS.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/ops"
	"github.com/mariexaos/wepod/internal/runner"
	"github.com/mariexaos/wepod/internal/scanner"
	"github.com/mariexaos/wepod/internal/sudo"
	"github.com/mariexaos/wepod/internal/tui"
)

// version is overridden at build time via -ldflags.
var version = "dev"

func main() {
	var (
		dryRun     bool
		debug      bool
		appsDir    string
		showVer    bool
		iconDirArg string
	)
	flag.BoolVar(&dryRun, "dry-run", false, "simulate destructive ops without executing them")
	flag.BoolVar(&debug, "debug", false, "write debug logs to $XDG_STATE_HOME/wepod/debug.log")
	flag.StringVar(&appsDir, "apps-dir", "/Applications", "override the applications directory")
	flag.BoolVar(&showVer, "version", false, "print version and exit")
	flag.StringVar(&iconDirArg, "icon-dir", "", "override the icon directory (default: ./icon next to binary)")
	flag.Parse()

	if showVer {
		fmt.Printf("wepod %s\n", version)
		return
	}

	if debug {
		closeFn, err := setupDebugLog()
		if err != nil {
			fmt.Fprintf(os.Stderr, "debug log: %v\n", err)
			os.Exit(2)
		}
		defer closeFn()
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve home: %v\n", err)
		os.Exit(2)
	}

	cfg := domain.DefaultConfig(home)
	cfg.AppsDir = appsDir
	cfg.OriginalApp = filepath.Join(appsDir, "WeChat.app")

	iconDir := iconDirArg
	if iconDir == "" {
		iconDir = defaultIconDir()
	}

	r := runner.NewExec()
	sess := sudo.New(r)
	svc := ops.NewService(cfg, r, sess, ops.WithDryRun(dryRun))

	model := tui.New(tui.Deps{
		Service:    svc,
		Sudo:       sess,
		Scanner:    scanner.New(scanner.OS{}, cfg),
		IconLister: tui.DefaultIconLister(iconDir),
	})

	// The progress sink references the tea.Program, which can only exist after
	// the model. Install the sink post-hoc.
	prog := tea.NewProgram(model, tea.WithAltScreen())
	svc.SetSink(tui.NewProgramSink(prog))

	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tea: %v\n", err)
		os.Exit(1)
	}
}

// defaultIconDir returns the icon directory next to the executable.
func defaultIconDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "icon"
	}
	return filepath.Join(filepath.Dir(exe), "icon")
}

func setupDebugLog() (func(), error) {
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		stateHome = filepath.Join(home, ".local", "state")
	}
	dir := filepath.Join(stateHome, "wepod")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "debug.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	log.SetOutput(f)
	return func() { _ = f.Close() }, nil
}
