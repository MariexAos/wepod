package tui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	Title         lipgloss.Style
	Subtle        lipgloss.Style
	HeaderRow     lipgloss.Style
	Row           lipgloss.Style
	RowSelected   lipgloss.Style
	RowFocused    lipgloss.Style
	Running       lipgloss.Style
	Stopped       lipgloss.Style
	Status        lipgloss.Style
	HelpBar       lipgloss.Style
	Modal         lipgloss.Style
	ModalTitle    lipgloss.Style
	Danger        lipgloss.Style
	Toast         lipgloss.Style
	ToastErr      lipgloss.Style
	Spinner       lipgloss.Style
}

func defaultStyles() styles {
	return styles{
		Title:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7DD3FC")),
		Subtle:       lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		HeaderRow:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245")),
		Row:          lipgloss.NewStyle(),
		RowSelected:  lipgloss.NewStyle().Foreground(lipgloss.Color("#A7F3D0")),
		RowFocused:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FBBF24")),
		Running:      lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")),
		Stopped:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		Status:       lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		HelpBar:      lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		Modal:        lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("#7DD3FC")),
		ModalTitle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7DD3FC")),
		Danger:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EF4444")),
		Toast:        lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")),
		ToastErr:     lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")),
		Spinner:      lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")),
	}
}
