package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aleksclark/trifle"
)

const (
	startupDelay     = 3 * time.Second
	dialogTransition = 500 * time.Millisecond
)

// TestWorktreeDialogOpens tests that the worktree dialog opens with ctrl+n.
func TestWorktreeDialogOpens(t *testing.T) {
	trifle.SkipOnWindows(t)

	term, err := trifle.NewTerminal(CrushBinary(), []string{}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	// Wait for TUI to initialize.
	time.Sleep(startupDelay)

	// Press ctrl+n to open new session dialog.
	if err := term.Write(trifle.CtrlN()); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// Should show the worktree dialog.
	locator := term.GetByText("New Worktree Session", trifle.WithFull())
	if err := locator.WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected worktree dialog: %v", err)
	}
}

// TestWorktreeDialogDescription tests the dialog description.
func TestWorktreeDialogDescription(t *testing.T) {
	trifle.SkipOnWindows(t)

	term, err := trifle.NewTerminal(CrushBinary(), []string{}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlN())

	locator := term.GetByText("Enter a name for the git worktree branch", trifle.WithFull())
	if err := locator.WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected dialog description: %v", err)
	}
}

// TestWorktreeDialogLabel tests the session name label.
func TestWorktreeDialogLabel(t *testing.T) {
	trifle.SkipOnWindows(t)

	term, err := trifle.NewTerminal(CrushBinary(), []string{}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlN())

	locator := term.GetByText("Session name:", trifle.WithFull())
	if err := locator.WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected session name label: %v", err)
	}
}

// TestWorktreeDialogPlaceholder tests the placeholder text.
func TestWorktreeDialogPlaceholder(t *testing.T) {
	trifle.SkipOnWindows(t)

	term, err := trifle.NewTerminal(CrushBinary(), []string{}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlN())

	locator := term.GetByText("feature-name", trifle.WithFull())
	if err := locator.WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected placeholder: %v", err)
	}
}

// TestWorktreeDialogKeybindings tests that keybinding hints are shown.
func TestWorktreeDialogKeybindings(t *testing.T) {
	trifle.SkipOnWindows(t)

	term, err := trifle.NewTerminal(CrushBinary(), []string{}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlN())

	if err := term.GetByText("create worktree", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected create keybinding: %v", err)
	}
	if err := term.GetByText("cancel", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected cancel keybinding: %v", err)
	}
}

// TestWorktreeDialogClose tests that escape closes the dialog.
func TestWorktreeDialogClose(t *testing.T) {
	trifle.SkipOnWindows(t)

	term, err := trifle.NewTerminal(CrushBinary(), []string{}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlN())

	// Verify dialog is open.
	if err := term.GetByText("New Worktree Session", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Fatalf("Dialog did not open: %v", err)
	}

	// Press escape to close.
	_ = term.KeyEscape()
	time.Sleep(dialogTransition)

	// Check that project init dialog is back.
	locator := term.GetByText("initialize this project", trifle.WithFull())
	if err := locator.WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected to return to init dialog: %v", err)
	}
}

// TestWorktreeDialogEmptyError tests error for empty name.
func TestWorktreeDialogEmptyError(t *testing.T) {
	trifle.SkipOnWindows(t)

	term, err := trifle.NewTerminal(CrushBinary(), []string{}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlN())
	time.Sleep(300 * time.Millisecond)

	// Press enter without typing a name.
	_ = term.Submit()
	time.Sleep(300 * time.Millisecond)

	// Should show error message.
	locator := term.GetByText("Name is required", trifle.WithFull())
	if err := locator.WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected error message: %v", err)
	}
}

// TestWorktreeDialogInput tests text input acceptance.
func TestWorktreeDialogInput(t *testing.T) {
	trifle.SkipOnWindows(t)

	term, err := trifle.NewTerminal(CrushBinary(), []string{}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlN())
	time.Sleep(300 * time.Millisecond)

	// Type a branch name.
	_ = term.Write("my-test-feature")
	time.Sleep(200 * time.Millisecond)

	// Should show typed text.
	locator := term.GetByText("my-test-feature", trifle.WithFull())
	if err := locator.WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected typed text: %v", err)
	}
}

// TestWorktreeCreationInGitRepo tests worktree creation in a temp git repo.
func TestWorktreeCreationInGitRepo(t *testing.T) {
	trifle.SkipOnWindows(t)

	// Bash script that creates a temp git repo and runs crush.
	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"
git init -q
git config user.email "test@test.com"
git config user.name "Test User"
echo "# Test Project" > README.md
git add README.md
git commit -q -m "Initial commit"
cat > crush.json << 'CRUSHCONFIG'
{
  "options": {
    "worktree_mode": true
  }
}
CRUSHCONFIG
exec "%s"
`, CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)

	// Open worktree dialog.
	_ = term.Write(trifle.CtrlN())
	time.Sleep(dialogTransition)

	// Verify dialog opened.
	if err := term.GetByText("New Worktree Session", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Fatalf("Dialog did not open: %v", err)
	}

	// Type a valid branch name.
	branchName := fmt.Sprintf("test-feature-%d", time.Now().UnixNano())
	_ = term.Write(branchName)
	time.Sleep(200 * time.Millisecond)

	// Submit the form.
	_ = term.Submit()
	time.Sleep(2 * time.Second)

	// Should show success message.
	locator := term.GetByText("Worktree created", trifle.WithFull())
	if err := locator.WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected success message: %v", err)
	}
}

// TestWorktreeSanitization tests that special characters are sanitized.
func TestWorktreeSanitization(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"
git init -q
git config user.email "test@test.com"
git config user.name "Test User"
echo "# Test Project" > README.md
git add README.md
git commit -q -m "Initial commit"
cat > crush.json << 'CRUSHCONFIG'
{
  "options": {
    "worktree_mode": true
  }
}
CRUSHCONFIG
exec "%s"
`, CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlN())
	time.Sleep(dialogTransition)

	// Type a name with spaces (will be sanitized to hyphens).
	_ = term.Write("my new feature")
	time.Sleep(200 * time.Millisecond)

	_ = term.Submit()
	time.Sleep(2 * time.Second)

	// Should succeed (name gets sanitized).
	locator := term.GetByText("Worktree created", trifle.WithFull())
	if err := locator.WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected success after sanitization: %v", err)
	}
}

// TestWorktreeNotGitRepo tests error when not in git repo.
func TestWorktreeNotGitRepo(t *testing.T) {
	trifle.SkipOnWindows(t)

	term, err := trifle.NewTerminal(CrushBinary(), []string{}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlN())
	time.Sleep(300 * time.Millisecond)

	_ = term.Write("test-feature")
	time.Sleep(200 * time.Millisecond)

	_ = term.Submit()
	time.Sleep(1 * time.Second)

	// Should show error about not being a git repository.
	output := term.Output()
	if !strings.Contains(output, "not a git repository") {
		t.Errorf("Expected git repo error, got: %s", output)
	}
}
