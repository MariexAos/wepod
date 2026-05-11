package tui_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/tui"
)

// Note: these tests drive the real Update loop through teatest. They are
// inherently slower than the synchronous tests in tui_test.go (each WaitFor
// polls every 50ms by default), so we cap each test with a tight FinalTimeout
// and use generous WaitFor durations to absorb CI jitter.

func newTeaModel(items []domain.Instance) (*tui.Model, *fakeService) {
	svc := &fakeService{cfg: domain.DefaultConfig("/Users/t")}
	deps := tui.Deps{
		Service:    svc,
		Sudo:       &fakeSudo{},
		Scanner:    &fakeScanner{items: items},
		IconLister: func() ([]string, error) { return []string{"/i/a.icns", "/i/b.icns"}, nil },
	}
	return tui.New(deps), svc
}

func waitForText(t *testing.T, r io.Reader, needle string) {
	t.Helper()
	teatest.WaitFor(t, r, func(b []byte) bool {
		return bytes.Contains(b, []byte(needle))
	}, teatest.WithCheckInterval(20*time.Millisecond), teatest.WithDuration(2*time.Second))
}

func TestTea_InitialScanRendersInstances(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	a := cfg.Original()
	b, _ := cfg.Copy(2)
	m, _ := newTeaModel([]domain.Instance{a, b})

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
	defer tm.Quit() //nolint:errcheck // best-effort cleanup

	waitForText(t, tm.Output(), "WeChat2")
}

func TestTea_DeleteFlow_NavigateSelectConfirm(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	a := cfg.Original()
	b, _ := cfg.Copy(2)
	m, svc := newTeaModel([]domain.Instance{a, b})

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
	waitForText(t, tm.Output(), "WeChat2")

	// Move to WeChat2, select it, press d, then y.
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	waitForText(t, tm.Output(), "将删除")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	// Wait until the fake registered the call.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		svc.mu.Lock()
		done := len(svc.deleteCalls) > 0
		svc.mu.Unlock()
		if done {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	svc.mu.Lock()
	defer svc.mu.Unlock()
	if len(svc.deleteCalls) != 1 || len(svc.deleteCalls[0]) != 1 || svc.deleteCalls[0][0] != 2 {
		t.Errorf("deleteCalls = %v, want [[2]]", svc.deleteCalls)
	}
}

func TestTea_NewCopyFormCreatesNextID(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	a := cfg.Original()
	m, svc := newTeaModel([]domain.Instance{a})

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
	waitForText(t, tm.Output(), "WeChat")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	waitForText(t, tm.Output(), "新建副本")

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		svc.mu.Lock()
		done := len(svc.createCalls) > 0
		svc.mu.Unlock()
		if done {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	svc.mu.Lock()
	defer svc.mu.Unlock()
	if len(svc.createCalls) != 1 || svc.createCalls[0] != 2 {
		t.Errorf("createCalls = %v, want [2]", svc.createCalls)
	}
}

func TestTea_QuitFromList(t *testing.T) {
	cfg := domain.DefaultConfig("/Users/t")
	m, _ := newTeaModel([]domain.Instance{cfg.Original()})

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))
	waitForText(t, tm.Output(), "WeChat")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
