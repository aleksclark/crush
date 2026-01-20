package clipboard_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/clipboard"
)

// TestReadPrimary verifies reading from PRIMARY selection.
func TestReadPrimary(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("No DISPLAY set")
	}

	testContent := "test primary read"

	// Set PRIMARY with xsel
	cmd := exec.Command("sh", "-c", "echo -n '"+testContent+"' | xsel -p -i")
	if err := cmd.Run(); err != nil {
		t.Fatalf("xsel failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Read via our wrapper
	content, err := clipboard.ReadPrimary()
	if err != nil {
		t.Fatalf("ReadPrimary failed: %v", err)
	}

	if content != testContent {
		t.Errorf("PRIMARY content mismatch: got %q, want %q", content, testContent)
	}
}

// TestWritePrimary verifies writing to PRIMARY selection.
func TestWritePrimary(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("No DISPLAY set")
	}

	testContent := "test primary write"

	// Write via our wrapper
	if err := clipboard.WritePrimary(testContent); err != nil {
		t.Fatalf("WritePrimary failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Read back with xsel
	out, err := exec.Command("xsel", "-p", "-o").Output()
	if err != nil {
		t.Fatalf("xsel -p -o failed: %v", err)
	}

	content := strings.TrimSpace(string(out))
	if content != testContent {
		t.Errorf("PRIMARY content mismatch: got %q, want %q", content, testContent)
	}
}

// TestWriteBoth verifies writing to both selections.
func TestWriteBoth(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("No DISPLAY set")
	}

	testContent := "test both selections"

	// Write via our wrapper
	if err := clipboard.WriteBoth(testContent); err != nil {
		t.Fatalf("WriteBoth failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Read CLIPBOARD with xsel -b
	out, err := exec.Command("xsel", "-b", "-o").Output()
	if err != nil {
		t.Fatalf("xsel -b -o failed: %v", err)
	}
	clipContent := strings.TrimSpace(string(out))

	// Read PRIMARY with xsel -p
	out, err = exec.Command("xsel", "-p", "-o").Output()
	if err != nil {
		t.Fatalf("xsel -p -o failed: %v", err)
	}
	primContent := strings.TrimSpace(string(out))

	if clipContent != testContent {
		t.Errorf("CLIPBOARD mismatch: got %q, want %q", clipContent, testContent)
	}

	if primContent != testContent {
		t.Errorf("PRIMARY mismatch: got %q, want %q", primContent, testContent)
	}
}

// TestPrimaryAndClipboardIndependent verifies they can have different content.
func TestPrimaryAndClipboardIndependent(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("No DISPLAY set")
	}

	clipContent := "clipboard content"
	primContent := "primary content"

	// Set CLIPBOARD
	if err := clipboard.WriteClipboard(clipContent); err != nil {
		t.Fatalf("WriteClipboard failed: %v", err)
	}

	// Set PRIMARY
	if err := clipboard.WritePrimary(primContent); err != nil {
		t.Fatalf("WritePrimary failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify CLIPBOARD with xsel -b
	out, err := exec.Command("xsel", "-b", "-o").Output()
	if err != nil {
		t.Fatalf("xsel -b -o failed: %v", err)
	}
	gotClip := strings.TrimSpace(string(out))

	// Verify PRIMARY with xsel -p
	out, err = exec.Command("xsel", "-p", "-o").Output()
	if err != nil {
		t.Fatalf("xsel -p -o failed: %v", err)
	}
	gotPrim := strings.TrimSpace(string(out))

	if gotClip != clipContent {
		t.Errorf("CLIPBOARD mismatch: got %q, want %q", gotClip, clipContent)
	}

	if gotPrim != primContent {
		t.Errorf("PRIMARY mismatch: got %q, want %q", gotPrim, primContent)
	}
}
