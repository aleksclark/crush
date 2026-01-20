package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aleksclark/trifle"
)

// TestAgentPermissionModes tests various permission mode configurations.
func TestAgentPermissionModes(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

mkdir -p .crush/agents

# Default permission mode (no mode specified)
cat > .crush/agents/default-mode.md << 'SUBAGENT'
---
name: default-mode
description: Agent with default permission mode
---
Default mode agent.
SUBAGENT

# Yolo mode using the alias
cat > .crush/agents/yolo-mode.md << 'SUBAGENT'
---
name: yolo-mode
description: Agent with yolo permission mode
permission_mode: yolo
---
Yolo mode agent.
SUBAGENT

# bypassPermissions mode (canonical name)
cat > .crush/agents/bypass-mode.md << 'SUBAGENT'
---
name: bypass-mode
description: Agent with bypassPermissions mode
permission_mode: bypassPermissions
---
Bypass mode agent.
SUBAGENT

# acceptEdits mode
cat > .crush/agents/accept-edits-mode.md << 'SUBAGENT'
---
name: accept-edits-mode
description: Agent with acceptEdits mode
permission_mode: acceptEdits
---
Accept edits mode agent.
SUBAGENT

# dontAsk mode
cat > .crush/agents/dont-ask-mode.md << 'SUBAGENT'
---
name: dont-ask-mode
description: Agent with dontAsk mode
permission_mode: dontAsk
---
Dont ask mode agent.
SUBAGENT

# plan mode
cat > .crush/agents/plan-mode.md << 'SUBAGENT'
---
name: plan-mode
description: Agent with plan mode
permission_mode: plan
---
Plan mode agent.
SUBAGENT

cat > crush.json << 'CONFIG'
%s
CONFIG

exec "%s"
`, IsolationScript(), TestConfigJSON(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 50,
		Cols: 120,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)

	// Open agents dialog.
	_ = term.Write(trifle.CtrlA())
	time.Sleep(dialogTransition)

	// Should show 6 agents.
	if err := term.GetByText("Agents (6)", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected 6 agents: %v", err)
	}
}

// TestAgentYoloMode tests yolo mode agent details.
func TestAgentYoloMode(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

mkdir -p .crush/agents

cat > .crush/agents/yolo-mode.md << 'SUBAGENT'
---
name: yolo-mode
description: Agent with yolo permission mode
permission_mode: yolo
---
Yolo mode agent.
SUBAGENT

cat > crush.json << 'CONFIG'
%s
CONFIG

exec "%s"
`, IsolationScript(), TestConfigJSON(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 50,
		Cols: 120,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlA())
	time.Sleep(dialogTransition)

	// Filter to yolo-mode agent.
	_ = term.Write("yolo-mode")
	time.Sleep(300 * time.Millisecond)

	// View details.
	_ = term.Submit()
	time.Sleep(dialogTransition)

	output := term.Output()
	if !strings.Contains(output, "Agent: yolo-mode") {
		t.Errorf("Expected yolo-mode agent in output: %s", output)
	}
	if !strings.Contains(output, "yolo") {
		t.Errorf("Expected yolo permission mode in output: %s", output)
	}
}

// TestAgentModelTypes tests model type configurations.
func TestAgentModelTypes(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

mkdir -p .crush/agents

# Sonnet model
cat > .crush/agents/sonnet-agent.md << 'SUBAGENT'
---
name: sonnet-agent
description: Uses sonnet model
model: sonnet
---
Sonnet agent prompt.
SUBAGENT

# Haiku model
cat > .crush/agents/haiku-agent.md << 'SUBAGENT'
---
name: haiku-agent
description: Uses haiku model
model: haiku
---
Haiku agent prompt.
SUBAGENT

# Opus model
cat > .crush/agents/opus-agent.md << 'SUBAGENT'
---
name: opus-agent
description: Uses opus model
model: opus
---
Opus agent prompt.
SUBAGENT

# Inherit model
cat > .crush/agents/inherit-agent.md << 'SUBAGENT'
---
name: inherit-agent
description: Inherits parent model
model: inherit
---
Inherit agent prompt.
SUBAGENT

cat > crush.json << 'CONFIG'
%s
CONFIG

exec "%s"
`, IsolationScript(), TestConfigJSON(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 50,
		Cols: 120,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlA())
	time.Sleep(dialogTransition)

	// Filter to sonnet-agent.
	_ = term.Write("sonnet-agent")
	time.Sleep(300 * time.Millisecond)

	_ = term.Submit()
	time.Sleep(dialogTransition)

	output := term.Output()
	if !strings.Contains(output, "sonnet") {
		t.Errorf("Expected sonnet model in output: %s", output)
	}
}

// TestAgentToolsConfig tests tools configuration display.
func TestAgentToolsConfig(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

mkdir -p .crush/agents

# Agent with specific tools
cat > .crush/agents/tools-agent.md << 'SUBAGENT'
---
name: tools-agent
description: Agent with specific tools
tools:
  - bash
  - edit
  - view
  - glob
  - grep
---
Tools agent prompt.
SUBAGENT

# Agent with disallowed tools
cat > .crush/agents/disallowed-agent.md << 'SUBAGENT'
---
name: disallowed-agent
description: Agent with disallowed tools
permission_mode: yolo
tools:
  - bash
  - edit
  - view
disallowed_tools:
  - download
  - fetch
---
Disallowed tools agent prompt.
SUBAGENT

cat > crush.json << 'CONFIG'
%s
CONFIG

exec "%s"
`, IsolationScript(), TestConfigJSON(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 50,
		Cols: 120,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)
	_ = term.Write(trifle.CtrlA())
	time.Sleep(dialogTransition)

	// Filter to tools-agent.
	_ = term.Write("tools-agent")
	time.Sleep(300 * time.Millisecond)

	_ = term.Submit()
	time.Sleep(dialogTransition)

	output := term.Output()
	if !strings.Contains(output, "Effective Tools") {
		t.Errorf("Expected Effective Tools section: %s", output)
	}
	if !strings.Contains(output, "bash") {
		t.Errorf("Expected bash in tools: %s", output)
	}
}

// TestAgentDialogNavigation tests dialog navigation functionality.
func TestAgentDialogNavigation(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

mkdir -p .crush/agents

cat > .crush/agents/agent-a.md << 'SUBAGENT'
---
name: agent-a
description: First agent
permission_mode: yolo
---
Agent A prompt.
SUBAGENT

cat > .crush/agents/agent-b.md << 'SUBAGENT'
---
name: agent-b
description: Second agent
permission_mode: plan
---
Agent B prompt.
SUBAGENT

cat > crush.json << 'CONFIG'
%s
CONFIG

exec "%s"
`, IsolationScript(), TestConfigJSON(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 50,
		Cols: 120,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)

	t.Run("can filter agents by typing", func(t *testing.T) {
		_ = term.Write(trifle.CtrlA())
		time.Sleep(dialogTransition)

		// Type to filter.
		_ = term.Write("agent-a")
		time.Sleep(300 * time.Millisecond)

		output := term.Output()
		if !strings.Contains(output, "agent-a") {
			t.Errorf("Expected agent-a in filtered output: %s", output)
		}
	})

	t.Run("can navigate with arrow keys", func(t *testing.T) {
		// Clear filter.
		for i := 0; i < 7; i++ {
			_ = term.KeyBackspace()
		}
		time.Sleep(200 * time.Millisecond)

		_ = term.KeyDown()
		time.Sleep(200 * time.Millisecond)

		_ = term.KeyUp()
		time.Sleep(200 * time.Millisecond)

		// Dialog should still be open.
		if err := term.GetByText("Agents", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
			t.Errorf("Dialog closed unexpectedly: %v", err)
		}
	})
}

// TestInvalidPermissionMode tests graceful handling of invalid permission modes.
func TestInvalidPermissionMode(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

mkdir -p .crush/agents

# Agent with invalid permission mode
cat > .crush/agents/invalid-mode.md << 'SUBAGENT'
---
name: invalid-mode
description: Agent with invalid permission mode
permission_mode: invalid_mode_value
---
Invalid mode agent.
SUBAGENT

# Valid agent for comparison
cat > .crush/agents/valid-agent.md << 'SUBAGENT'
---
name: valid-agent
description: Valid agent
permission_mode: yolo
---
Valid agent.
SUBAGENT

cat > crush.json << 'CONFIG'
%s
CONFIG

exec "%s"
`, IsolationScript(), TestConfigJSON(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 50,
		Cols: 120,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	// App should start despite invalid permission mode.
	time.Sleep(startupDelay)

	_ = term.Write(trifle.CtrlA())
	time.Sleep(dialogTransition)

	// Should show at least the valid agent.
	if err := term.GetByText("Agents", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected agents dialog: %v", err)
	}

	output := term.Output()
	if !strings.Contains(output, "valid-agent") {
		t.Errorf("Expected valid-agent to be loaded: %s", output)
	}
}
