package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aleksclark/trifle"
)

const clipboardTestDelay = 2 * time.Second

// TestClipboardCtrlVDoesNotCrash tests that Ctrl+V doesn't crash the app.
// This is a smoke test - actual clipboard content testing requires X11/Wayland.
func TestClipboardCtrlVDoesNotCrash(t *testing.T) {
	trifle.SkipOnWindows(t)

	// Create temp dir for isolated test.
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config", "crush")
	if err := os.MkdirAll(configPath, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Write test config.
	configFile := filepath.Join(configPath, "crush.json")
	if err := os.WriteFile(configFile, []byte(TestConfigJSON()), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	term, err := trifle.NewTerminal(CrushBinary(), []string{}, trifle.TerminalOptions{
		Rows: 30,
		Cols: 100,
		Env: []string{
			"XDG_CONFIG_HOME=" + filepath.Join(tmpDir, "config"),
			"XDG_DATA_HOME=" + filepath.Join(tmpDir, "data"),
			"HOME=" + tmpDir,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	// Wait for TUI to initialize.
	time.Sleep(clipboardTestDelay)

	// Type some text first.
	if err := term.Write("Hello world"); err != nil {
		t.Fatalf("Failed to type: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Press Ctrl+V (paste).
	// Note: Without actual clipboard content, this won't paste anything,
	// but it verifies the keybinding works and doesn't crash.
	if err := term.Write("\x16"); err != nil { // Ctrl+V
		t.Fatalf("Failed to send Ctrl+V: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Verify app is still running (didn't crash).
	// We just check that the terminal is responsive.
	if err := term.Write("x"); err != nil {
		t.Errorf("App not responsive after Ctrl+V: %v", err)
	}
}

// TestClipboardCopyKeybinding tests that 'c' key for copy doesn't crash.
func TestClipboardCopyKeybinding(t *testing.T) {
	trifle.SkipOnWindows(t)

	// Create temp dir for isolated test.
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config", "crush")
	if err := os.MkdirAll(configPath, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Write test config.
	configFile := filepath.Join(configPath, "crush.json")
	if err := os.WriteFile(configFile, []byte(TestConfigJSON()), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	term, err := trifle.NewTerminal(CrushBinary(), []string{}, trifle.TerminalOptions{
		Rows: 30,
		Cols: 100,
		Env: []string{
			"XDG_CONFIG_HOME=" + filepath.Join(tmpDir, "config"),
			"XDG_DATA_HOME=" + filepath.Join(tmpDir, "data"),
			"HOME=" + tmpDir,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	// Wait for TUI.
	time.Sleep(clipboardTestDelay)

	// Press 'c' (copy key in selection mode).
	// Without a selection, this should be a no-op.
	if err := term.Write("c"); err != nil {
		t.Fatalf("Failed to send 'c': %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Verify app is still responsive after copy attempt.
	if err := term.Write("test"); err != nil {
		t.Errorf("App not responsive after copy attempt: %v", err)
	}
}

// TestClipboardMultilineInput tests multi-line input with Ctrl+J (newline).
func TestClipboardMultilineInput(t *testing.T) {
	trifle.SkipOnWindows(t)

	// Create temp dir for isolated test.
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config", "crush")
	if err := os.MkdirAll(configPath, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Write test config.
	configFile := filepath.Join(configPath, "crush.json")
	if err := os.WriteFile(configFile, []byte(TestConfigJSON()), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	term, err := trifle.NewTerminal(CrushBinary(), []string{}, trifle.TerminalOptions{
		Rows: 30,
		Cols: 100,
		Env: []string{
			"XDG_CONFIG_HOME=" + filepath.Join(tmpDir, "config"),
			"XDG_DATA_HOME=" + filepath.Join(tmpDir, "data"),
			"HOME=" + tmpDir,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}
	defer term.Close()

	// Wait for TUI.
	time.Sleep(clipboardTestDelay)

	// Type multiple lines with Ctrl+J (newline without sending).
	if err := term.Write("Line 1"); err != nil {
		t.Fatalf("Failed to type: %v", err)
	}
	if err := term.Write("\n"); err != nil { // Ctrl+J / newline
		t.Fatalf("Failed to send Ctrl+J: %v", err)
	}
	if err := term.Write("Line 2"); err != nil {
		t.Fatalf("Failed to type: %v", err)
	}
	if err := term.Write("\n"); err != nil { // Ctrl+J / newline
		t.Fatalf("Failed to send Ctrl+J: %v", err)
	}
	if err := term.Write("Line 3"); err != nil {
		t.Fatalf("Failed to type: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Just verify we can still type (app didn't crash).
	if err := term.Write("test"); err != nil {
		t.Errorf("App not responsive after multiline input: %v", err)
	}
}
