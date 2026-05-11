package tui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mariexaos/wepod/internal/domain"
)

type newCopyState struct {
	input  string
	hint   string
	nextID domain.InstanceID
	err    string
}

func (m *Model) handleKeyNewCopy(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.mode = ModeList
		return m, nil
	case msg.Type == tea.KeyEnter:
		return m.submitNewCopy()
	case msg.Type == tea.KeyBackspace:
		if len(m.newcopy.input) > 0 {
			m.newcopy.input = m.newcopy.input[:len(m.newcopy.input)-1]
		}
	case msg.Type == tea.KeyRunes:
		for _, r := range msg.Runes {
			if r >= '0' && r <= '9' {
				m.newcopy.input += string(r)
			}
		}
	}
	return m, nil
}

func (m *Model) submitNewCopy() (tea.Model, tea.Cmd) {
	n, err := strconv.Atoi(m.newcopy.input)
	if err != nil {
		m.newcopy.err = "请输入数字"
		return m, nil
	}
	id := domain.InstanceID(n)
	if !id.IsValidCopy() {
		m.newcopy.err = fmt.Sprintf("范围 %d-%d", domain.MinCopyID, domain.MaxCopyID)
		return m, nil
	}
	for _, inst := range m.instances {
		if inst.ID == id {
			m.newcopy.err = fmt.Sprintf("WeChat%d 已存在", id)
			return m, nil
		}
	}
	m.mode = ModeBusy
	m.busy = busyState{title: fmt.Sprintf("创建 WeChat%d", id), total: 6}
	return m, m.createCmd(id)
}

func (m *Model) renderNewCopy() string {
	s := m.styles
	header := s.ModalTitle.Render("新建副本")
	body := fmt.Sprintf("编号: %s_\n%s", m.newcopy.input, s.Subtle.Render(m.newcopy.hint))
	if m.newcopy.err != "" {
		body += "\n" + s.ToastErr.Render(m.newcopy.err)
	}
	return s.Modal.Render(header + "\n\n" + body)
}
