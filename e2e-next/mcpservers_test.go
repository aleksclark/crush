package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aleksclark/trifle"
)

// TestMCPServersDialogOpens tests that the MCP servers dialog opens with ctrl+e.
func TestMCPServersDialogOpens(t *testing.T) {
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

	// Press ctrl+e to open MCP servers dialog.
	_ = term.Write(trifle.CtrlE())
	time.Sleep(700 * time.Millisecond)

	// Should show the MCP servers dialog title.
	locator := term.GetByText("MCP Servers", trifle.WithFull())
	if err := locator.WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected MCP servers dialog: %v", err)
	}
}

// TestMCPServersDialogClose tests that escape closes the dialog.
func TestMCPServersDialogClose(t *testing.T) {
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
	_ = term.Write(trifle.CtrlE())
	time.Sleep(700 * time.Millisecond)

	if err := term.GetByText("MCP Servers", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Fatalf("Dialog did not open: %v", err)
	}

	// Press escape to close.
	_ = term.Write(trifle.Escape())
	time.Sleep(700 * time.Millisecond)
	// Dialog should be closed now.
}

// TestMCPServersDialogHelp tests that help keybindings are shown.
func TestMCPServersDialogHelp(t *testing.T) {
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
	_ = term.Write(trifle.CtrlE())
	time.Sleep(700 * time.Millisecond)

	// Should show help text with keybindings.
	if err := term.GetByText("restart", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected restart keybinding: %v", err)
	}
	if err := term.GetByText("logs", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected logs keybinding: %v", err)
	}
}

// TestMCPServersWithConfig tests the dialog with actual MCP server configurations.
func TestMCPServersWithConfig(t *testing.T) {
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
  "mcp": {
    "test-server": {
      "type": "stdio",
      "command": "echo",
      "args": ["test"],
      "disabled": true
    },
    "another-server": {
      "type": "http",
      "url": "http://localhost:9998",
      "disabled": true
    }
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
	_ = term.Write(trifle.CtrlE())
	time.Sleep(700 * time.Millisecond)

	// Should show configured servers.
	output := term.Output()
	if !strings.Contains(output, "test-server") {
		t.Errorf("Expected test-server in output: %s", output)
	}
	if !strings.Contains(output, "another-server") {
		t.Errorf("Expected another-server in output: %s", output)
	}
}

// TestMCPServersEmptyState tests the dialog when no MCP servers are configured.
func TestMCPServersEmptyState(t *testing.T) {
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
	_ = term.Write(trifle.CtrlE())
	time.Sleep(700 * time.Millisecond)

	// First verify the dialog opened.
	if err := term.GetByText("MCP Servers", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Fatalf("Dialog did not open: %v", err)
	}

	// Should show empty state message.
	if err := term.GetByText("No MCP servers configured", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected empty state message: %v", err)
	}
}

// TestMCPLogsDialog tests the logs dialog opened with 'l' key.
func TestMCPLogsDialog(t *testing.T) {
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
  "mcp": {
    "test-server": {
      "type": "stdio",
      "command": "echo",
      "args": ["test"],
      "disabled": true
    }
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

	// Open MCP servers dialog.
	_ = term.Write(trifle.CtrlE())
	time.Sleep(700 * time.Millisecond)

	// Verify servers dialog is open.
	if err := term.GetByText("MCP Servers", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Fatalf("Dialog did not open: %v", err)
	}

	// Press 'l' to open logs.
	_ = term.Write("l")
	time.Sleep(700 * time.Millisecond)

	// Should show the logs dialog title with server name.
	if err := term.GetByText("Logs:", trifle.WithFull()).WaitVisible(5 * time.Second); err != nil {
		t.Errorf("Expected logs dialog: %v", err)
	}
}
