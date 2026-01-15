package mcpservers

import (
	"context"
	"fmt"
	"maps"
	"sync"

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

const (
	// MCPServersDialogID is the unique identifier for the MCP servers dialog.
	MCPServersDialogID dialogs.DialogID = "mcp-servers"
	serverStatusWidth  int              = 20
)

// ServerItem represents a single MCP server in the list.
type ServerItem struct {
	Name   string
	State  mcp.State
	Counts mcp.Counts
	Error  error
}

// MCPServersDialog implements the dialog interface for listing MCP servers.
type MCPServersDialog interface {
	dialogs.DialogModel
}

type mcpServersDialogCmp struct {
	wWidth      int
	wHeight     int
	width       int
	keyMap      KeyMap
	serversList list.FilterableList[list.CompletionItem[ServerItem]]
	help        help.Model
	loading     bool
	restarting  map[string]bool
	mu          sync.Mutex
	ctx         context.Context
}

// NewMCPServersDialogCmp creates a new MCP servers dialog.
func NewMCPServersDialogCmp(ctx context.Context) MCPServersDialog {
	t := styles.CurrentTheme()
	listKeyMap := list.DefaultKeyMap()
	keyMap := DefaultKeyMap()
	listKeyMap.Down.SetEnabled(false)
	listKeyMap.Up.SetEnabled(false)
	listKeyMap.DownOneItem = keyMap.Next
	listKeyMap.UpOneItem = keyMap.Previous

	inputStyle := t.S().Base.PaddingLeft(1).PaddingBottom(1)
	serversList := list.NewFilterableList(
		[]list.CompletionItem[ServerItem]{},
		list.WithFilterPlaceholder("Search MCP servers..."),
		list.WithFilterInputStyle(inputStyle),
		list.WithFilterListOptions(
			list.WithKeyMap(listKeyMap),
			list.WithWrapNavigation(),
		),
	)

	helpModel := help.New()
	helpModel.Styles = t.S().Help

	return &mcpServersDialogCmp{
		ctx:         ctx,
		keyMap:      keyMap,
		help:        helpModel,
		serversList: serversList,
		restarting:  make(map[string]bool),
	}
}

func (d *mcpServersDialogCmp) Init() tea.Cmd {
	return tea.Batch(
		d.serversList.Init(),
		d.serversList.Focus(),
		d.loadServersCmd(),
	)
}

func (d *mcpServersDialogCmp) loadServersCmd() tea.Cmd {
	return func() tea.Msg {
		return d.loadServers()
	}
}

type serversLoadedMsg struct {
	items []list.CompletionItem[ServerItem]
}

func (d *mcpServersDialogCmp) loadServers() tea.Msg {
	d.mu.Lock()
	restartingCopy := maps.Clone(d.restarting)
	d.mu.Unlock()

	t := styles.CurrentTheme()
	servers := mcp.GetAllServerNames()
	items := make([]list.CompletionItem[ServerItem], len(servers))

	for i, name := range servers {
		state, ok := mcp.GetState(name)

		var serverState mcp.State
		var serverErr error
		var counts mcp.Counts
		if ok {
			serverState = state.State
			serverErr = state.Error
			counts = state.Counts
		}

		// Check if this server is being restarted.
		isRestarting := restartingCopy[name]

		displayText := formatServerStatus(t, name, serverState, counts, serverErr, isRestarting)
		items[i] = list.NewCompletionItem(
			displayText,
			ServerItem{Name: name, State: serverState, Counts: counts, Error: serverErr},
			list.WithCompletionID(name),
		)
	}

	return serversLoadedMsg{items: items}
}

func formatServerStatus(t *styles.Theme, name string, state mcp.State, counts mcp.Counts, err error, isRestarting bool) string {
	var icon lipgloss.Style
	var status string

	// Show restarting state if actively restarting.
	if isRestarting {
		icon = t.ItemBusyIcon
		status = t.S().Subtle.Render("restarting...")
		return fmt.Sprintf("%s %-*s %s", icon.String(), serverStatusWidth, name, status)
	}

	switch state {
	case mcp.StateDisabled:
		icon = t.ItemOfflineIcon
		status = t.S().Subtle.Render("disabled")
	case mcp.StateStarting:
		icon = t.ItemBusyIcon
		status = t.S().Subtle.Render("starting...")
	case mcp.StateConnected:
		icon = t.ItemOnlineIcon
		if counts.Tools > 0 {
			label := "tools"
			if counts.Tools == 1 {
				label = "tool"
			}
			status = t.S().Subtle.Render(fmt.Sprintf("%d %s", counts.Tools, label))
		}
	case mcp.StateError:
		icon = t.ItemErrorIcon
		if err != nil {
			errStr := err.Error()
			if len(errStr) > 30 {
				errStr = errStr[:30] + "…"
			}
			status = t.S().Subtle.Render(errStr)
		} else {
			status = t.S().Subtle.Render("error")
		}
	default:
		icon = t.ItemOfflineIcon
		status = t.S().Subtle.Render("unknown")
	}

	return fmt.Sprintf("%s %-*s %s", icon.String(), serverStatusWidth, name, status)
}

func (d *mcpServersDialogCmp) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		var cmds []tea.Cmd
		d.wWidth = msg.Width
		d.wHeight = msg.Height
		d.width = min(120, d.wWidth-8)
		d.serversList.SetInputWidth(d.listWidth() - 2)
		cmds = append(cmds, d.serversList.SetSize(d.listWidth(), d.listHeight()))
		return d, tea.Batch(cmds...)

	case serversLoadedMsg:
		d.mu.Lock()
		t := styles.CurrentTheme()
		listKeyMap := list.DefaultKeyMap()
		listKeyMap.Down.SetEnabled(false)
		listKeyMap.Up.SetEnabled(false)
		listKeyMap.DownOneItem = d.keyMap.Next
		listKeyMap.UpOneItem = d.keyMap.Previous

		inputStyle := t.S().Base.PaddingLeft(1).PaddingBottom(1)
		d.serversList = list.NewFilterableList(
			msg.items,
			list.WithFilterPlaceholder("Search MCP servers..."),
			list.WithFilterInputStyle(inputStyle),
			list.WithFilterListOptions(
				list.WithKeyMap(listKeyMap),
				list.WithWrapNavigation(),
			),
		)
		d.loading = false
		d.mu.Unlock()
		return d, tea.Batch(
			d.serversList.Init(),
			d.serversList.Focus(),
			d.serversList.SetSize(d.listWidth(), d.listHeight()),
		)

	case restartCompleteMsg:
		d.mu.Lock()
		delete(d.restarting, msg.name)
		d.mu.Unlock()
		return d, d.loadServersCmd()

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, d.keyMap.Select):
			selected := d.serversList.SelectedItem()
			if selected != nil {
				item := (*selected).Value()
				return d, util.CmdHandler(dialogs.OpenDialogMsg{
					Model: NewMCPServerDetailsDialogCmp(d.ctx, item.Name),
				})
			}
		case key.Matches(msg, d.keyMap.Restart):
			selected := d.serversList.SelectedItem()
			if selected != nil {
				item := (*selected).Value()
				d.mu.Lock()
				d.restarting[item.Name] = true
				d.mu.Unlock()
				return d, tea.Batch(
					d.loadServersCmd(),
					d.restartServerCmd(item.Name),
				)
			}
		case key.Matches(msg, d.keyMap.ViewLogs):
			selected := d.serversList.SelectedItem()
			if selected != nil {
				item := (*selected).Value()
				return d, util.CmdHandler(dialogs.OpenDialogMsg{
					Model: NewMCPLogsDialogCmp(item.Name),
				})
			}
		case key.Matches(msg, d.keyMap.Close):
			return d, util.CmdHandler(dialogs.CloseDialogMsg{})
		default:
			u, cmd := d.serversList.Update(msg)
			d.serversList = u.(list.FilterableList[list.CompletionItem[ServerItem]])
			return d, cmd
		}
	}

	return d, nil
}

type restartCompleteMsg struct {
	name string
	err  error
}

func (d *mcpServersDialogCmp) restartServerCmd(name string) tea.Cmd {
	return func() tea.Msg {
		err := mcp.RestartServer(d.ctx, name)
		return restartCompleteMsg{name: name, err: err}
	}
}

func (d *mcpServersDialogCmp) View() string {
	t := styles.CurrentTheme()

	var content string
	if d.serversList.Len() == 0 {
		content = d.renderEmptyState()
	} else {
		content = d.serversList.View()
	}

	return d.style().Render(lipgloss.JoinVertical(
		lipgloss.Left,
		t.S().Base.Padding(0, 1, 1, 1).Render(core.Title("MCP Servers", d.width-4)),
		content,
		"",
		t.S().Base.Width(d.width-2).PaddingLeft(1).AlignHorizontal(lipgloss.Left).Render(d.help.View(d.keyMap)),
	))
}

func (d *mcpServersDialogCmp) renderEmptyState() string {
	t := styles.CurrentTheme()
	style := t.S().Base.
		Width(d.listWidth()).
		Height(d.listHeight()).
		Padding(2).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Foreground(t.FgMuted)

	return style.Render("No MCP servers configured.\n\nAdd servers to crush.json")
}

func (d *mcpServersDialogCmp) Cursor() *tea.Cursor {
	if cursor, ok := d.serversList.(util.Cursor); ok {
		cursor := cursor.Cursor()
		if cursor != nil {
			cursor = d.moveCursor(cursor)
		}
		return cursor
	}
	return nil
}

func (d *mcpServersDialogCmp) style() lipgloss.Style {
	t := styles.CurrentTheme()
	return t.S().Base.
		Width(d.width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus)
}

func (d *mcpServersDialogCmp) listHeight() int {
	return d.wHeight/2 - 6
}

func (d *mcpServersDialogCmp) listWidth() int {
	return d.width - 2
}

func (d *mcpServersDialogCmp) Position() (int, int) {
	row := d.wHeight/4 - 2
	col := d.wWidth / 2
	col -= d.width / 2
	return row, col
}

func (d *mcpServersDialogCmp) moveCursor(cursor *tea.Cursor) *tea.Cursor {
	row, col := d.Position()
	offset := row + 3
	cursor.Y += offset
	cursor.X = cursor.X + col + 2
	return cursor
}

// ID implements DialogModel.
func (d *mcpServersDialogCmp) ID() dialogs.DialogID {
	return MCPServersDialogID
}
