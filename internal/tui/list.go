package tui

import (
	"fmt"
	"strings"

	"github.com/mariexaos/wepod/internal/domain"
)

// renderList draws the main dashboard.
func (m *Model) renderList() string {
	s := m.styles
	var sb strings.Builder

	// Header
	dryNote := ""
	if m.deps.Service.DryRun() {
		dryNote = "  " + s.Danger.Render("[DRY-RUN]")
	}
	sb.WriteString(s.Title.Render("wepod"))
	sb.WriteString(s.Subtle.Render("  " + m.deps.Service.Config().AppsDir))
	sb.WriteString(dryNote)
	sb.WriteString("\n\n")

	// Table header
	sb.WriteString(s.HeaderRow.Render(
		fmt.Sprintf("  %-3s %-12s %-32s %-8s", "#", "名称", "Bundle ID", "状态"),
	))
	sb.WriteString("\n")

	// Rows
	for i, inst := range m.instances {
		sb.WriteString(m.renderRow(i, inst))
		sb.WriteString("\n")
	}

	if len(m.instances) == 0 {
		sb.WriteString(s.Subtle.Render("  (扫描中或目录为空)\n"))
	}

	sb.WriteString("\n")
	// Footer (status + help)
	sb.WriteString(m.renderFooter())
	return sb.String()
}

func (m *Model) renderRow(idx int, inst domain.Instance) string {
	s := m.styles
	selMark := " "
	if m.selected[inst.ID] {
		selMark = "*"
	}
	cursor := " "
	if idx == m.cursor {
		cursor = "▸"
	}
	rt := m.runtime[inst.ID]
	statusStr := s.Stopped.Render("未运行")
	if rt.Running {
		statusStr = s.Running.Render("运行中")
	}
	idStr := fmt.Sprintf("%d", inst.ID)
	if inst.IsOriginal() {
		idStr = "•"
	}
	line := fmt.Sprintf("%s %s %-3s %-12s %-32s %s", cursor, selMark, idStr, inst.Name, inst.BundleID, statusStr)
	if idx == m.cursor {
		return s.RowFocused.Render(line)
	}
	if m.selected[inst.ID] {
		return s.RowSelected.Render(line)
	}
	return s.Row.Render(line)
}

func (m *Model) renderFooter() string {
	s := m.styles
	help := "↑↓ 移动 · space 选 · enter 启动 · n 新建 · d 删 · s 停 · i 图标 · r 刷新 · ? 帮助 · q 退出"
	var toast string
	if m.toast != "" {
		if m.toastErr {
			toast = s.ToastErr.Render(m.toast)
		} else {
			toast = s.Toast.Render(m.toast)
		}
	} else {
		toast = s.Status.Render(fmt.Sprintf("已选 %d · 共 %d 实例", len(m.selected), len(m.instances)))
	}
	return toast + "\n" + s.HelpBar.Render(help)
}
