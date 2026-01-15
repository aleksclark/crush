package mcpservers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

const MCPServerDetailsDialogID dialogs.DialogID = "mcp-server-details"

// TabType represents the different tabs in the server details view.
type TabType int

const (
	TabTools TabType = iota
	TabPrompts
	TabResources
)

func (t TabType) String() string {
	switch t {
	case TabTools:
		return "Tools"
	case TabPrompts:
		return "Prompts"
	case TabResources:
		return "Resources"
	default:
		return "Unknown"
	}
}

// ListItem represents an item in any of the detail lists.
type ListItem struct {
	Name        string
	Description string
	Schema      string
}

type mcpServerDetailsDialogCmp struct {
	wWidth     int
	wHeight    int
	width      int
	serverName string
	currentTab TabType
	keyMap     DetailKeyMap
	help       help.Model
	ctx        context.Context

	// List items for each tab.
	toolsList     list.FilterableList[list.CompletionItem[ListItem]]
	promptsList   list.FilterableList[list.CompletionItem[ListItem]]
	resourcesList list.FilterableList[list.CompletionItem[ListItem]]

	// Detail view state.
	showDetail   bool
	selectedItem *ListItem
}

// NewMCPServerDetailsDialogCmp creates a new server details dialog.
func NewMCPServerDetailsDialogCmp(ctx context.Context, serverName string) dialogs.DialogModel {
	t := styles.CurrentTheme()
	keyMap := DefaultDetailKeyMap()

	helpModel := help.New()
	helpModel.Styles = t.S().Help

	dialog := &mcpServerDetailsDialogCmp{
		ctx:        ctx,
		serverName: serverName,
		currentTab: TabTools,
		keyMap:     keyMap,
		help:       helpModel,
	}

	dialog.loadData()
	return dialog
}

func (d *mcpServerDetailsDialogCmp) loadData() {
	d.toolsList = d.createToolsList()
	d.promptsList = d.createPromptsList()
	d.resourcesList = d.createResourcesList()
}

func (d *mcpServerDetailsDialogCmp) createToolsList() list.FilterableList[list.CompletionItem[ListItem]] {
	t := styles.CurrentTheme()
	listKeyMap := list.DefaultKeyMap()
	listKeyMap.Down.SetEnabled(false)
	listKeyMap.Up.SetEnabled(false)
	listKeyMap.DownOneItem = d.keyMap.Next
	listKeyMap.UpOneItem = d.keyMap.Previous

	tools := mcp.GetTools(d.serverName)
	items := make([]list.CompletionItem[ListItem], len(tools))

	for i, tool := range tools {
		schema := ""
		if tool.InputSchema != nil {
			if schemaBytes, err := json.MarshalIndent(tool.InputSchema, "", "  "); err == nil {
				schema = string(schemaBytes)
			}
		}

		item := ListItem{
			Name:        tool.Name,
			Description: tool.Description,
			Schema:      schema,
		}
		displayText := tool.Name
		if tool.Description != "" {
			displayText = fmt.Sprintf("%s - %s", tool.Name, truncateStr(tool.Description, 40))
		}
		items[i] = list.NewCompletionItem(displayText, item, list.WithCompletionID(tool.Name))
	}

	inputStyle := t.S().Base.PaddingLeft(1).PaddingBottom(1)
	return list.NewFilterableList(
		items,
		list.WithFilterPlaceholder("Search tools..."),
		list.WithFilterInputStyle(inputStyle),
		list.WithFilterListOptions(
			list.WithKeyMap(listKeyMap),
			list.WithWrapNavigation(),
		),
	)
}

func (d *mcpServerDetailsDialogCmp) createPromptsList() list.FilterableList[list.CompletionItem[ListItem]] {
	t := styles.CurrentTheme()
	listKeyMap := list.DefaultKeyMap()
	listKeyMap.Down.SetEnabled(false)
	listKeyMap.Up.SetEnabled(false)
	listKeyMap.DownOneItem = d.keyMap.Next
	listKeyMap.UpOneItem = d.keyMap.Previous

	prompts := mcp.GetPrompts(d.serverName)
	items := make([]list.CompletionItem[ListItem], len(prompts))

	for i, prompt := range prompts {
		item := ListItem{
			Name:        prompt.Name,
			Description: prompt.Description,
		}
		displayText := prompt.Name
		if prompt.Description != "" {
			displayText = fmt.Sprintf("%s - %s", prompt.Name, truncateStr(prompt.Description, 40))
		}
		items[i] = list.NewCompletionItem(displayText, item, list.WithCompletionID(prompt.Name))
	}

	inputStyle := t.S().Base.PaddingLeft(1).PaddingBottom(1)
	return list.NewFilterableList(
		items,
		list.WithFilterPlaceholder("Search prompts..."),
		list.WithFilterInputStyle(inputStyle),
		list.WithFilterListOptions(
			list.WithKeyMap(listKeyMap),
			list.WithWrapNavigation(),
		),
	)
}

func (d *mcpServerDetailsDialogCmp) createResourcesList() list.FilterableList[list.CompletionItem[ListItem]] {
	t := styles.CurrentTheme()
	listKeyMap := list.DefaultKeyMap()
	listKeyMap.Down.SetEnabled(false)
	listKeyMap.Up.SetEnabled(false)
	listKeyMap.DownOneItem = d.keyMap.Next
	listKeyMap.UpOneItem = d.keyMap.Previous

	resources := mcp.GetResources(d.serverName)
	items := make([]list.CompletionItem[ListItem], len(resources))

	for i, resource := range resources {
		item := ListItem{
			Name:        resource.Name,
			Description: resource.Description,
		}
		displayText := resource.Name
		if resource.Description != "" {
			displayText = fmt.Sprintf("%s - %s", resource.Name, truncateStr(resource.Description, 40))
		}
		items[i] = list.NewCompletionItem(displayText, item, list.WithCompletionID(resource.Name))
	}

	inputStyle := t.S().Base.PaddingLeft(1).PaddingBottom(1)
	return list.NewFilterableList(
		items,
		list.WithFilterPlaceholder("Search resources..."),
		list.WithFilterInputStyle(inputStyle),
		list.WithFilterListOptions(
			list.WithKeyMap(listKeyMap),
			list.WithWrapNavigation(),
		),
	)
}

func (d *mcpServerDetailsDialogCmp) currentList() list.FilterableList[list.CompletionItem[ListItem]] {
	switch d.currentTab {
	case TabTools:
		return d.toolsList
	case TabPrompts:
		return d.promptsList
	case TabResources:
		return d.resourcesList
	default:
		return d.toolsList
	}
}

func (d *mcpServerDetailsDialogCmp) Init() tea.Cmd {
	return tea.Batch(
		d.toolsList.Init(),
		d.toolsList.Focus(),
	)
}

func (d *mcpServerDetailsDialogCmp) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		var cmds []tea.Cmd
		d.wWidth = msg.Width
		d.wHeight = msg.Height
		d.width = min(120, d.wWidth-8)

		cmds = append(cmds, d.toolsList.SetSize(d.listWidth(), d.listHeight()))
		cmds = append(cmds, d.promptsList.SetSize(d.listWidth(), d.listHeight()))
		cmds = append(cmds, d.resourcesList.SetSize(d.listWidth(), d.listHeight()))
		return d, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		if d.showDetail {
			// Any key exits detail view.
			d.showDetail = false
			d.selectedItem = nil
			return d, nil
		}

		switch {
		case key.Matches(msg, d.keyMap.Tab):
			d.currentTab = (d.currentTab + 1) % 3
			return d, d.currentList().Focus()

		case key.Matches(msg, d.keyMap.Select):
			currentList := d.currentList()
			selected := currentList.SelectedItem()
			if selected != nil {
				item := (*selected).Value()
				d.selectedItem = &item
				d.showDetail = true
			}
			return d, nil

		case key.Matches(msg, d.keyMap.Close):
			return d, util.CmdHandler(dialogs.CloseDialogMsg{})

		default:
			currentList := d.currentList()
			u, cmd := currentList.Update(msg)
			switch d.currentTab {
			case TabTools:
				d.toolsList = u.(list.FilterableList[list.CompletionItem[ListItem]])
			case TabPrompts:
				d.promptsList = u.(list.FilterableList[list.CompletionItem[ListItem]])
			case TabResources:
				d.resourcesList = u.(list.FilterableList[list.CompletionItem[ListItem]])
			}
			return d, cmd
		}
	}

	return d, nil
}

func (d *mcpServerDetailsDialogCmp) View() string {
	t := styles.CurrentTheme()

	var content string
	if d.showDetail && d.selectedItem != nil {
		content = d.renderDetailView()
	} else {
		content = d.renderListView()
	}

	title := fmt.Sprintf("MCP: %s", d.serverName)
	return d.style().Render(lipgloss.JoinVertical(
		lipgloss.Left,
		t.S().Base.Padding(0, 1, 1, 1).Render(core.Title(title, d.width-4)),
		d.renderTabs(),
		content,
		"",
		t.S().Base.Width(d.width-2).PaddingLeft(1).AlignHorizontal(lipgloss.Left).Render(d.help.View(d.keyMap)),
	))
}

func (d *mcpServerDetailsDialogCmp) renderTabs() string {
	t := styles.CurrentTheme()
	tabs := []string{"Tools", "Prompts", "Resources"}
	var rendered []string

	for i, tab := range tabs {
		style := t.S().Base.Padding(0, 2)
		if TabType(i) == d.currentTab {
			style = style.Bold(true).Underline(true)
		} else {
			style = style.Foreground(t.FgMuted)
		}
		rendered = append(rendered, style.Render(tab))
	}

	return t.S().Base.PaddingLeft(1).PaddingBottom(1).Render(
		lipgloss.JoinHorizontal(lipgloss.Top, rendered...),
	)
}

func (d *mcpServerDetailsDialogCmp) renderListView() string {
	currentList := d.currentList()
	if currentList.Len() == 0 {
		return d.renderEmptyTabState()
	}
	return currentList.View()
}

func (d *mcpServerDetailsDialogCmp) renderEmptyTabState() string {
	t := styles.CurrentTheme()
	style := t.S().Base.
		Width(d.listWidth()).
		Height(d.listHeight()).
		Padding(2).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Foreground(t.FgMuted)

	var msg string
	switch d.currentTab {
	case TabTools:
		msg = "No tools available"
	case TabPrompts:
		msg = "No prompts available"
	case TabResources:
		msg = "No resources available"
	}
	return style.Render(msg)
}

func (d *mcpServerDetailsDialogCmp) renderDetailView() string {
	t := styles.CurrentTheme()
	item := d.selectedItem

	var lines []string
	lines = append(lines, fmt.Sprintf("Name: %s", item.Name))
	if item.Description != "" {
		lines = append(lines, "")
		lines = append(lines, "Description:")
		lines = append(lines, item.Description)
	}
	if item.Schema != "" {
		lines = append(lines, "")
		lines = append(lines, "Schema:")
		lines = append(lines, item.Schema)
	}

	style := t.S().Base.
		Width(d.listWidth()).
		Height(d.listHeight()).
		Padding(1)

	return style.Render(strings.Join(lines, "\n"))
}

func (d *mcpServerDetailsDialogCmp) Cursor() *tea.Cursor {
	if d.showDetail {
		return nil
	}
	currentList := d.currentList()
	if cursor, ok := any(currentList).(util.Cursor); ok {
		cursor := cursor.Cursor()
		if cursor != nil {
			cursor = d.moveCursor(cursor)
		}
		return cursor
	}
	return nil
}

func (d *mcpServerDetailsDialogCmp) style() lipgloss.Style {
	t := styles.CurrentTheme()
	return t.S().Base.
		Width(d.width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus)
}

func (d *mcpServerDetailsDialogCmp) listHeight() int {
	return d.wHeight/2 - 8 // Extra space for tabs.
}

func (d *mcpServerDetailsDialogCmp) listWidth() int {
	return d.width - 2
}

func (d *mcpServerDetailsDialogCmp) Position() (int, int) {
	row := d.wHeight/4 - 2
	col := d.wWidth / 2
	col -= d.width / 2
	return row, col
}

func (d *mcpServerDetailsDialogCmp) moveCursor(cursor *tea.Cursor) *tea.Cursor {
	row, col := d.Position()
	offset := row + 5 // Border + title + tabs.
	cursor.Y += offset
	cursor.X = cursor.X + col + 2
	return cursor
}

// ID implements DialogModel.
func (d *mcpServerDetailsDialogCmp) ID() dialogs.DialogID {
	return MCPServerDetailsDialogID
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
