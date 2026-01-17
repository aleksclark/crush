package skills

import (
	"fmt"
	"os"
	"path/filepath"
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

// SkillsDialogID is the unique identifier for the skills dialog.
const SkillsDialogID dialogs.DialogID = "skills"

// SkillsDialog interface for the skills list dialog.
type SkillsDialog interface {
	dialogs.DialogModel
}

// SkillsList is the filterable list of skills.
type SkillsList = list.FilterableList[list.CompletionItem[*skills.Skill]]

type skillsDialogCmp struct {
	wWidth     int
	wHeight    int
	width      int
	keyMap     KeyMap
	skillsList SkillsList
	help       help.Model
	skills     []*skills.Skill
	selected   *skills.Skill // Currently selected skill for detail view.
	showDetail bool          // Whether we're showing detail view.
}

// NewSkillsDialogCmp creates a new skills listing dialog.
func NewSkillsDialogCmp(skillsList []*skills.Skill) SkillsDialog {
	t := styles.CurrentTheme()
	listKeyMap := list.DefaultKeyMap()
	keyMap := DefaultKeyMap()
	listKeyMap.Down.SetEnabled(false)
	listKeyMap.Up.SetEnabled(false)
	listKeyMap.DownOneItem = keyMap.Next
	listKeyMap.UpOneItem = keyMap.Previous

	items := make([]list.CompletionItem[*skills.Skill], len(skillsList))
	for i, skill := range skillsList {
		// Include description in the display text.
		displayText := skill.Name
		if skill.Description != "" {
			displayText = fmt.Sprintf("%s - %s", skill.Name, truncate(skill.Description, 50))
		}
		items[i] = list.NewCompletionItem(
			displayText,
			skill,
			list.WithCompletionID(skill.Name),
		)
	}

	inputStyle := t.S().Base.PaddingLeft(1).PaddingBottom(1)
	listModel := list.NewFilterableList(
		items,
		list.WithFilterPlaceholder("Search skills..."),
		list.WithFilterInputStyle(inputStyle),
		list.WithFilterListOptions(
			list.WithKeyMap(listKeyMap),
			list.WithWrapNavigation(),
		),
	)

	helpModel := help.New()
	helpModel.Styles = t.S().Help

	return &skillsDialogCmp{
		keyMap:     keyMap,
		skillsList: listModel,
		help:       helpModel,
		skills:     skillsList,
	}
}

func (s *skillsDialogCmp) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, s.skillsList.Init())
	cmds = append(cmds, s.skillsList.Focus())
	return tea.Sequence(cmds...)
}

func (s *skillsDialogCmp) Update(msg tea.Msg) (util.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		var cmds []tea.Cmd
		s.wWidth = msg.Width
		s.wHeight = msg.Height
		s.width = min(120, s.wWidth-8)
		s.skillsList.SetInputWidth(s.listWidth() - 2)
		cmds = append(cmds, s.skillsList.SetSize(s.listWidth(), s.listHeight()))
		return s, tea.Batch(cmds...)
	case tea.KeyPressMsg:
		if s.showDetail {
			// In detail view, any key goes back to list.
			s.showDetail = false
			s.selected = nil
			return s, nil
		}
		switch {
		case key.Matches(msg, s.keyMap.Select):
			selectedItem := s.skillsList.SelectedItem()
			if selectedItem != nil {
				s.selected = (*selectedItem).Value()
				s.showDetail = true
			}
			return s, nil
		case key.Matches(msg, s.keyMap.ViewLogs):
			selectedItem := s.skillsList.SelectedItem()
			if selectedItem != nil {
				skill := (*selectedItem).Value()
				return s, util.CmdHandler(dialogs.OpenDialogMsg{
					Model: NewSkillLogsDialogCmp(skill.Name),
				})
			}
			return s, nil
		case key.Matches(msg, s.keyMap.Close):
			return s, util.CmdHandler(dialogs.CloseDialogMsg{})
		default:
			u, cmd := s.skillsList.Update(msg)
			s.skillsList = u.(SkillsList)
			return s, cmd
		}
	}
	return s, nil
}

func (s *skillsDialogCmp) View() string {
	t := styles.CurrentTheme()

	var content string
	if s.showDetail && s.selected != nil {
		content = s.renderDetailView()
	} else {
		content = s.renderListView()
	}

	return s.style().Render(lipgloss.JoinVertical(
		lipgloss.Left,
		t.S().Base.Padding(0, 1, 1, 1).Render(core.Title(s.title(), s.width-4)),
		content,
		"",
		t.S().Base.Width(s.width-2).PaddingLeft(1).AlignHorizontal(lipgloss.Left).Render(s.help.View(s.keyMap)),
	))
}

func (s *skillsDialogCmp) title() string {
	if s.showDetail && s.selected != nil {
		return fmt.Sprintf("Skill: %s", s.selected.Name)
	}
	return fmt.Sprintf("Skills (%d)", len(s.skills))
}

func (s *skillsDialogCmp) renderListView() string {
	if len(s.skills) == 0 {
		return s.renderEmptyState()
	}
	return s.skillsList.View()
}

func (s *skillsDialogCmp) renderEmptyState() string {
	t := styles.CurrentTheme()
	style := t.S().Base.
		Width(s.listWidth()).
		Height(s.listHeight()).
		Padding(2).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Foreground(t.FgMuted)

	return style.Render("No skills found.\n\nCreate skills in ~/.config/crush/skills/")
}

func (s *skillsDialogCmp) renderDetailView() string {
	t := styles.CurrentTheme()
	skill := s.selected

	var lines []string

	// Description.
	if skill.Description != "" {
		lines = append(lines, skill.Description)
		lines = append(lines, "")
	}

	// License.
	if skill.License != "" {
		lines = append(lines, fmt.Sprintf("License: %s", skill.License))
	}

	// Compatibility.
	if skill.Compatibility != "" {
		lines = append(lines, fmt.Sprintf("Compatibility: %s", skill.Compatibility))
	}

	// Metadata.
	if len(skill.Metadata) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Metadata:")
		for k, v := range skill.Metadata {
			lines = append(lines, fmt.Sprintf("  %s: %s", k, v))
		}
	}

	// File path.
	if skill.SkillFilePath != "" {
		shortPath := filepath.Base(filepath.Dir(skill.SkillFilePath)) + "/" + filepath.Base(skill.SkillFilePath)
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Location: %s", shortPath))
	}

	// List files in skill directory.
	if skill.Path != "" {
		files := s.listSkillFiles(skill.Path)
		if len(files) > 0 {
			lines = append(lines, "")
			lines = append(lines, "Files:")
			for _, f := range files {
				lines = append(lines, fmt.Sprintf("  %s", f))
			}
		}
	}

	// Instructions preview.
	if skill.Instructions != "" {
		lines = append(lines, "")
		instructionsPreview := skill.Instructions
		if len(instructionsPreview) > 300 {
			instructionsPreview = instructionsPreview[:300] + "..."
		}
		// Remove newlines for preview.
		instructionsPreview = strings.ReplaceAll(instructionsPreview, "\n", " ")
		lines = append(lines, fmt.Sprintf("Instructions: %s", instructionsPreview))
	}

	style := t.S().Base.
		Width(s.listWidth()).
		Height(s.listHeight()).
		Padding(1)

	return style.Render(strings.Join(lines, "\n"))
}

// listSkillFiles returns a list of files in the skill directory.
func (s *skillsDialogCmp) listSkillFiles(path string) []string {
	var files []string
	entries, err := os.ReadDir(path)
	if err != nil {
		return files
	}
	for _, entry := range entries {
		if entry.IsDir() {
			// Show directory with trailing slash.
			files = append(files, entry.Name()+"/")
		} else {
			files = append(files, entry.Name())
		}
		// Limit to 10 files.
		if len(files) >= 10 {
			files = append(files, "...")
			break
		}
	}
	return files
}

func (s *skillsDialogCmp) Cursor() *tea.Cursor {
	if s.showDetail {
		return nil
	}
	if cursor, ok := s.skillsList.(util.Cursor); ok {
		cursor := cursor.Cursor()
		if cursor != nil {
			cursor = s.moveCursor(cursor)
		}
		return cursor
	}
	return nil
}

func (s *skillsDialogCmp) style() lipgloss.Style {
	t := styles.CurrentTheme()
	return t.S().Base.
		Width(s.width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus)
}

func (s *skillsDialogCmp) listHeight() int {
	return s.wHeight/2 - 6
}

func (s *skillsDialogCmp) listWidth() int {
	return s.width - 2
}

func (s *skillsDialogCmp) Position() (int, int) {
	row := s.wHeight/4 - 2
	col := s.wWidth / 2
	col -= s.width / 2
	return row, col
}

func (s *skillsDialogCmp) moveCursor(cursor *tea.Cursor) *tea.Cursor {
	row, col := s.Position()
	offset := row + 3
	cursor.Y += offset
	cursor.X = cursor.X + col + 2
	return cursor
}

// ID implements SkillsDialog.
func (s *skillsDialogCmp) ID() dialogs.DialogID {
	return SkillsDialogID
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	if maxLen <= 3 {
		return str[:maxLen]
	}
	return str[:maxLen-3] + "..."
}
