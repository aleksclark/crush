package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/aleksclark/trifle"
)

// TestSkillsDialogOpens tests that the skills dialog opens with ctrl+k.
func TestSkillsDialogOpens(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

cat > crush.json << 'CONFIG'
%s
CONFIG

exec "%s"
`, IsolationScript(), TestConfigJSON(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)

	// Press ctrl+k to open skills dialog.
	_ = term.Write(trifle.CtrlK())
	time.Sleep(700 * time.Millisecond)

	// Should show the skills dialog title.
	locator := term.GetByText("Skills", trifle.WithFull())
	if err := locator.WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected skills dialog: %v", err)
	}
}

// TestSkillsDialogClose tests that escape closes the dialog.
func TestSkillsDialogClose(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

cat > crush.json << 'CONFIG'
%s
CONFIG

exec "%s"
`, IsolationScript(), TestConfigJSON(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlK())
	time.Sleep(700 * time.Millisecond)

	if err := term.GetByText("Skills", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Fatalf("Dialog did not open: %v", err)
	}

	// Press escape to close.
	_ = term.Write(trifle.Escape())
	time.Sleep(700 * time.Millisecond)
	// Dialog should be closed now.
}

// TestSkillsDialogHelp tests that help keybindings are shown.
func TestSkillsDialogHelp(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

cat > crush.json << 'CONFIG'
%s
CONFIG

exec "%s"
`, IsolationScript(), TestConfigJSON(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlK())
	time.Sleep(700 * time.Millisecond)

	// Should show help text with keybindings.
	if err := term.GetByText("logs", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected logs keybinding: %v", err)
	}
}

// TestSkillsWithConfig tests the dialog with actual skill configurations.
func TestSkillsWithConfig(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

# Create skills directory
mkdir -p skills/test-skill

# Create SKILL.md file
cat > skills/test-skill/SKILL.md << 'SKILL'
---
name: test-skill
description: A test skill for e2e testing
license: MIT
---

# Test Skill

This is a test skill.
SKILL

# Create a helper file
cat > skills/test-skill/helper.sh << 'HELPER'
#!/bin/bash
echo "Helper script"
HELPER

# Create crush.json with skills_paths
cat > crush.json << 'CONFIG'
{
  "providers": {
    "test": {
      "type": "openai-compat",
      "base_url": "http://localhost:9999",
      "api_key": "test-key"
    }
  },
  "models": {
    "large": { "provider": "test", "model": "test-model" },
    "small": { "provider": "test", "model": "test-model" }
  },
  "options": {
    "skills_paths": ["./skills"]
  }
}
CONFIG

exec "%s"
`, IsolationScript(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlK())
	time.Sleep(700 * time.Millisecond)

	// Should show configured skill.
	if err := term.GetByText("test-skill", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected test-skill: %v", err)
	}
}

// TestSkillsDetailView tests skill detail view on enter.
func TestSkillsDetailView(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

mkdir -p skills/test-skill
cat > skills/test-skill/SKILL.md << 'SKILL'
---
name: test-skill
description: A test skill
---
# Test Skill
SKILL

cat > crush.json << 'CONFIG'
{
  "providers": {
    "test": {
      "type": "openai-compat",
      "base_url": "http://localhost:9999",
      "api_key": "test-key"
    }
  },
  "models": {
    "large": { "provider": "test", "model": "test-model" },
    "small": { "provider": "test", "model": "test-model" }
  },
  "options": {
    "skills_paths": ["./skills"]
  }
}
CONFIG

exec "%s"
`, IsolationScript(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlK())
	time.Sleep(700 * time.Millisecond)

	// Press enter to view details.
	_ = term.Submit()
	time.Sleep(700 * time.Millisecond)

	// Should show Location in detail view.
	if err := term.GetByText("Location:", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected location in detail view: %v", err)
	}
}

// TestSkillsEmptyState tests the dialog when no skills are configured.
func TestSkillsEmptyState(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

cat > crush.json << 'CONFIG'
{
  "providers": {
    "test": {
      "type": "openai-compat",
      "base_url": "http://localhost:9999",
      "api_key": "test-key"
    }
  },
  "models": {
    "large": { "provider": "test", "model": "test-model" },
    "small": { "provider": "test", "model": "test-model" }
  },
  "options": {
    "skills_paths": []
  }
}
CONFIG

exec "%s"
`, IsolationScript(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlK())
	time.Sleep(700 * time.Millisecond)

	// First verify the dialog opened.
	if err := term.GetByText("Skills", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Fatalf("Dialog did not open: %v", err)
	}

	// Should show empty state message.
	if err := term.GetByText("No skills found", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected empty state message: %v", err)
	}
}

// TestSkillsLogsDialog tests the logs dialog opened with 'l' key.
func TestSkillsLogsDialog(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

mkdir -p skills/test-skill
cat > skills/test-skill/SKILL.md << 'SKILL'
---
name: test-skill
description: A test skill
---
# Test Skill
SKILL

cat > crush.json << 'CONFIG'
{
  "providers": {
    "test": {
      "type": "openai-compat",
      "base_url": "http://localhost:9999",
      "api_key": "test-key"
    }
  },
  "models": {
    "large": { "provider": "test", "model": "test-model" },
    "small": { "provider": "test", "model": "test-model" }
  },
  "options": {
    "skills_paths": ["./skills"]
  }
}
CONFIG

exec "%s"
`, IsolationScript(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)

	// Open skills dialog.
	_ = term.Write(trifle.CtrlK())
	time.Sleep(700 * time.Millisecond)

	// Press 'l' to open logs.
	_ = term.Write("l")
	time.Sleep(700 * time.Millisecond)

	// Should show the logs dialog title.
	if err := term.GetByText("Logs:", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected logs dialog: %v", err)
	}
}
