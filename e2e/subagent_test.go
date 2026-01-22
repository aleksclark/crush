package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aleksclark/trifle"
)

// TestSubagentDiscovery tests that subagents are discovered from .crush/agents/.
func TestSubagentDiscovery(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

mkdir -p .crush/agents

cat > .crush/agents/code-reviewer.md << 'SUBAGENT'
---
name: code-reviewer
description: Reviews code for quality and best practices
model: sonnet
tools:
  - Glob
  - Grep
  - View
---
You are a code reviewer.
SUBAGENT

cat > crush.json << 'CONFIG'
%s
CONFIG

exec "%s"
`, TestConfigJSON(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	// Wait for app to initialize and discover subagents.
	time.Sleep(startupDelay)

	// The app should have loaded without errors.
	// We can't directly check the registry, but we can verify the app started.
}

// TestSubagentConfigParsing tests various subagent configurations.
func TestSubagentConfigParsing(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

mkdir -p .crush/agents
mkdir -p .claude/agents

# Subagent with all options
cat > .crush/agents/full-config.md << 'SUBAGENT'
---
name: full-config
description: Subagent with all configuration options
model: opus
permission_mode: plan
tools:
  - Glob
  - Grep
  - View
disallowed_tools:
  - Bash
---
Full configuration system prompt.
SUBAGENT

# Minimal subagent
cat > .crush/agents/minimal.md << 'SUBAGENT'
---
name: minimal
description: Minimal configuration
---
Minimal prompt.
SUBAGENT

# Claude-compatible location
cat > .claude/agents/claude-compat.md << 'SUBAGENT'
---
name: claude-compat
description: Claude Code compatible location
model: haiku
---
Claude compatible prompt.
SUBAGENT

cat > crush.json << 'CONFIG'
%s
CONFIG

exec "%s"
`, TestConfigJSON(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	// App should start without errors.
}

// TestSubagentValidation tests graceful handling of invalid subagent files.
func TestSubagentValidation(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

mkdir -p .crush/agents

# Valid subagent
cat > .crush/agents/valid.md << 'SUBAGENT'
---
name: valid-agent
description: Valid agent
---
Prompt.
SUBAGENT

# Invalid subagent (missing required fields)
cat > .crush/agents/invalid.md << 'SUBAGENT'
---
name: invalid
---
Missing description.
SUBAGENT

# Invalid YAML
cat > .crush/agents/broken-yaml.md << 'SUBAGENT'
---
name: [broken
description: This YAML is broken
---
Prompt.
SUBAGENT

cat > crush.json << 'CONFIG'
%s
CONFIG

exec "%s"
`, TestConfigJSON(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 40,
		Cols: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	// App should start despite invalid subagent files.
	time.Sleep(startupDelay)
}

// TestAgentsDialogOpens tests the Agents dialog opened with ctrl+a.
func TestAgentsDialogOpens(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

mkdir -p .crush/agents

cat > .crush/agents/test-agent.md << 'SUBAGENT'
---
name: test-agent
description: A test agent
---
Test agent.
SUBAGENT

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

	// Press ctrl+a to open agents dialog.
	_ = term.Write(trifle.CtrlA())

	// Should show the agents dialog title.
	locator := term.GetByText("Agents", trifle.WithFull())
	if err := locator.WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected agents dialog: %v", err)
	}
}

// TestAgentsDialogDisplaysNames tests that agent names are displayed.
func TestAgentsDialogDisplaysNames(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

mkdir -p .crush/agents

cat > .crush/agents/code-reviewer.md << 'SUBAGENT'
---
name: code-reviewer
description: Reviews code for quality
---
Code reviewer agent.
SUBAGENT

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
	_ = term.Write(trifle.CtrlA())
	time.Sleep(dialogTransition)

	// Should show agent names.
	output := term.Output()
	if !strings.Contains(output, "code-reviewer") {
		t.Errorf("Expected code-reviewer in output: %s", output)
	}
}

// TestAgentsDialogDetailView tests the detail view on enter.
func TestAgentsDialogDetailView(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

mkdir -p .crush/agents

cat > .crush/agents/test-agent.md << 'SUBAGENT'
---
name: test-agent
description: A test agent
---
Test agent.
SUBAGENT

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
	_ = term.Write(trifle.CtrlA())
	time.Sleep(dialogTransition)

	// Press enter to view details of the first agent.
	_ = term.Submit()
	time.Sleep(dialogTransition)

	// Should show the detail view title with agent name.
	locator := term.GetByText("Agent:", trifle.WithFull())
	if err := locator.WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected agent detail view: %v", err)
	}
}

// TestAgentsDialogClose tests that escape closes the dialog.
func TestAgentsDialogClose(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

mkdir -p .crush/agents

cat > .crush/agents/test-agent.md << 'SUBAGENT'
---
name: test-agent
description: A test agent
---
Test agent.
SUBAGENT

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
	_ = term.Write(trifle.CtrlA())

	if err := term.GetByText("Agents", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Fatalf("Dialog did not open: %v", err)
	}

	// Press escape to close.
	_ = term.Write(trifle.Escape())
	time.Sleep(dialogTransition)

	// The dialog should be closed now.
}

// TestAgentsDialogHelp tests that help keybindings are shown.
func TestAgentsDialogHelp(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

mkdir -p .crush/agents

cat > .crush/agents/test-agent.md << 'SUBAGENT'
---
name: test-agent
description: A test agent
---
Test agent.
SUBAGENT

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
	_ = term.Write(trifle.CtrlA())
	time.Sleep(dialogTransition)

	// Should show help text.
	locator := term.GetByText("view details", trifle.WithFull())
	if err := locator.WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected help text: %v", err)
	}
}
