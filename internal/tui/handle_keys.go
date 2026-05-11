package tui

import (
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/scanner"
)

// handleKey dispatches to the active mode's key handler.
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Quit shortcut works everywhere.
	if key.Matches(msg, m.keys.Quit) && m.mode == ModeList {
		m.cancel()
		return m, tea.Quit
	}
	switch m.mode {
	case ModeList:
		return m.handleKeyList(msg)
	case ModeConfirm:
		return m.handleKeyConfirm(msg)
	case ModeNewCopy:
		return m.handleKeyNewCopy(msg)
	case ModeIconPick:
		return m.handleKeyIconPick(msg)
	case ModeHelp:
		if key.Matches(msg, m.keys.Cancel) || key.Matches(msg, m.keys.Help) || key.Matches(msg, m.keys.Quit) {
			m.mode = ModeList
		}
		return m, nil
	case ModeBusy:
		// busy state ignores keys (operation is in-flight)
		return m, nil
	}
	return m, nil
}

func (m *Model) handleKeyList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := m.keys
	switch {
	case key.Matches(msg, k.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, k.Down):
		if m.cursor < len(m.instances)-1 {
			m.cursor++
		}
	case key.Matches(msg, k.Select):
		if inst, ok := m.currentInstance(); ok {
			if m.selected[inst.ID] {
				delete(m.selected, inst.ID)
			} else {
				m.selected[inst.ID] = true
			}
		}
	case key.Matches(msg, k.SelectAll):
		if len(m.selected) == len(m.instances) {
			m.selected = map[domain.InstanceID]bool{}
		} else {
			for _, inst := range m.instances {
				m.selected[inst.ID] = true
			}
		}
	case key.Matches(msg, k.Refresh):
		return m, m.loadInstancesCmd()
	case key.Matches(msg, k.Help):
		m.mode = ModeHelp
	case key.Matches(msg, k.New):
		return m.openNewCopy()
	case key.Matches(msg, k.Launch):
		return m.startLaunch()
	case key.Matches(msg, k.Stop):
		return m.startStop()
	case key.Matches(msg, k.Delete):
		return m.openDeleteConfirm()
	case key.Matches(msg, k.Icon):
		return m.openIconPick()
	}
	return m, nil
}

// openNewCopy sets up the new-copy form with the next available ID prefilled.
func (m *Model) openNewCopy() (tea.Model, tea.Cmd) {
	next, err := scanner.NextAvailableID(m.instances)
	if err != nil {
		m.setErr("new", err)
		return m, nil
	}
	m.newcopy = newCopyState{
		input:  strconv.Itoa(int(next)),
		hint:   "回车确认 · esc 取消 · 范围 2-99",
		nextID: next,
	}
	m.mode = ModeNewCopy
	return m, nil
}

func (m *Model) startLaunch() (tea.Model, tea.Cmd) {
	insts := m.selectedOrCurrentInstances()
	if len(insts) == 0 {
		return m, nil
	}
	cmds := make([]tea.Cmd, 0, len(insts))
	for _, inst := range insts {
		cmds = append(cmds, m.launchCmd(inst))
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) startStop() (tea.Model, tea.Cmd) {
	var insts []domain.Instance
	if len(m.selected) == 0 {
		insts = m.instances
	} else {
		insts = m.selectedOrCurrentInstances()
	}
	return m, m.stopManyCmd(insts)
}

func (m *Model) openDeleteConfirm() (tea.Model, tea.Cmd) {
	ids := m.selectedOrCurrentIDs()
	// Filter out the original — never deletable.
	clean := make([]domain.InstanceID, 0, len(ids))
	for _, id := range ids {
		if !id.IsOriginal() {
			clean = append(clean, id)
		}
	}
	if len(clean) == 0 {
		m.setErr("delete", errNoDeletable)
		return m, nil
	}
	m.confirm = confirmState{
		title:    "删除副本",
		body:     deleteBody(clean),
		danger:   true,
		ids:      clean,
		withData: false,
		kind:     confirmDelete,
	}
	m.mode = ModeConfirm
	return m, nil
}

func (m *Model) openIconPick() (tea.Model, tea.Cmd) {
	// Filter to copies only.
	ids := m.selectedOrCurrentIDs()
	clean := make([]domain.InstanceID, 0, len(ids))
	for _, id := range ids {
		if !id.IsOriginal() {
			clean = append(clean, id)
		}
	}
	if len(clean) == 0 {
		m.setErr("icon", errNoIconTargets)
		return m, nil
	}
	return m, m.loadIconsCmd()
}
