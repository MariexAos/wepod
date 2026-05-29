package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mariexaos/wepod/internal/domain"
)

type confirmKind int

const (
	confirmDelete confirmKind = iota
	confirmUpdate
)

type confirmState struct {
	title    string
	body     string
	danger   bool
	ids      []domain.InstanceID
	withData bool
	kind     confirmKind
}

var (
	errNoDeletable   = errors.New("当前选中项无可删除副本（原版不可删）")
	errNoUpdatable   = errors.New("当前选中项无可更新副本（原版即更新来源）")
	errNoIconTargets = errors.New("当前选中项无可换图标副本（原版不可换）")
)

func (m *Model) handleKeyConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.mode = ModeList
		return m, nil
	case msg.String() == "d" && m.confirm.kind == confirmDelete:
		// Toggle "also delete data dir".
		m.confirm.withData = !m.confirm.withData
		return m, nil
	case key.Matches(msg, m.keys.Confirm):
		return m.executeConfirm()
	case key.Matches(msg, m.keys.Deny):
		m.mode = ModeList
		return m, nil
	}
	return m, nil
}

func (m *Model) executeConfirm() (tea.Model, tea.Cmd) {
	switch m.confirm.kind {
	case confirmDelete:
		ids := append([]domain.InstanceID(nil), m.confirm.ids...)
		withData := m.confirm.withData
		m.mode = ModeBusy
		m.busy = busyState{title: "删除中", total: len(ids)}
		return m, m.deleteManyCmd(ids, withData)
	case confirmUpdate:
		ids := append([]domain.InstanceID(nil), m.confirm.ids...)
		m.mode = ModeBusy
		m.busy = busyState{title: "更新中", total: len(ids)}
		return m, m.updateManyCmd(ids)
	}
	m.mode = ModeList
	return m, nil
}

func deleteBody(ids []domain.InstanceID) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "将删除 %d 个副本：\n", len(ids))
	for _, id := range ids {
		fmt.Fprintf(&sb, "  • WeChat%d\n", id)
	}
	sb.WriteString("\n副本会移到 ~/.Trash/wepod-undo/ — 5 秒内可手动 mv 回去。")
	return sb.String()
}

func updateBody(ids []domain.InstanceID) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "将用当前 WeChat.app 重建 %d 个副本：\n", len(ids))
	for _, id := range ids {
		fmt.Fprintf(&sb, "  • WeChat%d\n", id)
	}
	sb.WriteString("\n会保留各副本的 Bundle ID、名称与自定义图标。\n请先退出对应副本——更新会覆盖其 .app。")
	return sb.String()
}

func (m *Model) renderConfirm() string {
	s := m.styles
	header := s.ModalTitle.Render(m.confirm.title)
	body := m.confirm.body
	hint := "[y] 确认  [n/esc] 取消"
	if m.confirm.kind == confirmDelete {
		state := "否"
		if m.confirm.withData {
			state = "是"
		}
		hint = fmt.Sprintf("[y] 确认  [n/esc] 取消  [d] 同时删数据目录: %s", state)
	}
	if m.confirm.danger {
		header = s.Danger.Render(m.confirm.title)
	}
	return s.Modal.Render(header + "\n\n" + body + "\n\n" + s.HelpBar.Render(hint))
}
