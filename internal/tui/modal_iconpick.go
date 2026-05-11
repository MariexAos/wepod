package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mariexaos/wepod/internal/domain"
)

type iconPickState struct {
	icons   []string // absolute paths
	cursor  int
	targets []domain.InstanceID
}

func (m *Model) handleKeyIconPick(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.mode = ModeList
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.icon.cursor > 0 {
			m.icon.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.icon.cursor < len(m.icon.icons)-1 {
			m.icon.cursor++
		}
	case msg.Type == tea.KeyEnter:
		if m.icon.cursor < 0 || m.icon.cursor >= len(m.icon.icons) {
			return m, nil
		}
		path := m.icon.icons[m.icon.cursor]
		insts := instancesByIDs(m.instances, m.icon.targets)
		m.mode = ModeBusy
		m.busy = busyState{title: "应用图标", total: len(insts)}
		return m, m.applyIconCmd(insts, path)
	}
	return m, nil
}

func (m *Model) renderIconPick() string {
	s := m.styles
	if len(m.icon.icons) == 0 {
		return s.Modal.Render(s.ModalTitle.Render("选择图标") + "\n\n" + s.Subtle.Render("（未找到 .icns 文件）"))
	}
	var lines []string
	for i, p := range m.icon.icons {
		marker := "  "
		name := strings.TrimSuffix(filepath.Base(p), ".icns")
		if i == m.icon.cursor {
			marker = "▸ "
			name = s.RowFocused.Render(name)
		}
		lines = append(lines, marker+name)
	}
	body := strings.Join(lines, "\n")
	footer := fmt.Sprintf("\n\n应用到 %d 个副本 · [enter] 确认 · [esc] 取消", len(m.icon.targets))
	return s.Modal.Render(s.ModalTitle.Render("选择图标") + "\n\n" + body + s.Subtle.Render(footer))
}

func instancesByIDs(all []domain.Instance, ids []domain.InstanceID) []domain.Instance {
	byID := map[domain.InstanceID]domain.Instance{}
	for _, inst := range all {
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
