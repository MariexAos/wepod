package tui

import (
	"fmt"
	"strings"
)

type busyState struct {
	title string
	step  int
	total int
	label string
}

func (m *Model) renderBusy() string {
	s := m.styles
	header := s.ModalTitle.Render(m.busy.title)

	bar := progressBar(m.busy.step, m.busy.total, 32)
	body := fmt.Sprintf("%s  %d / %d\n%s", bar, m.busy.step, m.busy.total, s.Subtle.Render(m.busy.label))

	return s.Modal.Render(header + "\n\n" + body)
}

// progressBar renders a width-character bar; "□" empty, "■" filled.
func progressBar(step, total, width int) string {
	if total <= 0 {
		return strings.Repeat("□", width)
	}
	filled := step * width / total
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("■", filled) + strings.Repeat("□", width-filled)
}
