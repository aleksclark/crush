package clipboard_test

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestXselWorks verifies xsel can read/write selections.
func TestXselWorks(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("No DISPLAY set")
	}

	// Test PRIMARY
	cmd := exec.Command("sh", "-c", "echo 'test primary' | xsel -p -i")
	if err := cmd.Run(); err != nil {
		t.Fatalf("xsel -p failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	out, err := exec.Command("xsel", "-p", "-o").Output()
	if err != nil {
		t.Fatalf("xsel -p -o failed: %v", err)
	}

	t.Logf("PRIMARY: %q", string(out))

	// Test CLIPBOARD
	cmd = exec.Command("sh", "-c", "echo 'test clipboard' | xsel -b -i")
	if err := cmd.Run(); err != nil {
		t.Fatalf("xsel -b failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	out, err = exec.Command("xsel", "-b", "-o").Output()
	if err != nil {
		t.Fatalf("xsel -b -o failed: %v", err)
	}

	t.Logf("CLIPBOARD: %q", string(out))
}
