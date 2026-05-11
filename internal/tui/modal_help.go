package tui

import "strings"

func (m *Model) renderHelp() string {
	s := m.styles
	rows := [][2]string{
		{"↑/↓ · j/k", "上下移动"},
		{"space", "选中/取消选中"},
		{"a", "全选/取消全选"},
		{"enter", "启动当前/已选"},
		{"n", "新建副本"},
		{"d", "删除已选（确认模态可再按 d 切换是否删数据目录）"},
		{"s", "停止已选（无选中则停全部）"},
		{"i", "应用图标到已选"},
		{"r", "重新扫描"},
		{"?", "帮助"},
		{"esc", "关闭模态"},
		{"q · ctrl+c", "退出"},
	}
	var sb strings.Builder
	sb.WriteString(s.ModalTitle.Render("键位"))
	sb.WriteString("\n\n")
	for _, row := range rows {
		sb.WriteString(s.RowFocused.Render(row[0]))
		sb.WriteString("  ")
		sb.WriteString(row[1])
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(s.Subtle.Render("按任意键关闭"))
	return s.Modal.Render(sb.String())
}
