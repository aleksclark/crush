package agents

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/charmbracelet/crush/internal/subagent"
	"github.com/charmbracelet/crush/internal/tui/components/core"
	"github.com/charmbracelet/crush/internal/tui/components/dialogs"
	"github.com/charmbracelet/crush/internal/tui/exp/list"
	"github.com/charmbracelet/crush/internal/tui/styles"
	"github.com/charmbracelet/crush/internal/tui/util"
)

// AgentsDialogID is the unique identifier for the agents dialog.
const AgentsDialogID dialogs.DialogID = "agents"

// AgentsDialog interface for the agents list dialog.
type AgentsDialog interface {
	dialogs.DialogModel
}

// AgentsList is the filterable list of subagents.
type AgentsList = list.FilterableList[list.CompletionItem[*subagent.Subagent]]

type agentsDialogCmp struct {
	wWidth     int
	wHeight    int
	width      int
	keyMap     KeyMap
	agentsList AgentsList
	help       help.Model
	agents     []*subagent.Subagent
	selected   *subagent.Subagent // Currently selected agent for detail view.
	showDetail bool               // Whether we're showing detail view.
}

// NewAgentsDialogCmp creates a new agents listing dialog.
func NewAgentsDialogCmp(agents []*subagent.Subagent) AgentsDialog {
	t := styles.CurrentTheme()
	listKeyMap := list.DefaultKeyMap()
	keyMap := DefaultKeyMap()
	listKeyMap.Down.SetEnabled(false)
	listKeyMap.Up.SetEnabled(false)
	listKeyMap.DownOneItem = keyMap.Next
	listKeyMap.UpOneItem = keyMap.Previous

	items := make([]list.CompletionItem[*subagent.Subagent], len(agents))
	for i, agent := range agents {
		// Include description in the display text.
		displayText := agent.Name
		if agent.Description != "" {
			displayText = fmt.Sprintf("%s - %s", agent.Name, truncate(agent.Description, 50))
		}
		items[i] = list.NewCompletionItem(
			displayText,
			agent,
			list.WithCompletionID(agent.Name),
		)
	}

	inputStyle := t.S().Base.PaddingLeft(1).PaddingBottom(1)
	agentsList := list.NewFilterableList(
		items,
		list.WithFilterPlaceholder("Search agents..."),
		list.WithFilterInputStyle(inputStyle),
		list.WithFilterListOptions(
			list.WithKeyMap(listKeyMap),
			list.WithWrapNavigation(),
		),
	)

	helpModel := help.New()
	helpModel.Styles = t.S().Help

	return &agentsDialogCmp{
		keyMap:     keyMap,
		agentsList: agentsList,
		help:       helpModel,
		agents:     agents,
	}
}

func (a *agentsDialogCmp) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, a.agentsList.Init())
	cmds = append(cmds, a.agentsList.Focus())
	return tea.Sequence(cmds...)
}

func (a *agentsDialogCmp) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		var cmds []tea.Cmd
		a.wWidth = msg.Width
		a.wHeight = msg.Height
		a.width = min(120, a.wWidth-8)
		a.agentsList.SetInputWidth(a.listWidth() - 2)
		cmds = append(cmds, a.agentsList.SetSize(a.listWidth(), a.listHeight()))
		return a, tea.Batch(cmds...)
	case tea.KeyPressMsg:
		if a.showDetail {
			// In detail view, any key goes back to list.
			a.showDetail = false
			a.selected = nil
			return a, nil
		}
		switch {
		case key.Matches(msg, a.keyMap.Select):
			selectedItem := a.agentsList.SelectedItem()
			if selectedItem != nil {
				a.selected = (*selectedItem).Value()
				a.showDetail = true
			}
			return a, nil
		case key.Matches(msg, a.keyMap.Close):
			return a, util.CmdHandler(dialogs.CloseDialogMsg{})
		default:
			u, cmd := a.agentsList.Update(msg)
			a.agentsList = u.(AgentsList)
			return a, cmd
		}
	}
	return a, nil
}

func (a *agentsDialogCmp) View() string {
	t := styles.CurrentTheme()

	var content string
	if a.showDetail && a.selected != nil {
		content = a.renderDetailView()
	} else {
		content = a.renderListView()
	}

	return a.style().Render(lipgloss.JoinVertical(
		lipgloss.Left,
		t.S().Base.Padding(0, 1, 1, 1).Render(core.Title(a.title(), a.width-4)),
		content,
		"",
		t.S().Base.Width(a.width-2).PaddingLeft(1).AlignHorizontal(lipgloss.Left).Render(a.help.View(a.keyMap)),
	))
}

func (a *agentsDialogCmp) title() string {
	if a.showDetail && a.selected != nil {
		return fmt.Sprintf("Agent: %s", a.selected.Name)
	}
	return fmt.Sprintf("Agents (%d)", len(a.agents))
}

func (a *agentsDialogCmp) renderListView() string {
	if len(a.agents) == 0 {
		return a.renderEmptyState()
	}
	return a.agentsList.View()
}

func (a *agentsDialogCmp) renderEmptyState() string {
	t := styles.CurrentTheme()
	style := t.S().Base.
		Width(a.listWidth()).
		Height(a.listHeight()).
		Padding(2).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Foreground(t.FgMuted)

	return style.Render("No agents found.\n\nCreate agents in .crush/agents/ or .claude/agents/")
}

func (a *agentsDialogCmp) renderDetailView() string {
	t := styles.CurrentTheme()
	agent := a.selected

	var lines []string

	// Description.
	if agent.Description != "" {
		lines = append(lines, agent.Description)
		lines = append(lines, "")
	}

	// Model.
	if agent.Model != "" {
		lines = append(lines, fmt.Sprintf("Model: %s", agent.Model))
	}

	// Permission mode.
	if agent.PermissionMode != "" {
		lines = append(lines, fmt.Sprintf("Permission Mode: %s", agent.PermissionMode))
	}

	// Tools.
	if len(agent.Tools) > 0 {
		lines = append(lines, fmt.Sprintf("Allowed Tools: %s", strings.Join(agent.Tools, ", ")))
	}

	// Disallowed tools.
	if len(agent.DisallowedTools) > 0 {
		lines = append(lines, fmt.Sprintf("Disallowed Tools: %s", strings.Join(agent.DisallowedTools, ", ")))
	}

	// Skills.
	if len(agent.Skills) > 0 {
		lines = append(lines, fmt.Sprintf("Skills: %s", strings.Join(agent.Skills, ", ")))
	}

	// File path (shortened).
	if agent.FilePath != "" {
		shortPath := filepath.Base(filepath.Dir(agent.FilePath)) + "/" + filepath.Base(agent.FilePath)
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Source: %s", shortPath))
	}

	// System prompt preview.
	if agent.SystemPrompt != "" {
		lines = append(lines, "")
		promptPreview := agent.SystemPrompt
		if len(promptPreview) > 200 {
			promptPreview = promptPreview[:200] + "..."
		}
		// Remove newlines for preview.
		promptPreview = strings.ReplaceAll(promptPreview, "\n", " ")
		lines = append(lines, fmt.Sprintf("Prompt: %s", promptPreview))
	}

	style := t.S().Base.
		Width(a.listWidth()).
		Height(a.listHeight()).
		Padding(1)

	return style.Render(strings.Join(lines, "\n"))
}

func (a *agentsDialogCmp) Cursor() *tea.Cursor {
	if a.showDetail {
		return nil
	}
	if cursor, ok := a.agentsList.(util.Cursor); ok {
		cursor := cursor.Cursor()
		if cursor != nil {
			cursor = a.moveCursor(cursor)
		}
		return cursor
	}
	return nil
}

func (a *agentsDialogCmp) style() lipgloss.Style {
	t := styles.CurrentTheme()
	return t.S().Base.
		Width(a.width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus)
}

func (a *agentsDialogCmp) listHeight() int {
	return a.wHeight/2 - 6
}

func (a *agentsDialogCmp) listWidth() int {
	return a.width - 2
}

func (a *agentsDialogCmp) Position() (int, int) {
	row := a.wHeight/4 - 2
	col := a.wWidth / 2
	col -= a.width / 2
	return row, col
}

func (a *agentsDialogCmp) moveCursor(cursor *tea.Cursor) *tea.Cursor {
	row, col := a.Position()
	offset := row + 3
	cursor.Y += offset
	cursor.X = cursor.X + col + 2
	return cursor
}

// ID implements AgentsDialog.
func (a *agentsDialogCmp) ID() dialogs.DialogID {
	return AgentsDialogID
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
