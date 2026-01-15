package mcpservers

import (
	"charm.land/bubbles/v2/key"
)

// KeyMap defines key bindings for the MCP servers dialog.
type KeyMap struct {
	Select,
	Next,
	Previous,
	Restart,
	ViewLogs,
	Close key.Binding
}

// DefaultKeyMap returns the default key bindings for the MCP servers dialog.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "view details"),
		),
		Next: key.NewBinding(
			key.WithKeys("down", "ctrl+n"),
			key.WithHelp("↓", "next"),
		),
		Previous: key.NewBinding(
			key.WithKeys("up", "ctrl+p"),
			key.WithHelp("↑", "prev"),
		),
		Restart: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restart"),
		),
		ViewLogs: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "logs"),
		),
		Close: key.NewBinding(
			key.WithKeys("esc", "alt+esc"),
			key.WithHelp("esc", "back/close"),
		),
	}
}

// KeyBindings implements layout.KeyMapProvider.
func (k KeyMap) KeyBindings() []key.Binding {
	return []key.Binding{
		k.Select,
		k.Next,
		k.Previous,
		k.Restart,
		k.ViewLogs,
		k.Close,
	}
}

// FullHelp implements help.KeyMap.
func (k KeyMap) FullHelp() [][]key.Binding {
	m := [][]key.Binding{}
	slice := k.KeyBindings()
	for i := 0; i < len(slice); i += 4 {
		end := min(i+4, len(slice))
		m = append(m, slice[i:end])
	}
	return m
}

// ShortHelp implements help.KeyMap.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("down", "up"),
			key.WithHelp("↑↓", "choose"),
		),
		k.Select,
		k.Restart,
		k.ViewLogs,
		k.Close,
	}
}

// DetailKeyMap defines key bindings for the detail view.
type DetailKeyMap struct {
	Select,
	Next,
	Previous,
	Tab,
	Close key.Binding
}

// DefaultDetailKeyMap returns the default key bindings for the detail view.
func DefaultDetailKeyMap() DetailKeyMap {
	return DetailKeyMap{
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "view"),
		),
		Next: key.NewBinding(
			key.WithKeys("down", "ctrl+n"),
			key.WithHelp("↓", "next"),
		),
		Previous: key.NewBinding(
			key.WithKeys("up", "ctrl+p"),
			key.WithHelp("↑", "prev"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch tab"),
		),
		Close: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
	}
}

// KeyBindings implements layout.KeyMapProvider.
func (k DetailKeyMap) KeyBindings() []key.Binding {
	return []key.Binding{
		k.Select,
		k.Next,
		k.Previous,
		k.Tab,
		k.Close,
	}
}

// FullHelp implements help.KeyMap.
func (k DetailKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.KeyBindings()}
}

// ShortHelp implements help.KeyMap.
func (k DetailKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("down", "up"),
			key.WithHelp("↑↓", "choose"),
		),
		k.Tab,
		k.Select,
		k.Close,
	}
}

// LogsKeyMap defines key bindings for the logs view.
type LogsKeyMap struct {
	Next,
	Previous,
	FilterSession,
	Close key.Binding
}

// DefaultLogsKeyMap returns the default key bindings for the logs view.
func DefaultLogsKeyMap() LogsKeyMap {
	return LogsKeyMap{
		Next: key.NewBinding(
			key.WithKeys("down", "ctrl+n"),
			key.WithHelp("↓", "next"),
		),
		Previous: key.NewBinding(
			key.WithKeys("up", "ctrl+p"),
			key.WithHelp("↑", "prev"),
		),
		FilterSession: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "filter session"),
		),
		Close: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
	}
}

// KeyBindings implements layout.KeyMapProvider.
func (k LogsKeyMap) KeyBindings() []key.Binding {
	return []key.Binding{
		k.Next,
		k.Previous,
		k.FilterSession,
		k.Close,
	}
}

// FullHelp implements help.KeyMap.
func (k LogsKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.KeyBindings()}
}

// ShortHelp implements help.KeyMap.
func (k LogsKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(
			key.WithKeys("down", "up"),
			key.WithHelp("↑↓", "scroll"),
		),
		k.FilterSession,
		k.Close,
	}
}
