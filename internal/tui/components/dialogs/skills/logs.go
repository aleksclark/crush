package skills

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/charmbracelet/crush/internal/skills"
	"github.com/charmbracelet/crush/internal/tui/components/core"
	"github.com/charmbracelet/crush/internal/tui/components/dialogs"
	"github.com/charmbracelet/crush/internal/tui/exp/list"
	"github.com/charmbracelet/crush/internal/tui/styles"
	"github.com/charmbracelet/crush/internal/tui/util"
)

const SkillLogsDialogID dialogs.DialogID = "skill-logs"

// LogListItem represents a log entry in the list.
type LogListItem struct {
	Entry skills.LogEntry
}

type skillLogsDialogCmp struct {
	wWidth     int
	wHeight    int
	width      int
	skillName  string
	keyMap     LogsKeyMap
	help       help.Model
	logsList   list.FilterableList[list.CompletionItem[LogListItem]]
	showDetail bool
	selected   *skills.LogEntry
}

// NewSkillLogsDialogCmp creates a new logs dialog for a specific skill.
func NewSkillLogsDialogCmp(skillName string) dialogs.DialogModel {
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

	dialog := &skillLogsDialogCmp{
		skillName: skillName,
		keyMap:    keyMap,
		help:      helpModel,
		logsList:  logsList,
	}

	return dialog
}

func (d *skillLogsDialogCmp) Init() tea.Cmd {
	return tea.Batch(
		d.logsList.Init(),
		d.logsList.Focus(),
		d.loadLogsCmd(),
	)
}

type skillLogsLoadedMsg struct {
	entries []skills.LogEntry
}

func (d *skillLogsDialogCmp) loadLogsCmd() tea.Cmd {
	return func() tea.Msg {
		entries, _ := skills.GetLogEntries(d.skillName, "", 100)
		return skillLogsLoadedMsg{entries: entries}
	}
}

func (d *skillLogsDialogCmp) createLogsList(entries []skills.LogEntry) list.FilterableList[list.CompletionItem[LogListItem]] {
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
		displayText := fmt.Sprintf("%s %s %s %s",
			entry.Timestamp.Format("15:04:05"),
			status,
			truncate(entry.SkillName, 20),
			entry.Action,
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

func (d *skillLogsDialogCmp) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.wWidth = msg.Width
		d.wHeight = msg.Height
		d.width = min(140, d.wWidth-8)
		return d, d.logsList.SetSize(d.listWidth(), d.listHeight())

	case skillLogsLoadedMsg:
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
			d.selected = nil
			return d, nil
		}

		switch {
		case key.Matches(msg, d.keyMap.Close):
			return d, util.CmdHandler(dialogs.CloseDialogMsg{})

		default:
			selected := d.logsList.SelectedItem()
			if selected != nil && key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				item := (*selected).Value()
				d.selected = &item.Entry
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

func (d *skillLogsDialogCmp) View() string {
	t := styles.CurrentTheme()

	var content string
	if d.showDetail && d.selected != nil {
		content = d.renderDetailView()
	} else {
		content = d.renderListView()
	}

	title := fmt.Sprintf("Logs: %s", d.skillName)

	return d.style().Render(lipgloss.JoinVertical(
		lipgloss.Left,
		t.S().Base.Padding(0, 1, 1, 1).Render(core.Title(title, d.width-4)),
		content,
		"",
		t.S().Base.Width(d.width-2).PaddingLeft(1).AlignHorizontal(lipgloss.Left).Render(d.help.View(d.keyMap)),
	))
}

func (d *skillLogsDialogCmp) renderListView() string {
	if d.logsList.Len() == 0 {
		return d.renderEmptyState()
	}
	return d.logsList.View()
}

func (d *skillLogsDialogCmp) renderEmptyState() string {
	t := styles.CurrentTheme()
	style := t.S().Base.
		Width(d.listWidth()).
		Height(d.listHeight()).
		Padding(2).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Foreground(t.FgMuted)

	return style.Render("No logs available.\n\nSkill usage will be logged when the agent reads skills.")
}

func (d *skillLogsDialogCmp) renderDetailView() string {
	t := styles.CurrentTheme()
	entry := d.selected

	var lines []string
	lines = append(lines, fmt.Sprintf("Timestamp: %s", entry.Timestamp.Format("2006-01-02 15:04:05")))
	lines = append(lines, fmt.Sprintf("Session:   %s", entry.SessionID))
	lines = append(lines, fmt.Sprintf("Skill:     %s", entry.SkillName))
	lines = append(lines, fmt.Sprintf("Action:    %s", entry.Action))
	lines = append(lines, fmt.Sprintf("Success:   %v", entry.Success))

	if entry.FilePath != "" {
		lines = append(lines, fmt.Sprintf("File:      %s", entry.FilePath))
	}

	if entry.DurationMs > 0 {
		lines = append(lines, fmt.Sprintf("Duration:  %dms", entry.DurationMs))
	}

	if entry.Error != "" {
		lines = append(lines, "")
		lines = append(lines, "Error:")
		lines = append(lines, entry.Error)
	}

	style := t.S().Base.
		Width(d.listWidth()).
		Height(d.listHeight()).
		Padding(1)

	return style.Render(strings.Join(lines, "\n"))
}

func (d *skillLogsDialogCmp) Cursor() *tea.Cursor {
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

func (d *skillLogsDialogCmp) style() lipgloss.Style {
	t := styles.CurrentTheme()
	return t.S().Base.
		Width(d.width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus)
}

func (d *skillLogsDialogCmp) listHeight() int {
	return d.wHeight/2 - 6
}

func (d *skillLogsDialogCmp) listWidth() int {
	return d.width - 2
}

func (d *skillLogsDialogCmp) Position() (int, int) {
	row := d.wHeight/4 - 2
	col := d.wWidth / 2
	col -= d.width / 2
	return row, col
}

func (d *skillLogsDialogCmp) moveCursor(cursor *tea.Cursor) *tea.Cursor {
	row, col := d.Position()
	offset := row + 3
	cursor.Y += offset
	cursor.X = cursor.X + col + 2
	return cursor
}

// ID implements DialogModel.
func (d *skillLogsDialogCmp) ID() dialogs.DialogID {
	return SkillLogsDialogID
}
