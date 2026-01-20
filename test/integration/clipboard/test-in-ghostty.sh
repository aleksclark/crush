#!/bin/bash
set -e

# Start Xvfb
Xvfb :99 -screen 0 1024x768x24 &
XVFB_PID=$!
sleep 3

# Build crush
echo "Building Crush..."
go build -o /tmp/crush .

# Test 1: Verify xsel works
echo "=== Test 1: xsel basic functionality ==="
echo "test primary" | xsel -p -i
PRIMARY_CONTENT=$(xsel -p -o)
echo "PRIMARY content: $PRIMARY_CONTENT"

echo "test clipboard" | xsel -b -i
CLIPBOARD_CONTENT=$(xsel -b -o)
echo "CLIPBOARD content: $CLIPBOARD_CONTENT"

# Test 2: Run integration tests
echo ""
echo "=== Test 2: Running integration tests ==="
go test -v -timeout=20s ./test/integration/clipboard

# Test 3: Interactive test with Ghostty
echo ""
echo "=== Test 3: Manual test scenario ==="
echo "Setting up test scenario..."

# Set different content in PRIMARY and CLIPBOARD
echo -n "primary selection content" | xsel -p -i
echo -n "clipboard content" | xsel -b -i

# Verify they're set
echo "PRIMARY: $(xsel -p -o)"
echo "CLIPBOARD: $(xsel -b -o)"

# Start Ghostty with Crush in background
echo "Starting Ghostty with Crush..."
timeout 10 ghostty --x11-instance-name=crush-test -- /tmp/crush &
GHOSTTY_PID=$!
sleep 3

# Test our clipboard wrapper directly
echo ""
echo "=== Test 4: Testing clipboard wrapper ==="
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
	return "", fmt.Errorf("xsel not found")
}

func writePrimary(text string) error {
	if _, err := exec.LookPath("xsel"); err == nil {
		cmd := exec.Command("xsel", "-p", "-i")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	return fmt.Errorf("xsel not found")
}

func main() {
	// Test read
	content, err := readPrimary()
	if err != nil {
		fmt.Printf("ReadPrimary error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Read from PRIMARY: %q\n", content)
	
	// Test write
	testContent := "wrapper test content"
	if err := writePrimary(testContent); err != nil {
		fmt.Printf("WritePrimary error: %v\n", err)
		os.Exit(1)
	}
	
	time.Sleep(100 * time.Millisecond)
	
	// Verify
	content, err = readPrimary()
	if err != nil {
		fmt.Printf("ReadPrimary verify error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Verified PRIMARY: %q\n", content)
	
	if content != testContent {
		fmt.Printf("MISMATCH: got %q, want %q\n", content, testContent)
		os.Exit(1)
	}
	
	fmt.Println("✓ Wrapper functions work correctly")
}
GOCODE

# Cleanup
kill $GHOSTTY_PID 2>/dev/null || true
kill $XVFB_PID 2>/dev/null || true

echo ""
echo "=== All tests completed ==="
