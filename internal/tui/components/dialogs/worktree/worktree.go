// Package worktree provides a dialog for creating git worktrees for sessions.
package worktree

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/tui/components/dialogs"
	"github.com/charmbracelet/crush/internal/tui/styles"
	"github.com/charmbracelet/crush/internal/tui/util"
	wt "github.com/charmbracelet/crush/internal/worktree"
)

const dialogID dialogs.DialogID = "worktree"

// ShowWorktreeDialogMsg requests opening the worktree dialog.
type ShowWorktreeDialogMsg struct{}

// WorktreeCreatedMsg is sent when a worktree is successfully created.
type WorktreeCreatedMsg struct {
	Name string
	Path string
}

// Dialog is the worktree name input dialog.
type Dialog interface {
	dialogs.DialogModel
}

type dialog struct {
	wWidth, wHeight int
	width, height   int

	input    textinput.Model
	keys     KeyMap
	help     help.Model
	onSubmit func(name string) tea.Cmd
	err      string
}

// NewDialog creates a new worktree dialog.
func NewDialog(onSubmit func(name string) tea.Cmd) Dialog {
	t := styles.CurrentTheme()

	ti := textinput.New()
	ti.Placeholder = "feature-name"
	ti.SetWidth(40)
	ti.SetVirtualCursor(false)
	ti.Prompt = ""
	ti.SetStyles(t.S().TextInput)
	ti.Focus()

	return &dialog{
		input:    ti,
		keys:     DefaultKeyMap(),
		help:     help.New(),
		width:    60,
		onSubmit: onSubmit,
	}
}

// Init implements tea.Model.
func (d *dialog) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (d *dialog) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.wWidth = msg.Width
		d.wHeight = msg.Height
		d.width = min(70, d.wWidth)
		d.height = min(12, d.wHeight)
		d.input.SetWidth(d.width - (paddingHorizontal * 2))
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, d.keys.Close):
			return d, util.CmdHandler(dialogs.CloseDialogMsg{})
		case key.Matches(msg, d.keys.Confirm):
			name := d.input.Value()
			if name == "" {
				d.err = "Name is required"
				return d, nil
			}
			sanitized := wt.SanitizeName(name)
			if sanitized == "" {
				d.err = "Invalid name"
				return d, nil
			}
			return d, tea.Sequence(
				util.CmdHandler(dialogs.CloseDialogMsg{}),
				d.onSubmit(sanitized),
			)
		default:
			var cmd tea.Cmd
			d.input, cmd = d.input.Update(msg)
			d.err = ""
			return d, cmd
		}
	case tea.PasteMsg:
		var cmd tea.Cmd
		d.input, cmd = d.input.Update(msg)
		d.err = ""
		return d, cmd
	}
	return d, nil
}

// View implements tea.Model.
func (d *dialog) View() string {
	t := styles.CurrentTheme()
	baseStyle := t.S().Base

	title := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Padding(0, 1).
		Render("New Worktree Session")

	description := t.S().Text.
		Padding(0, 1).
		Foreground(t.FgMuted).
		Render("Enter a name for the git worktree branch")

	labelStyle := baseStyle.Padding(1, 1, 0, 1).Foreground(t.FgBase).Bold(true)
	label := labelStyle.Render("Session name:")

	field := t.S().Text.
		Padding(0, 1).
		Render(d.input.View())

	inputField := lipgloss.JoinVertical(lipgloss.Left, label, field)

	elements := []string{title, description, inputField}

	if d.err != "" {
		errStyle := baseStyle.Padding(0, 1).Foreground(t.Error)
		elements = append(elements, errStyle.Render(d.err))
	}

	d.help.ShowAll = false
	helpText := baseStyle.Padding(0, 1).Render(d.help.View(d.keys))
	elements = append(elements, "", helpText)

	content := lipgloss.JoinVertical(lipgloss.Left, elements...)

	return baseStyle.Padding(1, 1, 0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus).
		Width(d.width).
		Render(content)
}

// Cursor implements dialogs.DialogModel.
func (d *dialog) Cursor() *tea.Cursor {
	cursor := d.input.Cursor()
	if cursor != nil {
		cursor = d.moveCursor(cursor)
	}
	return cursor
}

const (
	headerHeight      = 4
	paddingHorizontal = 3
)

func (d *dialog) moveCursor(cursor *tea.Cursor) *tea.Cursor {
	row, col := d.Position()
	offset := row + headerHeight
	cursor.Y += offset
	cursor.X = cursor.X + col + paddingHorizontal
	return cursor
}

// Position implements dialogs.DialogModel.
func (d *dialog) Position() (int, int) {
	row := (d.wHeight / 2) - (d.height / 2)
	col := (d.wWidth / 2) - (d.width / 2)
	return row, col
}

// ID implements dialogs.DialogModel.
func (d *dialog) ID() dialogs.DialogID {
	return dialogID
}
