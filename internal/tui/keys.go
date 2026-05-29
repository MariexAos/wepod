package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	Select    key.Binding
	SelectAll key.Binding
	Launch    key.Binding
	New       key.Binding
	Delete    key.Binding
	Stop      key.Binding
	Update    key.Binding
	Icon      key.Binding
	Refresh   key.Binding
	Help      key.Binding
	Cancel    key.Binding
	Quit      key.Binding
	Confirm   key.Binding
	Deny      key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Select:    key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle select")),
		SelectAll: key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "select all")),
		Launch:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "launch")),
		New:       key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new copy")),
		Delete:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Stop:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "stop")),
		Update:    key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update")),
		Icon:      key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "icon")),
		Refresh:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Cancel:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Confirm:   key.NewBinding(key.WithKeys("y", "Y")),
		Deny:      key.NewBinding(key.WithKeys("n", "N")),
	}
}
