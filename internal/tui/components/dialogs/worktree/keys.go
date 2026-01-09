package worktree

import (
	"charm.land/bubbles/v2/key"
)

// KeyMap defines key bindings for the worktree dialog.
type KeyMap struct {
	Confirm key.Binding
	Close   key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "create worktree"),
		),
		Close: key.NewBinding(
			key.WithKeys("esc", "alt+esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// KeyBindings implements layout.KeyMapProvider.
func (k KeyMap) KeyBindings() []key.Binding {
	return []key.Binding{
		k.Confirm,
		k.Close,
	}
}

// FullHelp implements help.KeyMap.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.KeyBindings()}
}

// ShortHelp implements help.KeyMap.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Confirm,
		k.Close,
	}
}
