package mcpservers

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/agent/tools/mcp"
	"github.com/charmbracelet/crush/internal/tui/components/core"
	"github.com/charmbracelet/crush/internal/tui/components/dialogs"
	"github.com/charmbracelet/crush/internal/tui/exp/list"
	"github.com/charmbracelet/crush/internal/tui/styles"
	"github.com/charmbracelet/crush/internal/tui/util"
)

const MCPLogsDialogID dialogs.DialogID = "mcp-logs"

// LogListItem represents a log entry in the list.
type LogListItem struct {
	Entry mcp.LogEntry
}

type mcpLogsDialogCmp struct {
	wWidth         int
	wHeight        int
	width          int
	serverName     string
	keyMap         LogsKeyMap
	help           help.Model
	logsList       list.FilterableList[list.CompletionItem[LogListItem]]
	sessions       []mcp.LogSession
	currentSession string // Empty means all sessions.
	showDetail     bool
	selectedEntry  *mcp.LogEntry
	sessionIndex   int // For cycling through sessions.
}

// NewMCPLogsDialogCmp creates a new logs dialog for a specific MCP server.
func NewMCPLogsDialogCmp(serverName string) dialogs.DialogModel {
	t := styles.CurrentTheme()
	keyMap := DefaultLogsKeyMap()

	listKeyMap := list.DefaultKeyMap()
	listKeyMap.Down.SetEnabled(false)
	listKeyMap.Up.SetEnabled(false)
	listKeyMap.DownOneItem = keyMap.Next
	listKeyMap.UpOneItem = keyMap.Previous

	inputStyle := t.S().Base.PaddingLeft(1).PaddingBottom(1)
	logsList := list.NewFilterableList(
		[]list.CompletionItem[LogListItem]{},
		list.WithFilterPlaceholder("Search logs..."),
		list.WithFilterInputStyle(inputStyle),
		list.WithFilterListOptions(
			list.WithKeyMap(listKeyMap),
			list.WithWrapNavigation(),
		),
	)

	helpModel := help.New()
	helpModel.Styles = t.S().Help

	dialog := &mcpLogsDialogCmp{
		serverName: serverName,
		keyMap:     keyMap,
		help:       helpModel,
		logsList:   logsList,
	}

	return dialog
}

func (d *mcpLogsDialogCmp) Init() tea.Cmd {
	return tea.Batch(
		d.logsList.Init(),
		d.logsList.Focus(),
		d.loadLogsCmd(),
	)
}

type logsLoadedMsg struct {
	entries  []mcp.LogEntry
	sessions []mcp.LogSession
}

func (d *mcpLogsDialogCmp) loadLogsCmd() tea.Cmd {
	return func() tea.Msg {
		entries, _ := mcp.GetLogEntries(d.serverName, d.currentSession, 100)
		sessions, _ := mcp.GetSessions(d.serverName)
		return logsLoadedMsg{entries: entries, sessions: sessions}
	}
}

func (d *mcpLogsDialogCmp) createLogsList(entries []mcp.LogEntry) list.FilterableList[list.CompletionItem[LogListItem]] {
	t := styles.CurrentTheme()
	listKeyMap := list.DefaultKeyMap()
	listKeyMap.Down.SetEnabled(false)
	listKeyMap.Up.SetEnabled(false)
	listKeyMap.DownOneItem = d.keyMap.Next
	listKeyMap.UpOneItem = d.keyMap.Previous

	items := make([]list.CompletionItem[LogListItem], len(entries))

	for i, entry := range entries {
		status := "✓"
		if !entry.Success {
			status = "✗"
		}
		displayText := fmt.Sprintf("%s %s %s.%s (%dms)",
			entry.Timestamp.Format("15:04:05"),
			status,
			truncateStr(entry.ServerName, 15),
			truncateStr(entry.ToolName, 20),
			entry.DurationMs,
		)
		items[i] = list.NewCompletionItem(
			displayText,
			LogListItem{Entry: entry},
			list.WithCompletionID(fmt.Sprintf("%d", i)),
		)
	}

	inputStyle := t.S().Base.PaddingLeft(1).PaddingBottom(1)
	return list.NewFilterableList(
		items,
		list.WithFilterPlaceholder("Search logs..."),
		list.WithFilterInputStyle(inputStyle),
		list.WithFilterListOptions(
			list.WithKeyMap(listKeyMap),
			list.WithWrapNavigation(),
		),
	)
}

func (d *mcpLogsDialogCmp) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.wWidth = msg.Width
		d.wHeight = msg.Height
		d.width = min(140, d.wWidth-8)
		return d, d.logsList.SetSize(d.listWidth(), d.listHeight())

	case logsLoadedMsg:
		d.sessions = msg.sessions
		d.logsList = d.createLogsList(msg.entries)
		return d, tea.Batch(
			d.logsList.Init(),
			d.logsList.Focus(),
			d.logsList.SetSize(d.listWidth(), d.listHeight()),
		)

	case tea.KeyPressMsg:
		if d.showDetail {
			// Any key exits detail view.
			d.showDetail = false
			d.selectedEntry = nil
			return d, nil
		}

		switch {
		case key.Matches(msg, d.keyMap.FilterSession):
			// Cycle through sessions.
			if len(d.sessions) == 0 {
				return d, nil
			}
			d.sessionIndex = (d.sessionIndex + 1) % (len(d.sessions) + 1)
			if d.sessionIndex == 0 {
				d.currentSession = "" // All sessions.
			} else {
				d.currentSession = d.sessions[d.sessionIndex-1].ID
			}
			return d, d.loadLogsCmd()

		case key.Matches(msg, d.keyMap.Close):
			return d, util.CmdHandler(dialogs.CloseDialogMsg{})

		default:
			selected := d.logsList.SelectedItem()
			if selected != nil && key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				item := (*selected).Value()
				d.selectedEntry = &item.Entry
				d.showDetail = true
				return d, nil
			}

			u, cmd := d.logsList.Update(msg)
			d.logsList = u.(list.FilterableList[list.CompletionItem[LogListItem]])
			return d, cmd
		}
	}

	return d, nil
}

func (d *mcpLogsDialogCmp) View() string {
	t := styles.CurrentTheme()

	var content string
	if d.showDetail && d.selectedEntry != nil {
		content = d.renderDetailView()
	} else {
		content = d.renderListView()
	}

	title := fmt.Sprintf("Logs: %s", d.serverName)
	if d.currentSession != "" {
		title += fmt.Sprintf(" (session: %s)", truncateStr(d.currentSession, 8))
	}

	return d.style().Render(lipgloss.JoinVertical(
		lipgloss.Left,
		t.S().Base.Padding(0, 1, 1, 1).Render(core.Title(title, d.width-4)),
		content,
		"",
		t.S().Base.Width(d.width-2).PaddingLeft(1).AlignHorizontal(lipgloss.Left).Render(d.help.View(d.keyMap)),
	))
}

func (d *mcpLogsDialogCmp) renderListView() string {
	if d.logsList.Len() == 0 {
		return d.renderEmptyState()
	}
	return d.logsList.View()
}

func (d *mcpLogsDialogCmp) renderEmptyState() string {
	t := styles.CurrentTheme()
	style := t.S().Base.
		Width(d.listWidth()).
		Height(d.listHeight()).
		Padding(2).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Foreground(t.FgMuted)

	return style.Render("No logs available.\n\nInvoke MCP tools to generate logs.")
}

func (d *mcpLogsDialogCmp) renderDetailView() string {
	t := styles.CurrentTheme()
	entry := d.selectedEntry

	var lines []string
	lines = append(lines, fmt.Sprintf("Timestamp: %s", entry.Timestamp.Format(time.RFC3339)))
	lines = append(lines, fmt.Sprintf("Session:   %s", entry.SessionID))
	lines = append(lines, fmt.Sprintf("Server:    %s", entry.ServerName))
	lines = append(lines, fmt.Sprintf("Tool:      %s", entry.ToolName))
	lines = append(lines, fmt.Sprintf("Duration:  %dms", entry.DurationMs))
	lines = append(lines, fmt.Sprintf("Success:   %v", entry.Success))

	if entry.Error != "" {
		lines = append(lines, "")
		lines = append(lines, "Error:")
		lines = append(lines, entry.Error)
	}

	lines = append(lines, "")
	lines = append(lines, "Input:")
	inputPreview := entry.Input
	if len(inputPreview) > 500 {
		inputPreview = inputPreview[:500] + "..."
	}
	lines = append(lines, inputPreview)

	if entry.Output != "" {
		lines = append(lines, "")
		lines = append(lines, "Output:")
		outputPreview := entry.Output
		if len(outputPreview) > 500 {
			outputPreview = outputPreview[:500] + "..."
		}
		lines = append(lines, outputPreview)
	}

	style := t.S().Base.
		Width(d.listWidth()).
		Height(d.listHeight()).
		Padding(1)

	return style.Render(strings.Join(lines, "\n"))
}

func (d *mcpLogsDialogCmp) Cursor() *tea.Cursor {
	if d.showDetail {
		return nil
	}
	if cursor, ok := d.logsList.(util.Cursor); ok {
		cursor := cursor.Cursor()
		if cursor != nil {
			cursor = d.moveCursor(cursor)
		}
		return cursor
	}
	return nil
}

func (d *mcpLogsDialogCmp) style() lipgloss.Style {
	t := styles.CurrentTheme()
	return t.S().Base.
		Width(d.width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus)
}

func (d *mcpLogsDialogCmp) listHeight() int {
	return d.wHeight/2 - 6
}

func (d *mcpLogsDialogCmp) listWidth() int {
	return d.width - 2
}

func (d *mcpLogsDialogCmp) Position() (int, int) {
	row := d.wHeight/4 - 2
	col := d.wWidth / 2
	col -= d.width / 2
	return row, col
}

func (d *mcpLogsDialogCmp) moveCursor(cursor *tea.Cursor) *tea.Cursor {
	row, col := d.Position()
	offset := row + 3
	cursor.Y += offset
	cursor.X = cursor.X + col + 2
	return cursor
}

// ID implements DialogModel.
func (d *mcpLogsDialogCmp) ID() dialogs.DialogID {
	return MCPLogsDialogID
}
