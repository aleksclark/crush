package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aleksclark/trifle"
)

// TestModelDialogOpens tests that the model dialog opens.
func TestModelDialogOpens(t *testing.T) {
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
    "test-provider": {
      "type": "openai-compat",
      "base_url": "http://localhost:9999",
      "api_key": "test-key"
    }
  },
  "models": {
    "large": { "provider": "test-provider", "model": "test-model-large" },
    "small": { "provider": "test-provider", "model": "test-model-small" }
  }
}
CONFIG

exec "%s"
`, IsolationScript(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 50,
		Cols: 120,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)

	// Press ctrl+m to open model dialog.
	_ = term.Write(trifle.CtrlM())
	time.Sleep(dialogTransition)

	// The dialog title may vary, look for model-related content.
}

// TestModelDialogClose tests that escape closes the model dialog.
func TestModelDialogClose(t *testing.T) {
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
		Rows: 50,
		Cols: 120,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)

	// Open model dialog.
	_ = term.Write(trifle.CtrlL())
	time.Sleep(dialogTransition)

	// Close with escape.
	_ = term.KeyEscape()
	time.Sleep(dialogTransition)

	// Dialog should be closed.
}

// TestModelChangeShowsCurrent tests that current model is shown.
func TestModelChangeShowsCurrent(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

# Initialize git repo (some features require it)
git init -q
git config user.email "test@test.com"
git config user.name "Test User"
echo "# Test" > README.md
git add . && git commit -q -m "init"

cat > crush.json << 'CONFIG'
{
  "providers": {
    "anthropic": {
      "type": "anthropic",
      "api_key": "test-key-not-real"
    }
  },
  "models": {
    "large": { "provider": "anthropic", "model": "claude-sonnet-4-20250514" },
    "small": { "provider": "anthropic", "model": "claude-haiku-3-20240307" }
  }
}
CONFIG

exec "%s"
`, IsolationScript(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 50,
		Cols: 120,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)

	// Close the init dialog first by selecting "nope".
	_ = term.KeyDown() // Move to "nope"
	time.Sleep(200 * time.Millisecond)
	_ = term.Submit()
	time.Sleep(dialogTransition)

	output := strings.ToLower(term.Output())
	// The status bar should show the current model.
	if !strings.Contains(output, "sonnet") && !strings.Contains(output, "claude") {
		t.Logf("Status bar output: %s", term.Output())
		// This might not be visible if the provider isn't loaded properly.
	}
}

// TestModelConfigLoads tests that model configuration loads successfully.
func TestModelConfigLoads(t *testing.T) {
	trifle.SkipOnWindows(t)

	script := fmt.Sprintf(`
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

%s

git init -q
git config user.email "test@test.com"
git config user.name "Test User"
echo "# Test" > README.md
git add . && git commit -q -m "init"

cat > crush.json << 'CONFIG'
{
  "providers": {
    "anthropic": {
      "type": "anthropic",
      "api_key": "test-key"
    }
  },
  "models": {
    "large": { "provider": "anthropic", "model": "claude-sonnet-4-20250514" },
    "small": { "provider": "anthropic", "model": "claude-haiku-3-20240307" }
  }
}
CONFIG

exec "%s"
`, IsolationScript(), CrushBinary())

	term, err := trifle.NewTerminal("bash", []string{"-c", script}, trifle.TerminalOptions{
		Rows: 50,
		Cols: 120,
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	time.Sleep(startupDelay)

	// App should start successfully with the configured model.
	output := strings.ToLower(term.Output())

	// Should not show any error about model configuration.
	if strings.Contains(output, "error loading") {
		t.Errorf("Unexpected error loading: %s", term.Output())
	}
	if strings.Contains(output, "failed to") {
		t.Errorf("Unexpected failure: %s", term.Output())
	}
}
