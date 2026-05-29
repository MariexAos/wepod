package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/ops"
)

// programSender is satisfied by *tea.Program; abstracted for tests.
type programSender interface {
	Send(tea.Msg)
}

// ProgramSink adapts a *tea.Program into ops.ProgressSink.
type ProgramSink struct{ p programSender }

// NewProgramSink wraps a *tea.Program (or any compatible sender).
func NewProgramSink(p programSender) *ProgramSink { return &ProgramSink{p: p} }

// Send translates ops events into the TUI's message types.
func (s *ProgramSink) Send(v any) {
	if s.p == nil {
		return
	}
	switch ev := v.(type) {
	case ops.CreateProgressEvent:
		s.p.Send(createProgressMsg(ev))
	case ops.DeleteProgressEvent:
		s.p.Send(deleteProgressMsg(ev))
	case ops.UpdateProgressEvent:
		s.p.Send(updateProgressMsg(ev))
	case ops.IconApplyEvent:
		s.p.Send(iconProgressMsg(ev))
	default:
		// Drop unknown event types silently rather than spamming the message bus.
	}
}

func (m *Model) createCmd(id domain.InstanceID) tea.Cmd {
	return func() tea.Msg {
		err := m.deps.Service.Create(m.ctx, id)
		return createDoneMsg{ID: id, Err: err}
	}
}

func (m *Model) deleteManyCmd(ids []domain.InstanceID, withData bool) tea.Cmd {
	return func() tea.Msg {
		err := m.deps.Service.DeleteMany(m.ctx, ids, withData)
		return deleteDoneMsg{IDs: ids, Err: err}
	}
}

func (m *Model) updateManyCmd(ids []domain.InstanceID) tea.Cmd {
	return func() tea.Msg {
		err := m.deps.Service.UpdateMany(m.ctx, ids)
		return updateDoneMsg{IDs: ids, Err: err}
	}
}

func (m *Model) launchCmd(inst domain.Instance) tea.Cmd {
	return func() tea.Msg {
		err := m.deps.Service.Launch(m.ctx, inst)
		return launchDoneMsg{ID: inst.ID, Err: err}
	}
}

func (m *Model) stopManyCmd(insts []domain.Instance) tea.Cmd {
	return func() tea.Msg {
		err := m.deps.Service.StopMany(m.ctx, insts)
		return stopDoneMsg{Err: err}
	}
}

func (m *Model) applyIconCmd(insts []domain.Instance, iconPath string) tea.Cmd {
	return func() tea.Msg {
		err := m.deps.Service.ApplyIconMany(m.ctx, insts, iconPath)
		ids := make([]domain.InstanceID, len(insts))
		for i, inst := range insts {
			ids[i] = inst.ID
		}
		return iconAppliedMsg{IDs: ids, Err: err}
	}
}

func (m *Model) loadIconsCmd() tea.Cmd {
	lister := m.deps.IconLister
	return func() tea.Msg {
		if lister == nil {
			return iconsLoadedMsg{Paths: nil}
		}
		paths, err := lister()
		return iconsLoadedMsg{Paths: paths, Err: err}
	}
}
