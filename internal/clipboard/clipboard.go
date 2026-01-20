// Package clipboard provides clipboard operations with proper PRIMARY selection support.
package clipboard

import (
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
)

// ReadPrimary reads from the X11 PRIMARY selection (for middle-click paste).
// Falls back to regular clipboard on non-X11 systems.
func ReadPrimary() (string, error) {
	// On X11 systems, use xsel/xclip for PRIMARY selection
	if os.Getenv("DISPLAY") != "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		// Try xsel first
		if _, err := exec.LookPath("xsel"); err == nil {
			cmd := exec.Command("xsel", "-p", "-o")
			out, err := cmd.Output()
			if err == nil {
				return strings.TrimRight(string(out), "\n"), nil
			}
		}

		// Try xclip
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd := exec.Command("xclip", "-selection", "primary", "-o")
			out, err := cmd.Output()
			if err == nil {
				return strings.TrimRight(string(out), "\n"), nil
			}
		}
	}

	// Fall back to regular clipboard
	return clipboard.ReadAll()
}

// WritePrimary writes to the X11 PRIMARY selection (for middle-click paste).
// Also writes to regular clipboard on non-X11 systems.
func WritePrimary(text string) error {
	// On X11 systems, use xsel/xclip for PRIMARY selection
	if os.Getenv("DISPLAY") != "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		// Try xsel first
		if _, err := exec.LookPath("xsel"); err == nil {
			cmd := exec.Command("xsel", "-p", "-i")
			cmd.Stdin = strings.NewReader(text)
			if err := cmd.Run(); err == nil {
				return nil
			}
		}

		// Try xclip
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd := exec.Command("xclip", "-selection", "primary", "-i")
			cmd.Stdin = strings.NewReader(text)
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	}

	// Fall back to regular clipboard
	return clipboard.WriteAll(text)
}

// WriteClipboard writes to the system clipboard (Ctrl+C/Ctrl+V).
func WriteClipboard(text string) error {
	return clipboard.WriteAll(text)
}

// WriteBoth writes to both clipboard and PRIMARY selection.
// This is what should happen when text is selected/copied in the UI.
func WriteBoth(text string) error {
	// Write to clipboard first
	if err := WriteClipboard(text); err != nil {
		return err
	}

	// Try to write to PRIMARY (best effort, don't fail if it doesn't work)
	_ = WritePrimary(text)

	return nil
}
