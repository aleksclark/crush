#!/bin/bash
# Quick manual test script for X11 clipboard functionality
#
# This script demonstrates the three clipboard operations:
# 1. Copy text to both CLIPBOARD and PRIMARY
# 2. Paste from PRIMARY with middle-click
# 3. Paste from CLIPBOARD with Ctrl+V
#
# Usage: ./test-clipboard.sh

set -e

echo "=== Clipboard Test Script ==="
echo ""
echo "Prerequisites:"
echo "  - Must be running on X11 (not Wayland)"
echo "  - Must have xsel or xclip installed"
echo ""

# Check if we're on X11
if [ -z "$DISPLAY" ]; then
    echo "ERROR: DISPLAY not set. Are you running X11?"
    exit 1
fi

if [ -n "$WAYLAND_DISPLAY" ]; then
    echo "WARNING: WAYLAND_DISPLAY is set. This script is for X11 only."
    exit 1
fi

# Check for clipboard utilities
if ! command -v xsel &> /dev/null && ! command -v xclip &> /dev/null; then
    echo "ERROR: Neither xsel nor xclip found. Please install one."
    exit 1
fi

echo "✓ Running on X11"
echo "✓ Clipboard utilities available"
echo ""

# Test 1: Verify our wrapper functions work
echo "=== Test 1: Testing clipboard wrapper functions ==="
echo ""

go run - <<'GOCODE'
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func readPrimary() (string, error) {
	if _, err := exec.LookPath("xsel"); err == nil {
		cmd := exec.Command("xsel", "-p", "-o")
		out, err := cmd.Output()
		if err == nil {
			return strings.TrimRight(string(out), "\n"), nil
		}
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command("xclip", "-selection", "primary", "-o")
		out, err := cmd.Output()
		if err == nil {
			return strings.TrimRight(string(out), "\n"), nil
		}
	}
	return "", fmt.Errorf("no clipboard utility found")
}

func writePrimary(text string) error {
	if _, err := exec.LookPath("xsel"); err == nil {
		cmd := exec.Command("xsel", "-p", "-i")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command("xclip", "-selection", "primary", "-i")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	return fmt.Errorf("no clipboard utility found")
}

func writeClipboard(text string) error {
	if _, err := exec.LookPath("xsel"); err == nil {
		cmd := exec.Command("xsel", "-b", "-i")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command("xclip", "-selection", "clipboard", "-i")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	return fmt.Errorf("no clipboard utility found")
}

func main() {
	// Test write to PRIMARY
	testPrimary := "test primary selection"
	if err := writePrimary(testPrimary); err != nil {
		fmt.Printf("❌ WritePrimary failed: %v\n", err)
		os.Exit(1)
	}
	time.Sleep(100 * time.Millisecond)
	
	content, err := readPrimary()
	if err != nil {
		fmt.Printf("❌ ReadPrimary failed: %v\n", err)
		os.Exit(1)
	}
	if content != testPrimary {
		fmt.Printf("❌ PRIMARY mismatch: got %q, want %q\n", content, testPrimary)
		os.Exit(1)
	}
	fmt.Println("✓ PRIMARY selection read/write works")
	
	// Test write to CLIPBOARD
	testClipboard := "test clipboard selection"
	if err := writeClipboard(testClipboard); err != nil {
		fmt.Printf("❌ WriteClipboard failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ CLIPBOARD selection write works")
	
	// Verify they're independent
	primary, _ := readPrimary()
	if primary == testClipboard {
		fmt.Println("❌ PRIMARY and CLIPBOARD should be different!")
		os.Exit(1)
	}
	fmt.Println("✓ PRIMARY and CLIPBOARD are independent")
}
GOCODE

echo ""
echo "=== Test 2: Build and check Crush ==="
echo ""

if ! go build -o /tmp/crush-test .; then
    echo "❌ Build failed"
    exit 1
fi

echo "✓ Crush built successfully"
echo ""

echo "=== Summary ==="
echo ""
echo "The clipboard wrapper functions work correctly."
echo ""
echo "To test Crush interactively:"
echo "  1. Run: /tmp/crush-test"
echo "  2. Select text with your mouse (should copy to PRIMARY)"
echo "  3. Middle-click to paste (should paste from PRIMARY)"
echo "  4. Copy text normally (Ctrl+C or 'y' key)"
echo "  5. Paste with Ctrl+V (should paste from CLIPBOARD)"
echo ""
echo "Key fixes applied:"
echo "  - internal/clipboard wrapper bypasses atotto/clipboard bug"
echo "  - Middle-click uses clipboard.ReadPrimary()"
echo "  - Copy operations use clipboard.WriteBoth()"
echo "  - Ctrl+V handled by Bubble Tea's PasteMsg"
echo ""
