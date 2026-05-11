package tui_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/tui"
)

// ----- fakes -----

type fakeService struct {
	mu sync.Mutex

	cfg     domain.Config
	dryRun  bool
	runtime map[domain.InstanceID]domain.Runtime

	createCalls []domain.InstanceID
	deleteCalls [][]domain.InstanceID
	launchCalls []domain.InstanceID
	stopCalls   []domain.InstanceID
	iconCalls   []iconCall
}

type iconCall struct {
	ids  []domain.InstanceID
	path string
}

func (f *fakeService) Config() domain.Config { return f.cfg }
func (f *fakeService) DryRun() bool          { return f.dryRun }

func (f *fakeService) Create(_ context.Context, id domain.InstanceID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls = append(f.createCalls, id)
	return nil
}

func (f *fakeService) Delete(_ context.Context, id domain.InstanceID, _ bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteCalls = append(f.deleteCalls, []domain.InstanceID{id})
	return nil
}

func (f *fakeService) DeleteMany(_ context.Context, ids []domain.InstanceID, _ bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteCalls = append(f.deleteCalls, ids)
	return nil
}

func (f *fakeService) Launch(_ context.Context, inst domain.Instance) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.launchCalls = append(f.launchCalls, inst.ID)
	return nil
}

func (f *fakeService) Stop(_ context.Context, inst domain.Instance) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopCalls = append(f.stopCalls, inst.ID)
	return nil
}

func (f *fakeService) StopMany(_ context.Context, insts []domain.Instance) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, inst := range insts {
		f.stopCalls = append(f.stopCalls, inst.ID)
	}
	return nil
}

func (f *fakeService) ApplyIcon(_ context.Context, inst domain.Instance, path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.iconCalls = append(f.iconCalls, iconCall{ids: []domain.InstanceID{inst.ID}, path: path})
	return nil
}

func (f *fakeService) ApplyIconMany(_ context.Context, insts []domain.Instance, path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := make([]domain.InstanceID, len(insts))
	for i, inst := range insts {
		ids[i] = inst.ID
	}
	f.iconCalls = append(f.iconCalls, iconCall{ids: ids, path: path})
	return nil
}

func (f *fakeService) RefreshRuntime(_ context.Context, _ []domain.Instance) (map[domain.InstanceID]domain.Runtime, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[domain.InstanceID]domain.Runtime{}
	for k, v := range f.runtime {
		out[k] = v
	}
	return out, nil
}

type fakeSudo struct{ refreshed int }

func (f *fakeSudo) Ensure(context.Context) error { return nil }
func (f *fakeSudo) Refreshed()                   { f.refreshed++ }

type fakeScanner struct{ items []domain.Instance }

func (f *fakeScanner) Scan() ([]domain.Instance, error) { return f.items, nil }

// ----- helpers -----

// runTestModel constructs a fresh Model and synchronously drains Init's commands.
// Tick messages (runtimeTickMsg) are filtered to avoid infinite loops.
func newTestModel(t *testing.T, items []domain.Instance) (*tui.Model, *fakeService) {
	t.Helper()
	svc := &fakeService{cfg: domain.DefaultConfig("/Users/t")}
	deps := tui.Deps{
		Service:    svc,
		Sudo:       &fakeSudo{},
		Scanner:    &fakeScanner{items: items},
		IconLister: func() ([]string, error) { return []string{"/i/a.icns", "/i/b.icns"}, nil },
	}
	m := tui.New(deps)
	drainCmd(t, m, m.Init())
	return m, svc
}

// runCmd invokes a Cmd in a goroutine with a short timeout. tea.Tick commands
// would otherwise block for their configured duration — we treat any cmd that
// doesn't return promptly as "produces a tick" and discard it.
func runCmd(c tea.Cmd) (tea.Msg, bool) {
	if c == nil {
		return nil, false
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- c() }()
	select {
	case msg := <-done:
		return msg, true
	case <-time.After(50 * time.Millisecond):
		return nil, false
	}
}

func drainCmd(t *testing.T, m *tui.Model, cmd tea.Cmd) {
	t.Helper()
	queue := []tea.Cmd{cmd}
	for iter := 0; iter < 50 && len(queue) > 0; iter++ {
		c := queue[0]
		queue = queue[1:]
		msg, ok := runCmd(c)
		if !ok || msg == nil {
			continue
		}
		if typeName(msg) == "tui.runtimeTickMsg" {
			continue
		}
		if batched, ok := msg.(tea.BatchMsg); ok {
			queue = append(queue, batched...)
			continue
		}
		_, next := m.Update(msg)
		if next != nil {
			queue = append(queue, next)
		}
	}
}

func typeName(v any) string {
	return fmt.Sprintf("%T", v)
}

// ----- tests -----

func TestInitialScanPopulatesInstances(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	a := cfg.Original()
	b, _ := cfg.Copy(2)
	m, _ := newTestModel(t, []domain.Instance{a, b})

	if got, want := len(m.Instances()), 2; got != want {
		t.Errorf("got %d instances, want %d", got, want)
	}
}

func TestKeyDownAdvancesCursor(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	a := cfg.Original()
	b, _ := cfg.Copy(2)
	m, _ := newTestModel(t, []domain.Instance{a, b})

	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.Cursor() != 1 {
		t.Errorf("cursor = %d, want 1", m.Cursor())
	}
}

func TestSpaceTogglesSelection(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	a := cfg.Original()
	b, _ := cfg.Copy(2)
	m, _ := newTestModel(t, []domain.Instance{a, b})

	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if got := m.Selected(); len(got) != 1 || got[0] != 0 {
		t.Errorf("Selected = %v, want [0]", got)
	}
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if got := m.Selected(); len(got) != 0 {
		t.Errorf("Selected = %v, want empty", got)
	}
}

func TestSelectAllToggle(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	a := cfg.Original()
	b, _ := cfg.Copy(2)
	c, _ := cfg.Copy(3)
	m, _ := newTestModel(t, []domain.Instance{a, b, c})

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if got := m.Selected(); len(got) != 3 {
		t.Errorf("after a: %v, want all 3", got)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if got := m.Selected(); len(got) != 0 {
		t.Errorf("after second a: %v, want empty", got)
	}
}

func TestPressDOpensConfirmModal(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	a := cfg.Original()
	b, _ := cfg.Copy(2)
	m, _ := newTestModel(t, []domain.Instance{a, b})

	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	if !strings.Contains(m.View(), "删除副本") {
		t.Errorf("expected confirm modal in view; got:\n%s", m.View())
	}
}

func TestPressDOnOriginalOnlyShowsError(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	a := cfg.Original()
	m, _ := newTestModel(t, []domain.Instance{a})

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !strings.Contains(m.View(), "原版不可删") {
		t.Errorf("expected error toast; got:\n%s", m.View())
	}
}

func TestEnterLaunchesCurrent(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	a := cfg.Original()
	m, svc := newTestModel(t, []domain.Instance{a})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	drainCmd(t, m, cmd)
	if len(svc.launchCalls) != 1 || svc.launchCalls[0] != 0 {
		t.Errorf("launchCalls = %v, want [0]", svc.launchCalls)
	}
}

func TestPressNOpensNewCopyForm(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	a := cfg.Original()
	m, _ := newTestModel(t, []domain.Instance{a})

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !strings.Contains(m.View(), "新建副本") {
		t.Errorf("expected new copy modal; got:\n%s", m.View())
	}
}

func TestQuestionMarkOpensHelp(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	m, _ := newTestModel(t, []domain.Instance{cfg.Original()})

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !strings.Contains(m.View(), "键位") {
		t.Errorf("expected help modal; got:\n%s", m.View())
	}
}

func TestConfirmDeleteFlowCallsDeleteMany(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	a := cfg.Original()
	b, _ := cfg.Copy(2)
	c, _ := cfg.Copy(3)
	m, svc := newTestModel(t, []domain.Instance{a, b, c})

	// Select 2 and 3, press d, press y.
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	drainCmd(t, m, cmd)

	if len(svc.deleteCalls) != 1 {
		t.Fatalf("deleteCalls = %v, want 1 batch", svc.deleteCalls)
	}
	got := svc.deleteCalls[0]
	if len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Errorf("deleted ids = %v, want [2 3]", got)
	}
}

func TestNewCopyFormSubmitsCreate(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	a := cfg.Original()
	m, svc := newTestModel(t, []domain.Instance{a})

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	// Prefilled with "2" — press Enter.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	drainCmd(t, m, cmd)
	if len(svc.createCalls) != 1 || svc.createCalls[0] != 2 {
		t.Errorf("createCalls = %v, want [2]", svc.createCalls)
	}
}

func TestIconPickAppliesToSelected(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	a := cfg.Original()
	b, _ := cfg.Copy(2)
	m, svc := newTestModel(t, []domain.Instance{a, b})

	m.Update(tea.KeyMsg{Type: tea.KeyDown})  // to b
	m.Update(tea.KeyMsg{Type: tea.KeySpace}) // select 2
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	drainCmd(t, m, cmd)
	if m.Mode() != tui.ModeIconPick {
		t.Fatalf("after i: mode = %s, want iconpick; view=\n%s", m.Mode(), m.View())
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	drainCmd(t, m, cmd)

	if len(svc.iconCalls) != 1 {
		t.Fatalf("iconCalls = %v, want 1", svc.iconCalls)
	}
	if svc.iconCalls[0].path != "/i/a.icns" {
		t.Errorf("icon path = %q, want /i/a.icns", svc.iconCalls[0].path)
	}
	if len(svc.iconCalls[0].ids) != 1 || svc.iconCalls[0].ids[0] != 2 {
		t.Errorf("icon ids = %v, want [2]", svc.iconCalls[0].ids)
	}
}

func TestStopWithoutSelectionStopsAll(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	a := cfg.Original()
	b, _ := cfg.Copy(2)
	m, svc := newTestModel(t, []domain.Instance{a, b})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	drainCmd(t, m, cmd)
	if len(svc.stopCalls) != 2 {
		t.Errorf("stopCalls = %v, want both ids stopped", svc.stopCalls)
	}
}
