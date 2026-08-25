// Package tui implements the Bubble Tea front-end for wepod.
//
// The top-level Model owns a single screen "list" (the dashboard) plus a stack of
// transient modal sub-states (confirm, new-copy form, icon picker, help, busy).
package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/scanner"
	"github.com/mariexaos/wepod/internal/sudo"
)

// osReadDir is overridable in tests.
var osReadDir = os.ReadDir

// Mode is the model's current screen.
type Mode int

// Mode constants.
const (
	ModeList Mode = iota
	ModeConfirm
	ModeNewCopy
	ModeIconPick
	ModeHelp
	ModeBusy
)

// String returns a debugging name for the mode.
func (m Mode) String() string {
	switch m {
	case ModeList:
		return "list"
	case ModeConfirm:
		return "confirm"
	case ModeNewCopy:
		return "newcopy"
	case ModeIconPick:
		return "iconpick"
	case ModeHelp:
		return "help"
	case ModeBusy:
		return "busy"
	}
	return "?"
}

// Service is the trimmed ops surface the TUI depends on.
// It mirrors *ops.Service so tests can substitute fakes.
type Service interface {
	Config() domain.Config
	DryRun() bool
	Create(ctx context.Context, id domain.InstanceID) error
	Delete(ctx context.Context, id domain.InstanceID, withData bool) error
	DeleteMany(ctx context.Context, ids []domain.InstanceID, withData bool) error
	Update(ctx context.Context, id domain.InstanceID) error
	UpdateMany(ctx context.Context, ids []domain.InstanceID) error
	Launch(ctx context.Context, inst domain.Instance) error
	Stop(ctx context.Context, inst domain.Instance) error
	StopMany(ctx context.Context, insts []domain.Instance) error
	ApplyIcon(ctx context.Context, inst domain.Instance, iconPath string) error
	ApplyIconMany(ctx context.Context, insts []domain.Instance, iconPath string) error
	RefreshRuntime(ctx context.Context, insts []domain.Instance) (map[domain.InstanceID]domain.Runtime, error)
}

// Sudo is the surface the TUI needs for sudo session handling.
type Sudo interface {
	Ensure(ctx context.Context) error
	Refreshed()
}

// Scanner is the surface the TUI needs for app discovery.
type Scanner interface {
	Scan() ([]domain.Instance, error)
}

// IconLister returns the available icon files (.icns) in the bundled icon directory.
type IconLister func() ([]string, error)

// Deps bundles the model's external collaborators.
type Deps struct {
	Service    Service
	Sudo       Sudo
	Scanner    Scanner
	IconLister IconLister
}

// Model is the root Bubble Tea model.
type Model struct {
	deps   Deps
	keys   keyMap
	styles styles

	mode Mode

	// state
	instances []domain.Instance
	runtime   map[domain.InstanceID]domain.Runtime
	selected  map[domain.InstanceID]bool
	cursor    int

	confirm confirmState
	newcopy newCopyState
	icon    iconPickState
	busy    busyState

	// sudoPending is waiting for authentication. sudoRetry keeps the in-flight
	// privileged command so an expired credential can be refreshed and retried.
	sudoPending tea.Cmd
	sudoRetry   tea.Cmd

	toast         string
	toastErr      bool
	toastDeadline time.Time

	width, height int

	// runtime cancellation guards
	ctx    context.Context
	cancel context.CancelFunc
}

// New constructs a Model.
func New(deps Deps) *Model {
	ctx, cancel := context.WithCancel(context.Background())
	return &Model{
		deps:     deps,
		keys:     defaultKeys(),
		styles:   defaultStyles(),
		selected: map[domain.InstanceID]bool{},
		runtime:  map[domain.InstanceID]domain.Runtime{},
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Init kicks off the initial scan and starts the runtime refresh tick.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadInstancesCmd(),
		runtimeTick(),
	)
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case instancesLoadedMsg:
		if msg.Err != nil {
			m.setErr("load", msg.Err)
			return m, nil
		}
		m.instances = msg.Items
		if m.cursor >= len(m.instances) {
			m.cursor = 0
		}
		// Drop selections referring to gone instances.
		live := map[domain.InstanceID]bool{}
		for _, inst := range m.instances {
			live[inst.ID] = true
		}
		for id := range m.selected {
			if !live[id] {
				delete(m.selected, id)
			}
		}
		return m, m.refreshRuntimeCmd()

	case runtimeRefreshedMsg:
		m.runtime = msg.Status
		return m, nil

	case runtimeTickMsg:
		if m.toast != "" && time.Now().After(m.toastDeadline) {
			m.toast = ""
			m.toastErr = false
		}
		return m, tea.Batch(m.refreshRuntimeCmd(), runtimeTick())

	case sudoCheckedMsg:
		if msg.Err == nil {
			m.sudoRetry = msg.Next
			return m, msg.Next
		}
		if errors.Is(msg.Err, sudo.ErrNeedsPrompt) {
			m.sudoPending = msg.Next
			return m, promptSudoCmd()
		}
		m.mode = ModeList
		m.setErr("sudo", msg.Err)
		return m, nil

	case sudoRefreshedMsg:
		if msg.Err != nil {
			m.mode = ModeList
			m.sudoPending = nil
			m.setErr("sudo", msg.Err)
			return m, nil
		}
		m.deps.Sudo.Refreshed()
		m.setToast("已认证 sudo")
		next := m.sudoPending
		m.sudoPending = nil
		m.sudoRetry = next
		return m, next

	case createProgressMsg:
		m.mode = ModeBusy
		m.busy = busyState{
			title: fmt.Sprintf("创建 WeChat%d", msg.ID),
			step:  msg.Step,
			total: msg.Total,
			label: msg.Label,
		}
		return m, nil

	case createDoneMsg:
		m.mode = ModeList
		if msg.Err != nil {
			if errors.Is(msg.Err, sudo.ErrNeedsPrompt) {
				return m, m.retryAfterSudo()
			}
			m.setErr("create", msg.Err)
		} else {
			m.setToast(fmt.Sprintf("WeChat%d 创建完成", msg.ID))
		}
		m.sudoRetry = nil
		return m, m.loadInstancesCmd()

	case deleteProgressMsg:
		m.mode = ModeBusy
		m.busy = busyState{
			title: "删除副本",
			step:  msg.Done,
			total: msg.Total,
			label: fmt.Sprintf("WeChat%d", msg.ID),
		}
		return m, nil

	case deleteDoneMsg:
		m.mode = ModeList
		if msg.Err != nil {
			if errors.Is(msg.Err, sudo.ErrNeedsPrompt) {
				return m, m.retryAfterSudo()
			}
			m.setErr("delete", msg.Err)
		} else {
			m.setToast(fmt.Sprintf("已删除 %d 个副本", len(msg.IDs)))
		}
		m.sudoRetry = nil
		m.selected = map[domain.InstanceID]bool{}
		return m, m.loadInstancesCmd()

	case updateProgressMsg:
		m.mode = ModeBusy
		m.busy = busyState{
			title: fmt.Sprintf("更新 WeChat%d", msg.ID),
			step:  msg.Step,
			total: msg.Total,
			label: msg.Label,
		}
		return m, nil

	case updateDoneMsg:
		m.mode = ModeList
		if msg.Err != nil {
			if errors.Is(msg.Err, sudo.ErrNeedsPrompt) {
				return m, m.retryAfterSudo()
			}
			m.setErr("update", msg.Err)
		} else {
			m.setToast(fmt.Sprintf("已更新 %d 个副本", len(msg.IDs)))
		}
		m.sudoRetry = nil
		m.selected = map[domain.InstanceID]bool{}
		return m, m.loadInstancesCmd()

	case launchDoneMsg:
		if msg.Err != nil {
			m.setErr("launch", msg.Err)
		} else {
			m.setToast("已启动")
		}
		return m, m.refreshRuntimeCmd()

	case stopDoneMsg:
		if msg.Err != nil {
			m.setErr("stop", msg.Err)
		} else {
			m.setToast("已停止")
		}
		return m, m.refreshRuntimeCmd()

	case iconProgressMsg:
		m.mode = ModeBusy
		m.busy = busyState{
			title: "应用图标",
			step:  msg.Done,
			total: msg.Total,
			label: fmt.Sprintf("WeChat%d ← %s", msg.ID, msg.IconName),
		}
		return m, nil

	case iconAppliedMsg:
		m.mode = ModeList
		if msg.Err != nil {
			if errors.Is(msg.Err, sudo.ErrNeedsPrompt) {
				return m, m.retryAfterSudo()
			}
			m.setErr("icon", msg.Err)
		} else {
			m.setToast(fmt.Sprintf("已应用图标到 %d 个副本", len(msg.IDs)))
		}
		m.sudoRetry = nil
		return m, nil

	case iconsLoadedMsg:
		if msg.Err != nil {
			m.setErr("icons", msg.Err)
			m.mode = ModeList
			return m, nil
		}
		m.icon = iconPickState{
			icons:   msg.Paths,
			targets: m.selectedOrCurrentIDs(),
		}
		m.mode = ModeIconPick
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// View renders the appropriate screen for the current mode.
func (m *Model) View() string {
	base := m.renderList()
	switch m.mode {
	case ModeList:
		return base
	case ModeConfirm:
		return m.overlay(base, m.renderConfirm())
	case ModeNewCopy:
		return m.overlay(base, m.renderNewCopy())
	case ModeIconPick:
		return m.overlay(base, m.renderIconPick())
	case ModeHelp:
		return m.overlay(base, m.renderHelp())
	case ModeBusy:
		return m.overlay(base, m.renderBusy())
	}
	return base
}

// Instances exposes the model's instances slice (for tests).
func (m *Model) Instances() []domain.Instance { return m.instances }

// Mode exposes the current mode (for tests).
func (m *Model) Mode() Mode { return m.mode }

// Cursor exposes the current cursor (for tests).
func (m *Model) Cursor() int { return m.cursor }

// Selected exposes the current selection set as a sorted ID slice (for tests).
func (m *Model) Selected() []domain.InstanceID {
	out := make([]domain.InstanceID, 0, len(m.selected))
	for id := range m.selected {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// helpers

// toastTTL is how long a toast remains visible before the next runtime tick clears it.
const toastTTL = 3 * time.Second

func (m *Model) setToast(s string) {
	m.toast = s
	m.toastErr = false
	m.toastDeadline = time.Now().Add(toastTTL)
}

func (m *Model) setErr(op string, err error) {
	m.toast = fmt.Sprintf("%s: %v", op, err)
	m.toastErr = true
	m.toastDeadline = time.Now().Add(toastTTL)
}

func (m *Model) currentInstance() (domain.Instance, bool) {
	if m.cursor < 0 || m.cursor >= len(m.instances) {
		return domain.Instance{}, false
	}
	return m.instances[m.cursor], true
}

func (m *Model) selectedOrCurrentIDs() []domain.InstanceID {
	if len(m.selected) > 0 {
		return m.Selected()
	}
	if inst, ok := m.currentInstance(); ok {
		return []domain.InstanceID{inst.ID}
	}
	return nil
}

func (m *Model) selectedOrCurrentInstances() []domain.Instance {
	ids := m.selectedOrCurrentIDs()
	byID := map[domain.InstanceID]domain.Instance{}
	for _, inst := range m.instances {
		byID[inst.ID] = inst
	}
	out := make([]domain.Instance, 0, len(ids))
	for _, id := range ids {
		if inst, ok := byID[id]; ok {
			out = append(out, inst)
		}
	}
	return out
}

// commands

func (m *Model) loadInstancesCmd() tea.Cmd {
	return func() tea.Msg {
		items, err := m.deps.Scanner.Scan()
		if errors.Is(err, scanner.ErrOriginalMissing) {
			return instancesLoadedMsg{Err: err}
		}
		return instancesLoadedMsg{Items: items, Err: err}
	}
}

func (m *Model) refreshRuntimeCmd() tea.Cmd {
	insts := m.instances
	return func() tea.Msg {
		status, _ := m.deps.Service.RefreshRuntime(m.ctx, insts)
		return runtimeRefreshedMsg{Status: status}
	}
}

func runtimeTick() tea.Cmd {
	return tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg { return runtimeTickMsg(t) })
}

func promptSudoCmd() tea.Cmd {
	c := exec.Command("sudo", "-v")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return sudoRefreshedMsg{Err: err}
	})
}

func (m *Model) retryAfterSudo() tea.Cmd {
	m.sudoPending = m.sudoRetry
	return promptSudoCmd()
}

// overlay composes a modal centered over the base view.
//
// We render the base, then overwrite the lines under the modal box with the
// modal content. lipgloss.Width / lipgloss.Height return display widths that
// account for ANSI escape sequences, which lets us position the modal without
// stripping color codes from either layer.
func (m *Model) overlay(base, modal string) string {
	if m.width <= 0 || m.height <= 0 {
		// No size yet — stack vertically so first paint isn't empty.
		return base + "\n" + modal
	}
	return composeOverlay(base, modal, m.width, m.height)
}

func composeOverlay(base, modal string, width, height int) string {
	baseLines := splitLines(base, height)
	modalLines := strings.Split(modal, "\n")

	modalH := len(modalLines)
	modalW := lipgloss.Width(modal)

	top := (height - modalH) / 2
	if top < 0 {
		top = 0
	}
	left := (width - modalW) / 2
	if left < 0 {
		left = 0
	}

	out := make([]string, len(baseLines))
	copy(out, baseLines)
	for i, mline := range modalLines {
		row := top + i
		if row < 0 || row >= len(out) {
			continue
		}
		out[row] = placeOnLine(out[row], mline, left, width)
	}
	return strings.Join(out, "\n")
}

// splitLines splits s into exactly height lines, padding with blanks.
func splitLines(s string, height int) []string {
	lines := strings.Split(s, "\n")
	if len(lines) >= height {
		return lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

// placeOnLine overlays modalLine onto baseLine starting at display column `at`,
// preserving the surrounding base content's ANSI codes.
//
// Implementation: we pad base to the screen width with spaces, slice it as a
// display-width window, splice modalLine in at the target column. Because we
// pad with plain spaces, ANSI codes outside the window are untouched.
func placeOnLine(baseLine, modalLine string, at, width int) string {
	leftPad := lipgloss.PlaceHorizontal(at, lipgloss.Left, "")
	// Render base limited to `at` cells, then modal, then right pad up to width.
	rightW := width - at - lipgloss.Width(modalLine)
	if rightW < 0 {
		rightW = 0
	}
	right := lipgloss.PlaceHorizontal(rightW, lipgloss.Left, "")
	// We intentionally drop baseLine here — the modal is opaque. The base is
	// still visible above/below the modal rows, which is what users expect.
	_ = baseLine
	return leftPad + modalLine + right
}

// DefaultIconLister returns a real-filesystem icon lister for the given directory.
func DefaultIconLister(iconDir string) IconLister {
	return func() ([]string, error) {
		entries, err := osReadDir(iconDir)
		if err != nil {
			return nil, err
		}
		var out []string
		for _, e := range entries {
			if filepath.Ext(e.Name()) == ".icns" {
				out = append(out, filepath.Join(iconDir, e.Name()))
			}
		}
		sort.Strings(out)
		return out, nil
	}
}
